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

package adminauth

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// newTestThrottle returns a throttle whose clock is driven by the returned
// pointer, so that lockouts and refills can be exercised without sleeping.
func newTestThrottle() (*LoginThrottle, *time.Time) {
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t := NewLoginThrottle()
	t.now = func() time.Time { return clock }
	t.lastRefill = clock
	return t, &clock
}

func TestLoginThrottleLocksOutAfterRepeatedFailures(t *testing.T) {
	th, _ := newTestThrottle()

	for i := range maxClientFailures - 1 {
		if _, ok := th.Allow("192.0.2.1"); !ok {
			t.Fatalf("attempt %d denied, want allowed", i)
		}
		if lockedFor := th.RecordFailure("192.0.2.1"); lockedFor != 0 {
			t.Fatalf("attempt %d locked out early (%s)", i, lockedFor)
		}
	}

	if _, ok := th.Allow("192.0.2.1"); !ok {
		t.Fatal("last allowed attempt denied")
	}
	if lockedFor := th.RecordFailure("192.0.2.1"); lockedFor != baseClientLockout {
		t.Errorf("lockout = %s, want %s", lockedFor, baseClientLockout)
	}

	retryAfter, ok := th.Allow("192.0.2.1")
	if ok {
		t.Fatal("attempt allowed while locked out")
	}
	if retryAfter != baseClientLockout {
		t.Errorf("retryAfter = %s, want %s", retryAfter, baseClientLockout)
	}

	// Another client must not be affected by that lockout.
	if _, ok := th.Allow("192.0.2.2"); !ok {
		t.Error("unrelated client denied")
	}
}

func TestLoginThrottleLockoutExpiresAndGrows(t *testing.T) {
	th, clock := newTestThrottle()

	for range maxClientFailures {
		th.RecordFailure("192.0.2.1")
	}

	*clock = clock.Add(baseClientLockout)
	if _, ok := th.Allow("192.0.2.1"); !ok {
		t.Fatal("attempt denied after the lockout expired")
	}

	if lockedFor := th.RecordFailure("192.0.2.1"); lockedFor != 2*baseClientLockout {
		t.Errorf("second lockout = %s, want %s", lockedFor, 2*baseClientLockout)
	}
}

func TestLoginThrottleLockoutIsCapped(t *testing.T) {
	th, _ := newTestThrottle()

	var lockedFor time.Duration
	for range 100 {
		lockedFor = th.RecordFailure("192.0.2.1")
	}

	if lockedFor != maxClientLockout {
		t.Errorf("lockout = %s, want %s", lockedFor, maxClientLockout)
	}
}

func TestLoginThrottleForgetsOldFailures(t *testing.T) {
	th, clock := newTestThrottle()

	for range maxClientFailures - 1 {
		th.RecordFailure("192.0.2.1")
	}

	*clock = clock.Add(clientFailureWindow + time.Second)

	if lockedFor := th.RecordFailure("192.0.2.1"); lockedFor != 0 {
		t.Errorf("lockout = %s, want none: failures older than the window must be forgotten", lockedFor)
	}
}

func TestLoginThrottleSuccessClearsHistory(t *testing.T) {
	th, _ := newTestThrottle()

	for range maxClientFailures - 1 {
		th.RecordFailure("192.0.2.1")
	}
	th.RecordSuccess("192.0.2.1")

	if lockedFor := th.RecordFailure("192.0.2.1"); lockedFor != 0 {
		t.Errorf("lockout = %s, want none after a successful login", lockedFor)
	}
}

func TestLoginThrottleGlobalRateLimit(t *testing.T) {
	th, clock := newTestThrottle()

	// Rotating the source address defeats the per-client lockout, so the global
	// bucket has to run out. Successes are not counted, hence the refund check
	// below.
	for i := range globalBurst {
		if _, ok := th.Allow(rotatingClient(i)); !ok {
			t.Fatalf("attempt %d denied before the burst was consumed", i)
		}
	}

	retryAfter, ok := th.Allow(rotatingClient(globalBurst))
	if ok {
		t.Fatal("attempt allowed after the global burst was consumed")
	}
	if retryAfter <= 0 {
		t.Errorf("retryAfter = %s, want a positive delay", retryAfter)
	}

	*clock = clock.Add(globalRefill)
	if _, ok := th.Allow(rotatingClient(globalBurst)); !ok {
		t.Error("attempt denied after the bucket refilled")
	}
}

func TestLoginThrottleSuccessRefundsGlobalBudget(t *testing.T) {
	th, _ := newTestThrottle()

	if _, ok := th.Allow("192.0.2.1"); !ok {
		t.Fatal("first attempt denied")
	}
	th.RecordSuccess("192.0.2.1")

	// The successful attempt gave its token back, so the full burst is still
	// available to anyone else.
	for i := range globalBurst {
		if _, ok := th.Allow(rotatingClient(i)); !ok {
			t.Fatalf("attempt %d denied, want the successful login to have been refunded", i)
		}
	}
}

func TestLoginThrottleGroupsIPv6By64(t *testing.T) {
	th, _ := newTestThrottle()

	for range maxClientFailures {
		th.RecordFailure("2001:db8::1")
	}

	if _, ok := th.Allow("2001:db8::dead:beef"); ok {
		t.Error("another address of the same /64 escaped the lockout")
	}

	if _, ok := th.Allow("2001:db8:1::1"); !ok {
		t.Error("an address of a different /64 was locked out")
	}
}

func TestLoginThrottleCapsConcurrentVerifications(t *testing.T) {
	th, _ := newTestThrottle()

	releases := make([]func(), 0, maxConcurrentVerifications)
	for i := range maxConcurrentVerifications {
		release, ok := th.Acquire()
		if !ok {
			t.Fatalf("slot %d refused, want the first %d to be granted", i, maxConcurrentVerifications)
		}
		releases = append(releases, release)
	}

	// One more must be refused rather than pile another bcrypt onto the cores.
	// Acquire waits for a grace period, so free a slot concurrently to keep the
	// test fast and to check that a released slot is reusable.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		release, ok := th.Acquire()
		if !ok {
			t.Error("slot refused although one was released")
			return
		}
		release()
	}()

	releases[0]()
	wg.Wait()

	for _, release := range releases[1:] {
		release()
	}
}

func TestLoginThrottleReleaseIsIdempotent(t *testing.T) {
	th, _ := newTestThrottle()

	release, ok := th.Acquire()
	if !ok {
		t.Fatal("first slot refused")
	}
	release()
	release()

	// A double release must not have inflated the semaphore.
	releases := make([]func(), 0, maxConcurrentVerifications)
	for i := range maxConcurrentVerifications {
		r, ok := th.Acquire()
		if !ok {
			t.Fatalf("slot %d refused", i)
		}
		releases = append(releases, r)
	}
	for _, r := range releases {
		defer r()
	}

	if len(th.slots) != maxConcurrentVerifications {
		t.Errorf("semaphore holds %d slots, want %d", len(th.slots), maxConcurrentVerifications)
	}
}

func TestLoginThrottleEvictsTrackedClients(t *testing.T) {
	th, clock := newTestThrottle()

	for i := range maxTrackedClients + 100 {
		th.RecordFailure(rotatingClient(i))
		*clock = clock.Add(time.Millisecond)
	}

	if len(th.clients) > maxTrackedClients {
		t.Errorf("tracking %d clients, want at most %d", len(th.clients), maxTrackedClients)
	}
}

// rotatingClient builds a distinct IPv4 address for index i.
func rotatingClient(i int) string {
	return fmt.Sprintf("198.51.%d.%d", i/256, i%256)
}
