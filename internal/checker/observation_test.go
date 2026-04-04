// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package checker

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.happydns.org/happyDomain/model"
)

// blockingProvider is an ObservationProvider whose Collect blocks on the
// release channel until the test signals it. It records how many concurrent
// Collect calls are in flight at any moment.
type blockingProvider struct {
	key     happydns.ObservationKey
	release chan struct{}
	calls   int32
}

func (b *blockingProvider) Key() happydns.ObservationKey { return b.key }

func (b *blockingProvider) Collect(ctx context.Context, _ happydns.CheckerOptions) (any, error) {
	atomic.AddInt32(&b.calls, 1)
	defer atomic.AddInt32(&b.calls, -1)
	select {
	case <-b.release:
		return map[string]string{string(b.key): "ok"}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// TestObservationContext_ConcurrentDifferentKeys verifies that two Get calls
// for distinct observation keys can run their Collect concurrently, i.e.
// the per-context lock is not held across provider.Collect.
func TestObservationContext_ConcurrentDifferentKeys(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	pa := &blockingProvider{key: happydns.ObservationKey("test-a"), release: release}
	pb := &blockingProvider{key: happydns.ObservationKey("test-b"), release: release}

	oc := NewObservationContext(happydns.CheckTarget{}, happydns.CheckerOptions{}, nil, 0)
	oc.SetProviderOverride(pa.key, pa)
	oc.SetProviderOverride(pb.key, pb)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	results := make([]error, 2)
	for i, key := range []happydns.ObservationKey{pa.key, pb.key} {
		wg.Add(1)
		go func(idx int, k happydns.ObservationKey) {
			defer wg.Done()
			var dst map[string]string
			results[idx] = oc.Get(ctx, k, &dst)
		}(i, key)
	}

	// Wait until both providers are blocked inside Collect simultaneously.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&pa.calls) == 1 && atomic.LoadInt32(&pb.calls) == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if a, b := atomic.LoadInt32(&pa.calls), atomic.LoadInt32(&pb.calls); a != 1 || b != 1 {
		t.Fatalf("expected both providers to be collecting in parallel, got a=%d b=%d", a, b)
	}

	// Release both Collects and wait for the Get calls to return.
	release <- struct{}{}
	release <- struct{}{}
	wg.Wait()

	for i, err := range results {
		if err != nil {
			t.Errorf("Get %d returned error: %v", i, err)
		}
	}
}

// TestObservationContext_DedupesSameKey verifies that concurrent Get calls
// for the *same* key only invoke provider.Collect once.
func TestObservationContext_DedupesSameKey(t *testing.T) {
	release := make(chan struct{})

	var collectCount int32
	prov := &countingProvider{
		key:     happydns.ObservationKey("test-dedup"),
		release: release,
		count:   &collectCount,
	}

	oc := NewObservationContext(happydns.CheckTarget{}, happydns.CheckerOptions{}, nil, 0)
	oc.SetProviderOverride(prov.key, prov)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const N = 8
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			var dst map[string]string
			if err := oc.Get(ctx, prov.key, &dst); err != nil {
				t.Errorf("Get error: %v", err)
			}
		}()
	}

	// Wait for at least one collect to be in flight, then release it.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && atomic.LoadInt32(&collectCount) == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	close(release)
	wg.Wait()

	if got := atomic.LoadInt32(&collectCount); got != 1 {
		t.Errorf("expected exactly 1 Collect call, got %d", got)
	}
}

type countingProvider struct {
	key     happydns.ObservationKey
	release chan struct{}
	count   *int32
}

func (c *countingProvider) Key() happydns.ObservationKey { return c.key }

func (c *countingProvider) Collect(ctx context.Context, _ happydns.CheckerOptions) (any, error) {
	atomic.AddInt32(c.count, 1)
	select {
	case <-c.release:
		return map[string]string{"k": "v"}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
