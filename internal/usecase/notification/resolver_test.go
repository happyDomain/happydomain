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

package notification

import (
	"errors"
	"testing"

	"git.happydns.org/happyDomain/model"
)

type fakePrefStore struct {
	prefs []*happydns.NotificationPreference
	err   error
}

func (f *fakePrefStore) ListPreferencesByUser(_ happydns.Identifier) ([]*happydns.NotificationPreference, error) {
	return f.prefs, f.err
}
func (f *fakePrefStore) GetPreference(_ happydns.Identifier) (*happydns.NotificationPreference, error) {
	return nil, nil
}
func (f *fakePrefStore) CreatePreference(_ *happydns.NotificationPreference) error { return nil }
func (f *fakePrefStore) UpdatePreference(_ *happydns.NotificationPreference) error { return nil }
func (f *fakePrefStore) DeletePreference(_ happydns.Identifier) error              { return nil }

func TestResolvePreferenceFallsBackToDefault(t *testing.T) {
	user := &happydns.User{Id: happydns.Identifier{1}}
	target := happydns.CheckTarget{UserId: user.Id.String(), DomainId: "dom-1"}

	t.Run("no preferences returns opt-in default", func(t *testing.T) {
		r := NewResolver(nil, &fakePrefStore{})
		got := r.ResolvePreference(user, target)
		if got == nil {
			t.Fatal("expected default preference, got nil")
		}
		if !got.Enabled || got.MinStatus != happydns.StatusWarn {
			t.Errorf("default not opt-in: %+v", got)
		}
	})

	t.Run("store error returns default", func(t *testing.T) {
		r := NewResolver(nil, &fakePrefStore{err: errors.New("boom")})
		if got := r.ResolvePreference(user, target); got == nil || !got.Enabled {
			t.Errorf("expected enabled default on store error, got %+v", got)
		}
	})

	t.Run("matching preference wins over default", func(t *testing.T) {
		domId, _ := happydns.NewIdentifierFromString("dom-1")
		user := &happydns.User{Id: happydns.Identifier{1}}
		userPref := &happydns.NotificationPreference{
			DomainId:  &domId,
			Enabled:   true,
			MinStatus: happydns.StatusCrit,
		}
		r := NewResolver(nil, &fakePrefStore{prefs: []*happydns.NotificationPreference{userPref}})
		got := r.ResolvePreference(user, happydns.CheckTarget{DomainId: domId.String()})
		if got != userPref {
			t.Errorf("expected user preference, got %+v", got)
		}
	})

	t.Run("non-matching scoped preference falls back to default", func(t *testing.T) {
		otherDom, _ := happydns.NewIdentifierFromString("dom-other")
		userPref := &happydns.NotificationPreference{
			DomainId: &otherDom,
			Enabled:  true,
		}
		r := NewResolver(nil, &fakePrefStore{prefs: []*happydns.NotificationPreference{userPref}})
		got := r.ResolvePreference(user, happydns.CheckTarget{DomainId: "dom-1"})
		if got == nil || got == userPref {
			t.Errorf("expected default fallback, got %+v", got)
		}
		if !got.Enabled || got.MinStatus != happydns.StatusWarn {
			t.Errorf("default not opt-in: %+v", got)
		}
	})
}
