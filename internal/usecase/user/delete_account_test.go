// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package user_test

import (
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	authuserUC "git.happydns.org/happyDomain/internal/usecase/authuser"
	sessionUC "git.happydns.org/happyDomain/internal/usecase/session"
	"git.happydns.org/happyDomain/internal/usecase/user"
	"git.happydns.org/happyDomain/model"
)

// A user deleting their local account deletes their profile too: nothing
// else would, accounts authenticated elsewhere having no credentials.
func Test_DeleteAuthUserDeletesProfile(t *testing.T) {
	db, _ := inmemory.Instantiate()
	sessions := sessionUC.NewService(db)
	authUsers := authuserUC.NewAuthUserUsecases(&happydns.Options{}, &noopMailer{}, db, sessions)
	service := user.NewUserUsecases(db, nil, authUsers, &mockSessionCloser{})
	authUsers.SetOnDeleted(service.DeleteUserByID)

	ua := &happydns.UserAuth{Email: "leaving@example.com"}
	helpers.DefinePassword(ua, "Leaving-Password-123")
	if err := db.CreateAuthUser(ua); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateOrUpdateUser(&happydns.User{Id: ua.Id, Email: ua.Email}); err != nil {
		t.Fatal(err)
	}

	if err := authUsers.DeleteAuthUser(ua, "Leaving-Password-123"); err != nil {
		t.Fatalf("DeleteAuthUser: %v", err)
	}

	if _, err := db.GetUser(ua.Id); !errors.Is(err, happydns.ErrUserNotFound) {
		t.Errorf("the deleted account is still there: %v", err)
	}
}
