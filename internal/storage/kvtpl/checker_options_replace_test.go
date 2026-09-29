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
	"errors"
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

func TestReplaceCheckerConfiguration(t *testing.T) {
	s := newStorage(t)
	user, _ := happydns.NewRandomIdentifier()
	if err := s.UpdateCheckerConfiguration("c", &user, nil, nil, happydns.CheckerOptions{"k": "before", "other": "kept"}); err != nil {
		t.Fatal(err)
	}

	err := s.ReplaceCheckerConfiguration("c", &user, nil, nil, func(opts happydns.CheckerOptions) (happydns.CheckerOptions, error) {
		if opts["k"] != "before" {
			t.Errorf("update got %v, want the stored options", opts)
		}
		opts["k"] = "after"
		return opts, nil
	})
	if err != nil {
		t.Fatalf("ReplaceCheckerConfiguration: %v", err)
	}
	// GetCheckerConfiguration returns every level up to this scope.
	got, _ := s.GetCheckerConfiguration("c", &user, nil, nil)
	for _, p := range got {
		if p.UserId == nil || !p.UserId.Equals(user) || p.Options["k"] != "after" || p.Options["other"] != "kept" {
			t.Errorf("stored = %+v, want the replaced options in the same scope", p)
		}
	}
	if list, _ := s.ListCheckerConfiguration("c"); len(list) != 1 {
		t.Errorf("the checker lists %d scopes, want 1", len(list))
	}

	// Deleted meanwhile: not brought back.
	err = s.ReplaceCheckerConfiguration("c", &user, nil, nil, func(opts happydns.CheckerOptions) (happydns.CheckerOptions, error) {
		if err := s.DeleteCheckerConfiguration("c", &user, nil, nil); err != nil {
			t.Fatal(err)
		}
		return opts, nil
	})
	if !errors.Is(err, happydns.ErrNotFound) {
		t.Errorf("ReplaceCheckerConfiguration over a deletion = %v, want ErrNotFound", err)
	}
	if got, _ := s.GetCheckerConfiguration("c", &user, nil, nil); len(got) != 0 {
		t.Errorf("deleted options were brought back: %+v", got)
	}
}
