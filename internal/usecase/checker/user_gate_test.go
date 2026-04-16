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
	"testing"
	"time"

	"git.happydns.org/happyDomain/model"
)

// mockUserResolver is declared in janitor_test.go (same package).

func newGateResolver() *mockUserResolver {
	return &mockUserResolver{users: make(map[string]*happydns.User)}
}

func addGateUser(r *mockUserResolver, quota happydns.UserQuota, lastSeen time.Time) string {
	uid, _ := happydns.NewRandomIdentifier()
	r.users[uid.String()] = &happydns.User{
		Id:       uid,
		LastSeen: lastSeen,
		Quota:    quota,
	}
	return uid.String()
}

// --- Allow tests ---

func TestUserGater_ActiveUser(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{}, time.Now())

	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: uid}

	if !g.Allow(target) {
		t.Error("expected active user to be allowed")
	}
}

func TestUserGater_SchedulingPaused(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{SchedulingPaused: true}, time.Now())

	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: uid}

	if g.Allow(target) {
		t.Error("expected paused user to be blocked")
	}
}

func TestUserGater_InactiveUser(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{}, time.Now().AddDate(0, 0, -100))

	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: uid}

	if g.Allow(target) {
		t.Error("expected inactive user (100 days) to be blocked with 90-day threshold")
	}
}

func TestUserGater_InactiveUserWithinThreshold(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{}, time.Now().AddDate(0, 0, -30))

	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: uid}

	if !g.Allow(target) {
		t.Error("expected user seen 30 days ago to be allowed with 90-day threshold")
	}
}

func TestUserGater_PerUserInactivityOverride(t *testing.T) {
	r := newGateResolver()
	// User has custom 14-day inactivity threshold, last seen 20 days ago.
	uid := addGateUser(r, happydns.UserQuota{InactivityPauseDays: 14}, time.Now().AddDate(0, 0, -20))

	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: uid}

	if g.Allow(target) {
		t.Error("expected user with 14-day override to be blocked after 20 days")
	}
}

func TestUserGater_NegativeInactivityDaysDisablesCheck(t *testing.T) {
	r := newGateResolver()
	// User opts out of inactivity pause with negative value, last seen 1 year ago.
	uid := addGateUser(r, happydns.UserQuota{InactivityPauseDays: -1}, time.Now().AddDate(-1, 0, 0))

	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: uid}

	if !g.Allow(target) {
		t.Error("expected negative InactivityPauseDays to disable inactivity check")
	}
}

func TestUserGater_ZeroDefaultInactivityDisablesCheck(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{}, time.Now().AddDate(-1, 0, 0))

	g := NewUserGater(r, 0, 0) // system default disabled
	target := happydns.CheckTarget{UserId: uid}

	if !g.Allow(target) {
		t.Error("expected zero defaultInactivityDays to disable inactivity check")
	}
}

func TestUserGater_NegativeDefaultInactivityDisablesCheck(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{}, time.Now().AddDate(-1, 0, 0))

	g := NewUserGater(r, -1, 0)
	target := happydns.CheckTarget{UserId: uid}

	if !g.Allow(target) {
		t.Error("expected negative defaultInactivityDays to disable inactivity check")
	}
}

func TestUserGater_ZeroLastSeenAllowed(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{}, time.Time{})

	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: uid}

	if !g.Allow(target) {
		t.Error("expected zero LastSeen to be allowed (user never logged in yet)")
	}
}

func TestUserGater_UnknownUserAllowed(t *testing.T) {
	r := newGateResolver()
	uid, _ := happydns.NewRandomIdentifier()

	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: uid.String()}

	if !g.Allow(target) {
		t.Error("expected unknown user to be allowed (fail-open)")
	}
}

func TestUserGater_EmptyUserIdAllowed(t *testing.T) {
	r := newGateResolver()
	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: ""}

	if !g.Allow(target) {
		t.Error("expected empty UserId to be allowed")
	}
}

func TestUserGater_NilResolverAllowed(t *testing.T) {
	g := NewUserGater(nil, 90, 0)
	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String()}

	if !g.Allow(target) {
		t.Error("expected nil resolver to allow all targets")
	}
}

// --- Cache tests ---

func TestUserGater_CacheHit(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{SchedulingPaused: true}, time.Now())

	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: uid}

	// First call populates cache.
	if g.Allow(target) {
		t.Fatal("expected paused user to be blocked")
	}

	// Remove user from resolver; cached result should still apply.
	delete(r.users, uid)

	if g.Allow(target) {
		t.Error("expected cached blocked result to persist")
	}
}

