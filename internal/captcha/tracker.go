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

package captcha

import (
	"sync"
	"time"
)

type failureEntry struct {
	count   int
	expires time.Time
}

// FailureTracker tracks login failures by IP and email in memory.
// It triggers captcha requirement after a configurable threshold.
type FailureTracker struct {
	mu        sync.Mutex
	threshold int
	window    time.Duration
	entries   map[string]*failureEntry
	stopCh    chan struct{}
}

// NewFailureTracker creates a new FailureTracker with the given threshold and window.
// threshold is the number of failures before captcha is required.
// window is how long failures are remembered after the last failure.
func NewFailureTracker(threshold int, window time.Duration) *FailureTracker {
	t := &FailureTracker{
		threshold: threshold,
		window:    window,
		entries:   make(map[string]*failureEntry),
		stopCh:    make(chan struct{}),
	}

	go t.cleanup()

	return t
}

// RecordFailure records a login failure for the given IP and/or email.
func (t *FailureTracker) RecordFailure(ip, email string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	expires := time.Now().Add(t.window)

	for _, key := range keysFor(ip, email) {
		if e, ok := t.entries[key]; ok {
			e.count++
			e.expires = expires
		} else {
			t.entries[key] = &failureEntry{count: 1, expires: expires}
		}
	}
}

// RecordSuccess clears failure counts for the given IP and/or email.
func (t *FailureTracker) RecordSuccess(ip, email string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, key := range keysFor(ip, email) {
		delete(t.entries, key)
	}
}

// RequiresCaptcha returns true if the number of failures for the IP or email
// has reached or exceeded the threshold.
func (t *FailureTracker) RequiresCaptcha(ip, email string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for _, key := range keysFor(ip, email) {
		if e, ok := t.entries[key]; ok {
			if e.expires.After(now) && e.count >= t.threshold {
				return true
			}
		}
	}

	return false
}

// Close stops the background cleanup goroutine.
func (t *FailureTracker) Close() {
	close(t.stopCh)
}

func (t *FailureTracker) cleanup() {
	ticker := time.NewTicker(t.window)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.mu.Lock()
			now := time.Now()
			for key, e := range t.entries {
				if e.expires.Before(now) {
					delete(t.entries, key)
				}
			}
			t.mu.Unlock()
		case <-t.stopCh:
			return
		}
	}
}

func keysFor(ip, email string) []string {
	var keys []string
	if ip != "" {
		keys = append(keys, "ip:"+ip)
	}
	if email != "" {
		keys = append(keys, "email:"+email)
	}
	return keys
}
