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

// Package periodic runs a task in the background at a fixed interval.
package periodic

import (
	"context"
	"sync"
	"time"
)

// Runner runs Task every Interval between Start and Stop. Its fields are read
// when it starts: set them before calling Start.
type Runner struct {
	// Interval between two runs of Task.
	Interval time.Duration
	// RunAtStart runs Task as soon as the Runner starts, rather than after
	// a first Interval.
	RunAtStart bool
	// Task is the work to run. It receives a context cancelled when the
	// Runner stops, and is never run concurrently with itself.
	Task func(ctx context.Context)

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// Start launches the background loop. Calling Start while the Runner is
// already running is a no-op.
func (r *Runner) Start(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done != nil {
		return
	}

	ctx, r.cancel = context.WithCancel(ctx)
	r.done = make(chan struct{})

	go r.loop(ctx, r.done, r.Interval, r.RunAtStart, r.Task)
}

// Stop halts the background loop, if any, and waits for the running Task to
// return.
func (r *Runner) Stop() {
	r.mu.Lock()
	cancel, done := r.cancel, r.done
	r.cancel, r.done = nil, nil
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (r *Runner) loop(ctx context.Context, done chan struct{}, interval time.Duration, runAtStart bool, task func(context.Context)) {
	defer close(done)

	if runAtStart {
		task(ctx)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			task(ctx)
		}
	}
}
