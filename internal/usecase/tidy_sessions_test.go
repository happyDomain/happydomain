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

package usecase_test

import (
	"testing"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase"
	"git.happydns.org/happyDomain/model"
)

type scopePurger struct{ alive map[string]bool }

func (p *scopePurger) PurgeScopes(alive func(string) bool) {
	for _, scope := range []string{"live", "gone"} {
		p.alive[scope] = alive(scope)
	}
}

// Tidying the sessions forgets the keystores unlocked during those gone.
func TestTidySessionsPurgesUnlockedKeystores(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	user := &happydns.User{Id: happydns.Identifier{0x01}, Email: "u@example.com"}
	if err := db.CreateOrUpdateUser(user); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateSession(&happydns.Session{Id: "live", IdUser: user.Id}); err != nil {
		t.Fatal(err)
	}

	p := &scopePurger{alive: map[string]bool{}}
	if err := usecase.NewTidyUpUsecase(db, usecase.TidyUnlockedKeystores(p)).TidySessions(false); err != nil {
		t.Fatal(err)
	}
	if !p.alive["live"] || p.alive["gone"] {
		t.Errorf("alive = %v, want only the session stored", p.alive)
	}
}
