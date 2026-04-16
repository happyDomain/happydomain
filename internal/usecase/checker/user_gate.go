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
	"log"
	"sync"
	"sync/atomic"
	"time"

	"git.happydns.org/happyDomain/model"
)

// throttleShortIntervalCutoff is the interval below which jobs are skipped
// when the daily budget is getting tight (>= throttleFillRatio consumed).
// Jobs scheduled more rarely than this are considered "important enough"
// that they should be allowed through so they are not starved by frequent
// short-interval checks.
const throttleShortIntervalCutoff = 4 * time.Hour

// throttleFillRatio is the budget consumption ratio above which
// short-interval jobs start being throttled.
const throttleFillRatio = 0.8

// defaultSweepInterval is how often the background sweeper runs by default
// to evict expired policy cache entries and stale daily-budget entries.
const defaultSweepInterval = 1 * time.Hour

// UserGater builds a Scheduler gate function that filters out check jobs
// belonging to users that are paused, have been inactive for too long, or
// have exhausted their daily MaxChecksPerDay quota.
//
// Lookups are cached for a short TTL so the scheduler hot path does not hit
// storage on every job pop. Daily usage counters are kept in memory; they
// reset at midnight UTC and on process restart. A background sweeper
// (started via Start) periodically evicts expired cache entries and
// previous-day budget entries so the in-memory maps do not grow unbounded
// in a long-running process with many transient users.
type UserGater struct {
	resolver               JanitorUserResolver
	defaultInactivityDays  int
	defaultMaxChecksPerDay int
	cacheTTL               time.Duration
	sweepInterval          time.Duration

	mu    sync.Mutex
	cache map[string]gateCacheEntry

	budgetMu sync.Mutex
	budgets  map[string]*dailyBudget

	sweepMu      sync.Mutex
	sweepCancel  context.CancelFunc
	sweepDone    chan struct{}
	sweepRunning bool
}

type gateCacheEntry struct {
	allow   bool
	expires time.Time
}

// dailyBudget tracks per-user check executions for the current UTC day.
// date is immutable after creation; limit and used are atomic so that
// Invalidate can refresh the limit in place without taking budgetMu for
// the (slow) storage lookup, and without losing concurrent increments.
type dailyBudget struct {
	date  time.Time // UTC midnight of the tracked day (immutable)
	limit atomic.Int64
	used  atomic.Int64
}

// NewUserGater creates a UserGater. defaultInactivityDays is used for users
// whose UserQuota.InactivityPauseDays is zero. A negative effective value
// disables inactivity-based pausing for that user. defaultMaxChecksPerDay
// is used for users whose UserQuota.MaxChecksPerDay is zero; 0 means
// unlimited. A negative UserQuota.MaxChecksPerDay disables the daily cap
// for that specific user regardless of the system default.
func NewUserGater(resolver JanitorUserResolver, defaultInactivityDays, defaultMaxChecksPerDay int) *UserGater {
	return &UserGater{
		resolver:               resolver,
		defaultInactivityDays:  defaultInactivityDays,
		defaultMaxChecksPerDay: defaultMaxChecksPerDay,
		cacheTTL:               5 * time.Minute,
		sweepInterval:          defaultSweepInterval,
		cache:                  map[string]gateCacheEntry{},
		budgets:                map[string]*dailyBudget{},
	}
}

// Allow returns true if the scheduler should run jobs for the given target.
// It is equivalent to AllowWithInterval(target, 0), which treats the job as
// "long-interval" for the purpose of budget-aware throttling (i.e. the job
// is only denied when the hard limit is reached, never throttled early).
func (g *UserGater) Allow(target happydns.CheckTarget) bool {
	return g.AllowWithInterval(target, 0)
}

// AllowWithInterval returns true if the scheduler should run jobs for the
// given target. The job interval is used for budget-aware throttling: when
// the user's daily budget is more than throttleFillRatio consumed, jobs
// with interval shorter than throttleShortIntervalCutoff are denied so that
// rarer checks (interval >= cutoff) are not starved.
func (g *UserGater) AllowWithInterval(target happydns.CheckTarget, interval time.Duration) bool {
	uid := target.UserId
	if uid == "" || g.resolver == nil {
		return true
	}

	// 1) Policy layer (paused / inactivity), cached for cacheTTL.
	if !g.allowPolicy(uid) {
		return false
	}

	// 2) Daily budget layer (not cached; the counter changes on every
	// execution and must be accurate).
	return g.allowBudget(uid, interval)
}

// allowPolicy runs the paused / inactivity checks with caching.
func (g *UserGater) allowPolicy(uid string) bool {
	g.mu.Lock()
	if e, ok := g.cache[uid]; ok && time.Now().Before(e.expires) {
		g.mu.Unlock()
		return e.allow
	}
	g.mu.Unlock()

	allow := g.compute(uid)

	g.mu.Lock()
	g.cache[uid] = gateCacheEntry{allow: allow, expires: time.Now().Add(g.cacheTTL)}
	g.mu.Unlock()

	return allow
}