func TestUserGater_Invalidate(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{SchedulingPaused: true}, time.Now())

	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: uid}

	// Populate cache with blocked result.
	if g.Allow(target) {
		t.Fatal("expected paused user to be blocked")
	}

	// Admin unpauses the user.
	r.users[uid].Quota.SchedulingPaused = false

	// Without invalidation, cache still blocks.
	if g.Allow(target) {
		t.Fatal("expected cache to still block before invalidation")
	}

	// Invalidate and re-check.
	g.Invalidate(uid)

	if !g.Allow(target) {
		t.Error("expected user to be allowed after invalidation and unpause")
	}
}

func TestUserGater_CacheExpiry(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{SchedulingPaused: true}, time.Now())

	g := NewUserGater(r, 90, 0)
	g.cacheTTL = 10 * time.Millisecond // very short TTL for testing
	target := happydns.CheckTarget{UserId: uid}

	// Populate cache.
	if g.Allow(target) {
		t.Fatal("expected paused user to be blocked")
	}

	// Unpause and wait for cache expiry.
	r.users[uid].Quota.SchedulingPaused = false
	time.Sleep(20 * time.Millisecond)

	if !g.Allow(target) {
		t.Error("expected cache to expire and re-evaluate to allowed")
	}
}

// --- Daily budget tests ---

func TestUserGater_UnlimitedByDefault(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{}, time.Now())
	g := NewUserGater(r, 90, 0) // default = 0 means unlimited
	target := happydns.CheckTarget{UserId: uid}

	for i := 0; i < 1000; i++ {
		if !g.Allow(target) {
			t.Fatalf("expected unlimited quota to always allow, blocked at i=%d", i)
		}
		g.IncrementUsage(happydns.CheckTarget{UserId: uid})
	}
	if g.IsRateLimited(uid, 0) {
		t.Error("expected IsRateLimited=false for unlimited user")
	}
}

func TestUserGater_NegativeQuotaOptsOutPerUser(t *testing.T) {
	// A negative per-user MaxChecksPerDay means "explicitly unlimited for
	// this user", regardless of the system default.
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{MaxChecksPerDay: -1}, time.Now())
	g := NewUserGater(r, 90, 5) // default = 5 (tight)
	target := happydns.CheckTarget{UserId: uid}

	for i := 0; i < 50; i++ {
		if !g.Allow(target) {
			t.Fatalf("expected per-user opt-out (negative quota) to always allow, blocked at i=%d", i)
		}
		g.IncrementUsage(happydns.CheckTarget{UserId: uid})
	}
	if g.IsRateLimited(uid, 0) {
		t.Error("expected IsRateLimited=false for user with negative quota")
	}
	if g.IsRateLimited(uid, time.Minute) {
		t.Error("expected IsRateLimited=false for short interval on opted-out user")
	}
	// Interval-aware throttling must also be bypassed.
	if !g.AllowWithInterval(target, time.Minute) {
		t.Error("expected short-interval check to be allowed for opted-out user")
	}
}

func TestUserGater_DailyQuotaEnforced(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{MaxChecksPerDay: 5}, time.Now())
	g := NewUserGater(r, 90, 100)
	target := happydns.CheckTarget{UserId: uid}

	// User quota overrides default (5 < 100).
	for i := 0; i < 5; i++ {
		if !g.Allow(target) {
			t.Fatalf("expected first 5 checks to be allowed, blocked at i=%d", i)
		}
		g.IncrementUsage(happydns.CheckTarget{UserId: uid})
	}
	if g.Allow(target) {
		t.Error("expected 6th check to be blocked (quota 5 exceeded)")
	}
	if !g.IsRateLimited(uid, 0) {
		t.Error("expected IsRateLimited=true after exhausting quota")
	}
}

func TestUserGater_SystemDefaultEnforced(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{}, time.Now()) // no per-user quota
	g := NewUserGater(r, 90, 3)                             // default = 3
	target := happydns.CheckTarget{UserId: uid}

	for i := 0; i < 3; i++ {
		if !g.Allow(target) {
			t.Fatalf("expected first 3 checks to be allowed, blocked at i=%d", i)
		}
		g.IncrementUsage(happydns.CheckTarget{UserId: uid})
	}
	if g.Allow(target) {
		t.Error("expected 4th check to be blocked (default 3 exceeded)")
	}
}

