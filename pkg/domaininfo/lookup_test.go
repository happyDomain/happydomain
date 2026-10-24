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

package domaininfo

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.happydns.org/happyDomain/pkg/domaininfo/types"
)

func TestLookupMany(t *testing.T) {
	var mu sync.Mutex
	calls := map[string]int{}
	var running, maxRunning atomic.Int32

	get := func(ctx context.Context, name string) (*DomainInfo, error) {
		n := running.Add(1)
		defer running.Add(-1)
		for {
			m := maxRunning.Load()
			if n <= m || maxRunning.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)

		mu.Lock()
		calls[name]++
		mu.Unlock()

		if name == "free.example" {
			return nil, types.ErrDomainDoesNotExist
		}
		return &DomainInfo{Name: name}, nil
	}

	names := []string{"a.example", "b.example", "a.example", "free.example", "c.example", "d.example"}
	res := LookupMany(context.Background(), get, names, 2)

	if len(res) != 5 {
		t.Fatalf("expected 5 results, got %d", len(res))
	}
	for name, n := range calls {
		if n != 1 {
			t.Errorf("%s looked up %d times", name, n)
		}
	}
	if maxRunning.Load() > 2 {
		t.Errorf("%d lookups ran at once, want at most 2", maxRunning.Load())
	}
	if r := res["a.example"]; r.Err != nil || r.Info == nil || r.Info.Name != "a.example" {
		t.Errorf("a.example = %+v", r)
	}
	if r := res["free.example"]; !errors.Is(r.Err, types.ErrDomainDoesNotExist) {
		t.Errorf("free.example = %+v", r)
	}
}

func TestLookupMany_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	get := func(ctx context.Context, name string) (*DomainInfo, error) {
		return &DomainInfo{Name: name}, nil
	}

	if res := LookupMany(ctx, get, []string{"a.example", "b.example", "c.example"}, 1); len(res) != 0 {
		t.Errorf("expected no lookup once cancelled, got %d", len(res))
	}
}

func TestLookupMany_CancelledMidway(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	get := func(ctx context.Context, name string) (*DomainInfo, error) {
		cancel()
		// Hold the only slot past the cancellation.
		time.Sleep(20 * time.Millisecond)
		return &DomainInfo{Name: name}, nil
	}

	res := LookupMany(ctx, get, []string{"a.example", "b.example", "c.example"}, 1)
	if len(res) != 1 || res["a.example"].Info == nil {
		t.Errorf("expected only the lookup started before the cancellation, got %v", res)
	}
}

func TestLookupMany_NoConcurrency(t *testing.T) {
	get := func(ctx context.Context, name string) (*DomainInfo, error) {
		return &DomainInfo{Name: name}, nil
	}

	if res := LookupMany(context.Background(), get, []string{"a.example", "b.example"}, 0); len(res) != 2 {
		t.Errorf("expected 2 results with a concurrency of 0, got %d", len(res))
	}
}

func TestLookupMany_NoName(t *testing.T) {
	get := func(ctx context.Context, name string) (*DomainInfo, error) {
		t.Errorf("unexpected lookup of %s", name)
		return nil, nil
	}

	if res := LookupMany(context.Background(), get, nil, 2); res == nil || len(res) != 0 {
		t.Errorf("expected an empty result, got %#v", res)
	}
}