// allowBudget enforces the MaxChecksPerDay quota. Returns true if the user
// still has budget for a job of the given interval. A limit of 0 means
// unlimited.
func (g *UserGater) allowBudget(uid string, interval time.Duration) bool {
	b := g.getOrCreateBudget(uid)
	limit := b.limit.Load()
	if limit <= 0 {
		return true
	}
	used := b.used.Load()
	if used >= limit {
		return false
	}
	// Interval-aware throttling: once 80% of the budget is consumed,
	// start skipping short-interval jobs so longer-interval (rarer,
	// more "important") jobs are not starved.
	if float64(used)/float64(limit) >= throttleFillRatio && interval > 0 && interval < throttleShortIntervalCutoff {
		return false
	}
	return true
}

// getOrCreateBudget returns the current day's budget for the user, resetting
// it if the stored date is stale. Always returns a non-nil *dailyBudget; a
// zero limit (unlimited) is still represented by an entry in the map so
// concurrent Invalidate calls have something to update atomically.
func (g *UserGater) getOrCreateBudget(uid string) *dailyBudget {
	today := todayUTC()

	g.budgetMu.Lock()
	b, ok := g.budgets[uid]
	if ok && b.date.Equal(today) {
		g.budgetMu.Unlock()
		return b
	}
	g.budgetMu.Unlock()

	// Need to (re)compute the limit. Resolve the user outside the lock.
	limit := g.resolveLimit(uid)

	g.budgetMu.Lock()
	defer g.budgetMu.Unlock()
	// Re-check in case another goroutine created/refreshed it.
	b, ok = g.budgets[uid]
	if ok && b.date.Equal(today) {
		return b
	}
	b = &dailyBudget{date: today}
	b.limit.Store(int64(limit))
	g.budgets[uid] = b
	return b
}

// resolveUser fetches a user by ID string. Returns nil when the resolver
// is absent, the ID is malformed, or the user is unknown — callers should
// treat that as "no policy/limit info available" and apply fail-open
// defaults.
func (g *UserGater) resolveUser(uid string) *happydns.User {
	if g.resolver == nil {
		return nil
	}
	id, err := happydns.NewIdentifierFromString(uid)
	if err != nil {
		return nil
	}
	user, err := g.resolver.GetUser(id)
	if err != nil {
		return nil
	}
	return user
}

// resolveLimit fetches the user and resolves the effective daily cap.
// Returns the system default when the user cannot be resolved. A positive
// UserQuota.MaxChecksPerDay is used as-is; a negative value means the user
// has explicitly opted out of the cap (returns 0, i.e. unlimited); zero
// falls back to the system default.
func (g *UserGater) resolveLimit(uid string) int {
	user := g.resolveUser(uid)
	if user == nil {
		return g.defaultMaxChecksPerDay
	}
	if user.Quota.MaxChecksPerDay > 0 {
		return user.Quota.MaxChecksPerDay
	}
	if user.Quota.MaxChecksPerDay < 0 {
		return 0
	}
	return g.defaultMaxChecksPerDay
}

// IncrementUsage records one executed check against the user's daily budget.
// Called by the scheduler after successfully launching a scheduled job.
// No-op when the target has no user.
func (g *UserGater) IncrementUsage(target happydns.CheckTarget) {
	uid := target.UserId
	if uid == "" {
		return
	}
	g.getOrCreateBudget(uid).used.Add(1)
}

// IsRateLimited returns true when a scheduled execution of the given interval
// would be denied by the user's daily budget right now — either because the
// hard MaxChecksPerDay cap is reached or because interval-aware throttling is
// currently skipping short-interval jobs. Returns false for unknown users and
// for users with no limit.
//
// This is a one-shot convenience that wraps RateLimiterFor. Callers that need
// to evaluate many intervals for the same user should use RateLimiterFor
// directly to avoid repeated budget lookups.
func (g *UserGater) IsRateLimited(userID string, interval time.Duration) bool {
	return g.RateLimiterFor(userID)(interval)
}

