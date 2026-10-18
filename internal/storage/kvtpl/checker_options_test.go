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

package database_test

import (
	"slices"
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

// Each level is read once, and only when its identifier is given: a nil one
// must not make the previous level read again.
func TestGetCheckerConfigurationLevels(t *testing.T) {
	s := newStorage(t)
	user, _ := happydns.NewRandomIdentifier()
	domain, _ := happydns.NewRandomIdentifier()
	service, _ := happydns.NewRandomIdentifier()

	levels := []struct {
		name          string
		uid, did, sid *happydns.Identifier
	}{
		{"admin", nil, nil, nil},
		{"user", &user, nil, nil},
		{"domain", &user, &domain, nil},
		{"service", &user, &domain, &service},
	}
	for _, l := range levels {
		if err := s.UpdateCheckerConfiguration("levels", l.uid, l.did, l.sid, happydns.CheckerOptions{"level": l.name}); err != nil {
			t.Fatal(err)
		}
	}

	for _, l := range levels {
		pos, err := s.GetCheckerConfiguration("levels", l.uid, l.did, l.sid)
		if err != nil {
			t.Fatalf("%s: GetCheckerConfiguration: %v", l.name, err)
		}
		var got []string
		for _, p := range pos {
			got = append(got, p.Options["level"].(string))
		}

		var want []string
		for _, w := range levels {
			want = append(want, w.name)
			if w.name == l.name {
				break
			}
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: levels = %v, want %v", l.name, got, want)
		}
	}
}
