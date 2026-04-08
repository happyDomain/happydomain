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

	g := NewUserGater(r, 90)
	target := happydns.CheckTarget{UserId: uid}

	if !g.Allow(target) {
		t.Error("expected active user to be allowed")
	}
}

func TestUserGater_SchedulingPaused(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{SchedulingPaused: true}, time.Now())

	g := NewUserGater(r, 90)
	target := happydns.CheckTarget{UserId: uid}

	if g.Allow(target) {
		t.Error("expected paused user to be blocked")
	}
}

func TestUserGater_InactiveUser(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{}, time.Now().AddDate(0, 0, -100))

	g := NewUserGater(r, 90)
	target := happydns.CheckTarget{UserId: uid}

	if g.Allow(target) {
		t.Error("expected inactive user (100 days) to be blocked with 90-day threshold")
	}
}

func TestUserGater_InactiveUserWithinThreshold(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{}, time.Now().AddDate(0, 0, -30))

	g := NewUserGater(r, 90)
	target := happydns.CheckTarget{UserId: uid}

	if !g.Allow(target) {
		t.Error("expected user seen 30 days ago to be allowed with 90-day threshold")
	}
}

func TestUserGater_PerUserInactivityOverride(t *testing.T) {
	r := newGateResolver()
	// User has custom 14-day inactivity threshold, last seen 20 days ago.
	uid := addGateUser(r, happydns.UserQuota{InactivityPauseDays: 14}, time.Now().AddDate(0, 0, -20))

	g := NewUserGater(r, 90)
	target := happydns.CheckTarget{UserId: uid}

	if g.Allow(target) {
		t.Error("expected user with 14-day override to be blocked after 20 days")
	}
}

func TestUserGater_NegativeInactivityDaysDisablesCheck(t *testing.T) {
	r := newGateResolver()
	// User opts out of inactivity pause with negative value, last seen 1 year ago.
	uid := addGateUser(r, happydns.UserQuota{InactivityPauseDays: -1}, time.Now().AddDate(-1, 0, 0))

	g := NewUserGater(r, 90)
	target := happydns.CheckTarget{UserId: uid}

	if !g.Allow(target) {
		t.Error("expected negative InactivityPauseDays to disable inactivity check")
	}
}

func TestUserGater_ZeroDefaultInactivityDisablesCheck(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{}, time.Now().AddDate(-1, 0, 0))

	g := NewUserGater(r, 0) // system default disabled
	target := happydns.CheckTarget{UserId: uid}

	if !g.Allow(target) {
		t.Error("expected zero defaultInactivityDays to disable inactivity check")
	}
}

func TestUserGater_NegativeDefaultInactivityDisablesCheck(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{}, time.Now().AddDate(-1, 0, 0))

	g := NewUserGater(r, -1)
	target := happydns.CheckTarget{UserId: uid}

	if !g.Allow(target) {
		t.Error("expected negative defaultInactivityDays to disable inactivity check")
	}
}

func TestUserGater_ZeroLastSeenAllowed(t *testing.T) {
	r := newGateResolver()
	uid := addGateUser(r, happydns.UserQuota{}, time.Time{})

	g := NewUserGater(r, 90)
	target := happydns.CheckTarget{UserId: uid}

	if !g.Allow(target) {
		t.Error("expected zero LastSeen to be allowed (user never logged in yet)")
	}
}

func TestUserGater_UnknownUserAllowed(t *testing.T) {
	r := newGateResolver()
	uid, _ := happydns.NewRandomIdentifier()

	g := NewUserGater(r, 90)
	target := happydns.CheckTarget{UserId: uid.String()}

	if !g.Allow(target) {
		t.Error("expected unknown user to be allowed (fail-open)")
	}
}

func TestUserGater_EmptyUserIdAllowed(t *testing.T) {
	r := newGateResolver()
	g := NewUserGater(r, 90)
	target := happydns.CheckTarget{UserId: ""}

	if !g.Allow(target) {
		t.Error("expected empty UserId to be allowed")
	}
}

func TestUserGater_NilResolverAllowed(t *testing.T) {
	g := NewUserGater(nil, 90)
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

	g := NewUserGater(r, 90)
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

	g := NewUserGater(r, 90)
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

	g := NewUserGater(r, 90)
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