// RateLimiterFor returns a closure that reports whether a scheduled execution
// of the given interval would be denied by the user's daily budget right now.
// The budget is resolved (and any locks acquired) exactly once at call time;
// the returned closure is a pure function over the interval and performs no
// further lookups. Callers iterating over many planned jobs for a single user
// should use this in preference to IsRateLimited.
//
// Snapshot semantics: the closure reflects the used/limit values observed at
// call time. Concurrent IncrementUsage or Invalidate happening after the call
// will not affect subsequent closure invocations. This is acceptable for the
// API-layer "list planned executions" use case, which is already a
// best-effort view.
//
// Returns a closure that always reports false when the userID is empty, the
// user is unknown, or the user has no effective limit.
func (g *UserGater) RateLimiterFor(userID string) func(time.Duration) bool {
	none := func(time.Duration) bool { return false }
	if userID == "" {
		return none
	}
	b := g.getOrCreateBudget(userID)
	limit := b.limit.Load()
	if limit <= 0 {
		return none
	}
	used := b.used.Load()
	if used >= limit {
		return func(time.Duration) bool { return true }
	}
	if float64(used)/float64(limit) < throttleFillRatio {
		return none
	}
	// Throttling active: short-interval jobs are denied, longer ones pass.
	return func(interval time.Duration) bool {
		return interval > 0 && interval < throttleShortIntervalCutoff
	}
}

// Invalidate drops any cached policy decision for the given user. Call this
// when a user's quota or LastSeen changes (e.g. on login or admin update).
// Also refreshes the cached daily budget limit so a quota change takes
// effect immediately; the usage counter is preserved.
//
// The refresh updates b.limit atomically in place rather than replacing
// the budget entry. This is important: readers in allowBudget /
// RateLimiterFor and writers in IncrementUsage all operate on the same
// *dailyBudget, so concurrent increments cannot be lost during the
// (possibly slow) storage lookup inside resolveLimit.
func (g *UserGater) Invalidate(userID string) {
	g.mu.Lock()
	delete(g.cache, userID)
	g.mu.Unlock()

	g.budgetMu.Lock()
	b, ok := g.budgets[userID]
	if !ok {
		g.budgetMu.Unlock()
		return
	}
	// Stale entry from a previous UTC day: drop it. getOrCreateBudget
	// will build a fresh one (with a freshly resolved limit) on the next
	// call, so there is no point refreshing the limit here.
	if !b.date.Equal(todayUTC()) {
		delete(g.budgets, userID)
		g.budgetMu.Unlock()
		return
	}
	g.budgetMu.Unlock()

	// Refresh the limit outside budgetMu. Concurrent readers see either
	// the old or new limit atomically; the used counter is untouched.
	b.limit.Store(int64(g.resolveLimit(userID)))
}

func (g *UserGater) compute(uid string) bool {
	user := g.resolveUser(uid)
	if user == nil {
		// Be conservative: allow rather than silently dropping work.
		return true
	}
	if user.Quota.SchedulingPaused {
		return false
	}

	days := user.Quota.InactivityPauseDays
	if days == 0 {
		days = g.defaultInactivityDays
	}
	if days <= 0 {
		return true
	}
	if user.LastSeen.IsZero() {
		return true
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	return user.LastSeen.After(cutoff)
}

// Sweep evicts entries that can no longer be consulted usefully:
//   - policy-cache entries whose TTL has expired,
//   - daily-budget entries from a previous UTC day.
//
// Returns the number of entries removed from each map. It is safe to call
// at any time, and is invoked periodically by the background sweeper that
// Start launches.
func (g *UserGater) Sweep() (cachePruned, budgetsPruned int) {
	now := time.Now()

	g.mu.Lock()
	for uid, e := range g.cache {
		if !e.expires.After(now) {
			delete(g.cache, uid)
			cachePruned++
		}
	}
	g.mu.Unlock()

	today := todayUTC()
	g.budgetMu.Lock()
	for uid, b := range g.budgets {
		if !b.date.Equal(today) {
			delete(g.budgets, uid)
			budgetsPruned++
		}
	}
	g.budgetMu.Unlock()

	return
}

// Start launches a background goroutine that periodically sweeps stale
// cache and budget entries. Calling Start again while the sweeper is
// already running is a no-op.
func (g *UserGater) Start(ctx context.Context) {
	g.sweepMu.Lock()
	if g.sweepRunning {
		g.sweepMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	g.sweepCancel = cancel
	g.sweepDone = done
	g.sweepRunning = true
	interval := g.sweepInterval
	g.sweepMu.Unlock()

	go g.sweepLoop(ctx, done, interval)
}

// Stop halts the background sweeper, if any, and waits for it to exit.
func (g *UserGater) Stop() {
	g.sweepMu.Lock()
	cancel := g.sweepCancel
	done := g.sweepDone
	g.sweepCancel = nil
	g.sweepDone = nil
	g.sweepRunning = false
	g.sweepMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (g *UserGater) sweepLoop(ctx context.Context, done chan struct{}, interval time.Duration) {
	defer close(done)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cache, budgets := g.Sweep()
			if cache > 0 || budgets > 0 {
				log.Printf("UserGater: swept %d cache / %d budget entries", cache, budgets)
			}
		}
	}
}

// todayUTC returns UTC midnight of the current day.
func todayUTC() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
