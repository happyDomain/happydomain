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
	"sync"
	"time"

	"git.happydns.org/happyDomain/model"
)

// UserGater builds a Scheduler gate function that filters out check jobs
// belonging to users that are paused or have been inactive for too long.
//
// Lookups are cached for a short TTL so the scheduler hot path does not hit
// storage on every job pop.
type UserGater struct {
	resolver               JanitorUserResolver
	defaultInactivityDays  int
	cacheTTL               time.Duration

	mu    sync.Mutex
	cache map[string]gateCacheEntry
}

type gateCacheEntry struct {
	allow   bool
	expires time.Time
}

// NewUserGater creates a UserGater. defaultInactivityDays is used for users
// whose UserQuota.InactivityPauseDays is zero. A negative effective value
// disables inactivity-based pausing for that user.
func NewUserGater(resolver JanitorUserResolver, defaultInactivityDays int) *UserGater {
	return &UserGater{
		resolver:              resolver,
		defaultInactivityDays: defaultInactivityDays,
		cacheTTL:              5 * time.Minute,
		cache:                 map[string]gateCacheEntry{},
	}
}

// Allow returns true if the scheduler should run jobs for the given target.
func (g *UserGater) Allow(target happydns.CheckTarget) bool {
	uid := target.UserId
	if uid == "" || g.resolver == nil {
		return true
	}

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

// Invalidate drops any cached decision for the given user. Call this when a
// user's quota or LastSeen changes (e.g. on login or admin update).
func (g *UserGater) Invalidate(userID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.cache, userID)
}

func (g *UserGater) compute(uid string) bool {
	id, err := happydns.NewIdentifierFromString(uid)
	if err != nil {
		return true
	}
	user, err := g.resolver.GetUser(id)
	if err != nil || user == nil {
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
