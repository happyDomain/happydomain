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
	"net"
	"sync"
	"time"
)

// MaxLoginBodyBytes caps the size of an admin login request body. A password is
// truncated at 72 bytes by bcrypt anyway, so anything past a kilobyte of JSON is
// an attempt to make an unauthenticated caller allocate server memory.
const MaxLoginBodyBytes = 1024

const (
	// maxClientFailures is the number of consecutive failed attempts a single
	// client may make before it gets locked out.
	maxClientFailures = 5

	// clientFailureWindow is how long a failure is remembered. A client that
	// stays quiet for that long starts over with a clean counter.
	clientFailureWindow = 15 * time.Minute

	// baseClientLockout is the lockout applied once maxClientFailures is
	// reached; it doubles for every further failure, up to maxClientLockout.
	baseClientLockout = 30 * time.Second
	maxClientLockout  = 30 * time.Minute

	// maxConcurrentVerifications bounds how many password verifications may run
	// at once. Verification is a deliberately expensive bcrypt comparison, so
	// without this cap an unauthenticated caller can peg every core of the
	// process, which also serves the public API.
	maxConcurrentVerifications = 4

	// globalBurst and globalRefill form the token bucket that bounds the login
	// attempt rate across all clients, so that rotating source addresses does
	// not defeat the per-client lockout.
	globalBurst  = 30
	globalRefill = 500 * time.Millisecond

	// maxTrackedClients bounds the memory the per-client table may use. Once
	// reached, the least recently seen entries are evicted.
	maxTrackedClients = 4096
)

// clientState is the failure history of a single client.
type clientState struct {
	failures  int
	lastSeen  time.Time
	lockUntil time.Time
}

// LoginThrottle protects the admin login endpoint against online password
// guessing and against the CPU exhaustion its own password hashing would
// otherwise enable. It combines three limits: a per-client lockout with
// exponential backoff, a global attempt rate, and a cap on how many
// verifications may run concurrently.
//
// It is safe for concurrent use, and its zero value is not usable: build one
// with NewLoginThrottle.
type LoginThrottle struct {
	mu      sync.Mutex
	clients map[string]*clientState

	// tokens is the global bucket, refilled lazily from lastRefill.
	tokens     float64
	lastRefill time.Time

	// slots is the verification concurrency semaphore.
	slots chan struct{}

	// now is overridable in tests.
	now func() time.Time
}

func NewLoginThrottle() *LoginThrottle {
	t := &LoginThrottle{
		clients: make(map[string]*clientState),
		tokens:  globalBurst,
		slots:   make(chan struct{}, maxConcurrentVerifications),
		now:     time.Now,
	}
	t.lastRefill = t.now()
	return t
}

// Allow reports whether client may attempt a login right now, consuming one
// token of the global budget when it may. When it may not, it returns how long
// the caller should wait before trying again.
//
// A client that is allowed through must report the outcome with RecordFailure or
// RecordSuccess, otherwise its failure counter never moves.
func (t *LoginThrottle) Allow(client string) (retryAfter time.Duration, ok bool) {
	key := clientKey(client)

	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()

	if st := t.clients[key]; st != nil && now.Before(st.lockUntil) {
		st.lastSeen = now
		return roundUpSecond(st.lockUntil.Sub(now)), false
	}

	t.refillLocked(now)
	if t.tokens < 1 {
		return roundUpSecond(time.Duration(float64(globalRefill) * (1 - t.tokens))), false
	}
	t.tokens--

	return 0, true
}

// Acquire reserves one of the limited verification slots, waiting up to a short
// grace period for one to free up. The returned release function must be called
// once verification is done. When no slot becomes available, ok is false and the
// caller must reject the request rather than run the verification anyway.
func (t *LoginThrottle) Acquire() (release func(), ok bool) {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	select {
	case t.slots <- struct{}{}:
		var once sync.Once
		return func() { once.Do(func() { <-t.slots }) }, true
	case <-timer.C:
		return func() {}, false
	}
}

// RecordFailure accounts a failed attempt for client, locking it out once it has
// failed too many times in a row. It returns the lockout now in effect, or zero
// when the client is still allowed to try again immediately.
func (t *LoginThrottle) RecordFailure(client string) (lockedFor time.Duration) {
	key := clientKey(client)

	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()

	st := t.clients[key]
	if st == nil {
		t.evictLocked(now)
		st = &clientState{}
		t.clients[key] = st
	} else if now.Sub(st.lastSeen) > clientFailureWindow {
		// The client went quiet long enough for its history to expire.
		st.failures = 0
	}

	st.failures++
	st.lastSeen = now

	if st.failures < maxClientFailures {
		return 0
	}

	lockout := baseClientLockout << min(st.failures-maxClientFailures, 16)
	if lockout > maxClientLockout || lockout <= 0 {
		lockout = maxClientLockout
	}
	st.lockUntil = now.Add(lockout)

	return lockout
}

// RecordSuccess clears the failure history of client and refunds the global
// budget it consumed: a legitimate operator signing in must not bring the
// endpoint closer to its rate limit.
func (t *LoginThrottle) RecordSuccess(client string) {
	key := clientKey(client)

	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.clients, key)

	if t.tokens < globalBurst {
		t.tokens++
	}
}

// refillLocked tops the global bucket up for the time elapsed since the last
// refill. Callers must hold t.mu.
func (t *LoginThrottle) refillLocked(now time.Time) {
	elapsed := now.Sub(t.lastRefill)
	if elapsed <= 0 {
		return
	}

	t.lastRefill = now
	t.tokens = min(t.tokens+float64(elapsed)/float64(globalRefill), globalBurst)
}

// evictLocked makes room for a new client entry: expired entries first, then the
// least recently seen one if the table is still full. Callers must hold t.mu.
func (t *LoginThrottle) evictLocked(now time.Time) {
	if len(t.clients) < maxTrackedClients {
		return
	}

	var oldestKey string
	var oldestSeen time.Time

	for key, st := range t.clients {
		if now.After(st.lockUntil) && now.Sub(st.lastSeen) > clientFailureWindow {
			delete(t.clients, key)
			continue
		}

		if oldestKey == "" || st.lastSeen.Before(oldestSeen) {
			oldestKey, oldestSeen = key, st.lastSeen
		}
	}

	if len(t.clients) >= maxTrackedClients && oldestKey != "" {
		delete(t.clients, oldestKey)
	}
}

// clientKey normalizes a client address into the identity the lockout applies
// to. IPv6 clients are grouped by their /64 prefix: a single host is routinely
// handed a whole /64, so counting failures per address would let it rotate out
// of any lockout for free.
// An address that cannot be parsed shares a single bucket rather than handing
// out a fresh entry per garbage value, which would otherwise be a way to grow
// the client table. This mirrors the UnknownClientKey fallback the API side
// applies in internal/api/middleware.ClientKey; the two must stay in agreement.
func clientKey(client string) string {
	ip := net.ParseIP(client)
	if ip == nil {
		return "unknown"
	}

	if ip.To4() != nil {
		return ip.String()
	}

	return ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
}

// roundUpSecond rounds d up to the next whole second, with a floor of one
// second, so that it can be advertised in a Retry-After header.
func roundUpSecond(d time.Duration) time.Duration {
	if d <= time.Second {
		return time.Second
	}

	if rest := d % time.Second; rest > 0 {
		d += time.Second - rest
	}

	return d
}