func TestUserGater_IntervalThrottling(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{MaxChecksPerDay: 10}, time.Now())
	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: uid}

	// Burn 80% of the budget (8 checks) — we are now in throttle mode.
	for i := 0; i < 8; i++ {
		if !g.AllowWithInterval(target, time.Minute) {
			t.Fatalf("expected first 8 checks to be allowed, blocked at i=%d", i)
		}
		g.IncrementUsage(happydns.CheckTarget{UserId: uid})
	}

	// At 80% usage: short-interval jobs denied.
	if g.AllowWithInterval(target, time.Minute) {
		t.Error("expected 1-minute interval to be throttled at 80% usage")
	}
	if g.AllowWithInterval(target, time.Hour) {
		t.Error("expected 1-hour interval (< 4h) to be throttled at 80% usage")
	}

	// Long-interval jobs still allowed.
	if !g.AllowWithInterval(target, 6*time.Hour) {
		t.Error("expected 6-hour interval to be allowed at 80% usage")
	}
	if !g.AllowWithInterval(target, 24*time.Hour) {
		t.Error("expected daily interval to be allowed at 80% usage")
	}
}

func TestUserGater_AllowTreatsZeroIntervalAsLong(t *testing.T) {
	// Plain Allow (no interval info) should not trigger interval-based
	// throttling — it should only deny at the hard limit.
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{MaxChecksPerDay: 10}, time.Now())
	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: uid}

	for i := 0; i < 9; i++ {
		g.IncrementUsage(happydns.CheckTarget{UserId: uid})
	}
	// At 90% usage, Allow (interval=0) should still allow.
	if !g.Allow(target) {
		t.Error("expected Allow to ignore interval-throttling (interval=0 means long)")
	}
}

func TestUserGater_IncrementUsage(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{MaxChecksPerDay: 2}, time.Now())
	g := NewUserGater(r, 90, 0)

	g.IncrementUsage(happydns.CheckTarget{UserId: uid})
	g.IncrementUsage(happydns.CheckTarget{UserId: uid})
	if !g.IsRateLimited(uid, 0) {
		t.Error("expected IsRateLimited=true after 2 increments with limit 2")
	}
}

func TestUserGater_IncrementUsageEmptyUID(t *testing.T) {
	g := NewUserGater(nil, 90, 5)
	g.IncrementUsage(happydns.CheckTarget{}) // must not panic
}

func TestUserGater_IsRateLimitedUnknownUser(t *testing.T) {
	g := NewUserGater(nil, 90, 5)
	if g.IsRateLimited("nonexistent", 0) {
		t.Error("expected unknown user to not be rate-limited (nil resolver, no user known)")
	}
}

func TestUserGater_IsRateLimitedEmptyUID(t *testing.T) {
	g := NewUserGater(nil, 90, 5)
	if g.IsRateLimited("", time.Minute) {
		t.Error("expected empty userID to not be rate-limited")
	}
}

func TestUserGater_RateLimiterForMatchesIsRateLimited(t *testing.T) {
	// The closure returned by RateLimiterFor must agree with IsRateLimited
	// across the interesting state-space: unlimited / under-threshold /
	// throttling / hard-limit. IsRateLimited is the single-shot convenience
	// wrapper built on top of RateLimiterFor, so any divergence signals a
	// bug in either code path.
	intervals := []time.Duration{
		0,
		time.Minute,
		time.Hour,
		3*time.Hour + 59*time.Minute, // just under the 4h cutoff
		4 * time.Hour,                // exactly at the cutoff (not throttled)
		6 * time.Hour,
		24 * time.Hour,
	}

	cases := []struct {
		name      string
		increment int
	}{
		{"zero usage (no throttling, no hard limit)", 0},
		{"under threshold (used=7, limit=10)", 7},
		{"at threshold (used=8, limit=10, throttling)", 8},
		{"near hard limit (used=9, limit=10, throttling)", 9},
		{"hard limit reached (used=10, limit=10)", 10},
		{"past hard limit (used=15, limit=10)", 15},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newGateResolver()
			uid := addGateUser(r, happydns.UserQuota{MaxChecksPerDay: 10}, time.Now())
			g := NewUserGater(r, 90, 0)

			for i := 0; i < tc.increment; i++ {
				g.IncrementUsage(happydns.CheckTarget{UserId: uid})
			}

			snapshot := g.RateLimiterFor(uid)
			for _, interval := range intervals {
				got := snapshot(interval)
				want := g.IsRateLimited(uid, interval)
				if got != want {
					t.Errorf("RateLimiterFor(%q)(%v) = %v; IsRateLimited = %v (disagreement)",
						uid, interval, got, want)
				}
			}
		})
	}
}

