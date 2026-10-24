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

package periodic

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunner_RunsAtStartAndOnTicks(t *testing.T) {
	var runs atomic.Int32
	r := &Runner{
		Interval:   5 * time.Millisecond,
		RunAtStart: true,
		Task:       func(context.Context) { runs.Add(1) },
	}

	r.Start(context.Background())
	deadline := time.Now().Add(2 * time.Second)
	for runs.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	r.Stop()

	if runs.Load() < 3 {
		t.Fatalf("expected at least 3 runs, got %d", runs.Load())
	}

	// No run after Stop.
	after := runs.Load()
	time.Sleep(20 * time.Millisecond)
	if runs.Load() != after {
		t.Errorf("task ran after Stop")
	}
}

func TestRunner_NoRunAtStart(t *testing.T) {
	var runs atomic.Int32
	r := &Runner{
		Interval: time.Hour,
		Task:     func(context.Context) { runs.Add(1) },
	}

	r.Start(context.Background())
	time.Sleep(10 * time.Millisecond)
	r.Stop()

	if runs.Load() != 0 {
		t.Errorf("expected no run before the first interval, got %d", runs.Load())
	}
}

func TestRunner_StopCancelsTask(t *testing.T) {
	started := make(chan struct{})
	r := &Runner{
		Interval:   time.Hour,
		RunAtStart: true,
		Task: func(ctx context.Context) {
			close(started)
			<-ctx.Done()
		},
	}

	r.Start(context.Background())
	<-started
	r.Stop() // must not hang: the task context is cancelled
}

func TestRunner_StartIsIdempotentAndRestartable(t *testing.T) {
	var runs atomic.Int32
	r := &Runner{
		Interval:   time.Hour,
		RunAtStart: true,
		Task:       func(context.Context) { runs.Add(1) },
	}

	r.Stop() // before Start: no-op
	r.Start(context.Background())
	r.Start(context.Background()) // no second loop
	r.Stop()
	r.Stop()
	r.Start(context.Background()) // restart
	r.Stop()

	if runs.Load() != 2 {
		t.Errorf("expected 2 runs (one per actual start), got %d", runs.Load())
	}
}