func TestUserGater_RateLimiterForSnapshot(t *testing.T) {
	// A closure obtained before further IncrementUsage calls must reflect
	// the used/limit values at the moment RateLimiterFor was invoked, not
	// the current state. This is the behavioural contract that makes the
	// API amortise the budget lookup across many planned-job lookups.
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{MaxChecksPerDay: 10}, time.Now())
	g := NewUserGater(r, 90, 0)

	// At used=0, nothing is throttled.
	snapshot := g.RateLimiterFor(uid)
	if snapshot(time.Minute) {
		t.Fatal("expected fresh snapshot to allow short interval")
	}

	// Drive the live state into hard-limit territory.
	for i := 0; i < 20; i++ {
		g.IncrementUsage(happydns.CheckTarget{UserId: uid})
	}
	// The snapshot captured earlier must still allow; only a fresh snapshot
	// should see the new state.
	if snapshot(time.Minute) {
		t.Error("expected snapshot to be frozen at used=0, but it reflected later increments")
	}
	if !g.RateLimiterFor(uid)(time.Minute) {
		t.Error("expected fresh snapshot to deny short interval at used>=limit")
	}
}

func TestUserGater_MidnightReset(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{MaxChecksPerDay: 1}, time.Now())
	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: uid}

	g.IncrementUsage(happydns.CheckTarget{UserId: uid})
	if g.Allow(target) {
		t.Fatal("expected user to be over budget after 1 check with limit 1")
	}

	// Simulate crossing a day boundary by rewriting the stored date.
	g.budgetMu.Lock()
	b := g.budgets[uid]
	b.date = b.date.AddDate(0, 0, -1) // pretend stored budget is from yesterday
	g.budgetMu.Unlock()

	// Next call should reset the counter and allow.
	if !g.Allow(target) {
		t.Error("expected budget to reset on new day and allow")
	}
	if g.IsRateLimited(uid, 0) {
		t.Error("expected IsRateLimited=false after day reset")
	}
}

func TestUserGater_InvalidatePreservesUsage(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{MaxChecksPerDay: 5}, time.Now())
	g := NewUserGater(r, 90, 0)

	g.IncrementUsage(happydns.CheckTarget{UserId: uid})
	g.IncrementUsage(happydns.CheckTarget{UserId: uid})
	g.IncrementUsage(happydns.CheckTarget{UserId: uid})

	// Admin raises the quota; invalidate is called.
	r.users[uid].Quota.MaxChecksPerDay = 10
	g.Invalidate(uid)

	// Counter should be preserved, not reset.
	g.budgetMu.Lock()
	b := g.budgets[uid]
	g.budgetMu.Unlock()
	used := int64(0)
	limit := int64(0)
	if b != nil {
		used = b.used.Load()
		limit = b.limit.Load()
	}

	if used != 3 {
		t.Errorf("expected usage counter preserved at 3, got %d", used)
	}
	if limit != 10 {
		t.Errorf("expected new limit 10 after invalidate, got %d", limit)
	}
}

func TestUserGater_BlockedPolicyShortCircuitsBudget(t *testing.T) {
	// When policy (paused) denies, budget should not be consulted/mutated.
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{SchedulingPaused: true, MaxChecksPerDay: 100}, time.Now())
	g := NewUserGater(r, 90, 0)
	target := happydns.CheckTarget{UserId: uid}

	if g.Allow(target) {
		t.Error("expected paused user to be blocked regardless of budget")
	}
}

// TestUserGater_InvalidateDoesNotLoseIncrements guards against the
// delete+replace race the old Invalidate had: IncrementUsage calls running
// concurrently with Invalidate must not be dropped. Run with -race.
func TestUserGater_InvalidateDoesNotLoseIncrements(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{MaxChecksPerDay: 1_000_000}, time.Now())
	g := NewUserGater(r, 90, 0)

	const increments = 10_000
	const invalidations = 100

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < increments; i++ {
			g.IncrementUsage(happydns.CheckTarget{UserId: uid})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < invalidations; i++ {
			g.Invalidate(uid)
		}
	}()

	wg.Wait()

	g.budgetMu.Lock()
	b := g.budgets[uid]
	g.budgetMu.Unlock()
	if b == nil {
		t.Fatal("expected budget entry to exist after concurrent run")
	}
	if got := b.used.Load(); got != increments {
		t.Errorf("lost increments under concurrent Invalidate: got used=%d, want %d", got, increments)
	}
}

// --- Sweep tests ---

func TestUserGater_SweepExpiredCacheEntries(t *testing.T) {
	r := newGateResolver()
	liveUID := addGateUser(r, happydns.UserQuota{}, time.Now())
	staleUID := addGateUser(r, happydns.UserQuota{}, time.Now())

	g := NewUserGater(r, 90, 0)

	// Populate the cache for both users.
	g.Allow(happydns.CheckTarget{UserId: liveUID})
	g.Allow(happydns.CheckTarget{UserId: staleUID})

	// Expire the stale entry by rewriting its expiry time.
	g.mu.Lock()
	e := g.cache[staleUID]
	e.expires = time.Now().Add(-time.Second)
	g.cache[staleUID] = e
	g.mu.Unlock()

	cachePruned, _ := g.Sweep()
	if cachePruned != 1 {
		t.Errorf("expected 1 cache entry pruned, got %d", cachePruned)
	}

	g.mu.Lock()
	_, staleStillThere := g.cache[staleUID]
	_, liveStillThere := g.cache[liveUID]
	g.mu.Unlock()

	if staleStillThere {
		t.Error("expected stale cache entry to be evicted")
	}
	if !liveStillThere {
		t.Error("expected live cache entry to be preserved")
	}
}

func TestUserGater_SweepStaleBudgets(t *testing.T) {
	r := newGateResolver()
	todayUID := addGateUser(r, happydns.UserQuota{MaxChecksPerDay: 5}, time.Now())
	yesterdayUID := addGateUser(r, happydns.UserQuota{MaxChecksPerDay: 5}, time.Now())

	g := NewUserGater(r, 90, 0)

	g.IncrementUsage(happydns.CheckTarget{UserId: todayUID})
	g.IncrementUsage(happydns.CheckTarget{UserId: yesterdayUID})

	// Backdate the second user's budget to a previous UTC day.
	g.budgetMu.Lock()
	g.budgets[yesterdayUID].date = g.budgets[yesterdayUID].date.AddDate(0, 0, -1)
	g.budgetMu.Unlock()

	_, budgetsPruned := g.Sweep()
	if budgetsPruned != 1 {
		t.Errorf("expected 1 budget entry pruned, got %d", budgetsPruned)
	}

	g.budgetMu.Lock()
	_, yesterdayStillThere := g.budgets[yesterdayUID]
	_, todayStillThere := g.budgets[todayUID]
	g.budgetMu.Unlock()

	if yesterdayStillThere {
		t.Error("expected stale budget entry to be evicted")
	}
	if !todayStillThere {
		t.Error("expected today's budget entry to be preserved")
	}
}

func TestUserGater_SweepNoopWhenFresh(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{MaxChecksPerDay: 5}, time.Now())
	g := NewUserGater(r, 90, 0)

	g.Allow(happydns.CheckTarget{UserId: uid})
	g.IncrementUsage(happydns.CheckTarget{UserId: uid})

	cachePruned, budgetsPruned := g.Sweep()
	if cachePruned != 0 || budgetsPruned != 0 {
		t.Errorf("expected Sweep to be a no-op on fresh state, got cache=%d budgets=%d", cachePruned, budgetsPruned)
	}
}

func TestUserGater_StartStopRunsSweep(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{MaxChecksPerDay: 5}, time.Now())
	g := NewUserGater(r, 90, 0)
	g.sweepInterval = 10 * time.Millisecond

	// Populate a stale budget so the sweeper has something to prune.
	g.IncrementUsage(happydns.CheckTarget{UserId: uid})
	g.budgetMu.Lock()
	g.budgets[uid].date = g.budgets[uid].date.AddDate(0, 0, -1)
	g.budgetMu.Unlock()

	g.Start(context.Background())

	// Poll for the sweeper to prune the stale entry.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		g.budgetMu.Lock()
		_, ok := g.budgets[uid]
		g.budgetMu.Unlock()
		if !ok {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	g.Stop()

	g.budgetMu.Lock()
	_, staleStillThere := g.budgets[uid]
	g.budgetMu.Unlock()
	if staleStillThere {
		t.Error("expected background sweeper to evict stale budget entry")
	}
}

func TestUserGater_StartIsIdempotent(t *testing.T) {
	g := NewUserGater(newGateResolver(), 90, 0)
	g.sweepInterval = time.Hour // effectively disable ticks during the test

	g.Start(context.Background())
	g.Start(context.Background()) // second call must be a no-op, not start another goroutine

	g.Stop()
	// Reaching this point without hanging or panicking is the assertion.
}

func TestUserGater_StopWithoutStart(t *testing.T) {
	g := NewUserGater(newGateResolver(), 90, 0)
	g.Stop() // must not panic or block
}
