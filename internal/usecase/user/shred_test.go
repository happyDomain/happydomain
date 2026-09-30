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

package user_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	authuserUC "git.happydns.org/happyDomain/internal/usecase/authuser"
	sessionUC "git.happydns.org/happyDomain/internal/usecase/session"
	"git.happydns.org/happyDomain/internal/usecase/user"
	"git.happydns.org/happyDomain/model"
)

type sealedBody struct {
	ApiKey happydns.Secret `json:"apikey"`
}

// newShredFixture wires an in-memory store, a Manager under PolicyInstance,
// and the authuser and user services the way the application does.
func newShredFixture(t *testing.T) (storage.Storage, *secret.Manager, *authuserUC.Service, *user.Service) {
	t.Helper()
	db, _ := inmemory.Instantiate()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	secrets, err := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})
	if err != nil {
		t.Fatal(err)
	}

	sessions := sessionUC.NewService(db)
	authUsers := authuserUC.NewAuthUserUsecases(&happydns.Options{}, &noopMailer{}, db, sessions)
	service := user.NewUserUsecases(db, nil, authUsers, &mockSessionCloser{}, secrets)
	return db, secrets, authUsers, service
}

// sealFor seals a secret of owner and returns its token.
func sealFor(t *testing.T, secrets *secret.Manager, owner happydns.Identifier) string {
	t.Helper()
	body := &sealedBody{ApiKey: happydns.NewSecret("key of " + string(owner))}
	sc := secret.SecretContext{Owner: owner, ObjectType: "provider", ObjectId: "p"}
	if err := secrets.SealObject(context.Background(), sc, body); err != nil {
		t.Fatal(err)
	}
	return body.ApiKey.Token()
}

// Deleting a user deletes their safes: what was sealed in them no longer
// opens from the database. A backup taken before still holds the safes.
func Test_DeleteUserShredsSecrets(t *testing.T) {
	for name, del := range map[string]func(*user.Service, happydns.Identifier) error{
		"DeleteUserByID": (*user.Service).DeleteUserByID,
		"DeleteUser":     (*user.Service).DeleteUser,
	} {
		t.Run(name, func(t *testing.T) {
			db, secrets, _, service := newShredFixture(t)

			u := &happydns.User{Id: happydns.Identifier("doomed"), Email: "doomed@example.com"}
			other := &happydns.User{Id: happydns.Identifier("survivor"), Email: "survivor@example.com"}
			for _, x := range []*happydns.User{u, other} {
				if err := db.CreateOrUpdateUser(x); err != nil {
					t.Fatal(err)
				}
			}

			doomedToken := sealFor(t, secrets, u.Id)
			survivorToken := sealFor(t, secrets, other.Id)

			if err := del(service, u.Id); err != nil {
				t.Fatalf("%s: %v", name, err)
			}

			if _, err := db.GetSafeByOwner(u.Id, secret.KindInstance); !errors.Is(err, happydns.ErrSafeNotFound) {
				t.Errorf("the deleted user's safe is still there: %v", err)
			}

			open := func(owner happydns.Identifier, token string) error {
				var body sealedBody
				if err := json.Unmarshal([]byte(`{"apikey":"`+token+`"}`), &body); err != nil {
					t.Fatal(err)
				}
				return secrets.OpenObject(context.Background(), secret.SecretContext{Owner: owner, ObjectType: "provider", ObjectId: "p"}, &body)
			}
			if err := open(u.Id, doomedToken); !errors.Is(err, secret.ErrUnknownSafe) {
				t.Errorf("a secret of the deleted user still opens: %v", err)
			}
			if err := open(other.Id, survivorToken); err != nil {
				t.Errorf("another user's secret no longer opens: %v", err)
			}
		})
	}
}

// A user deleting their local account, wired the way the application wires
// it, deletes their safes too, and not only their credentials.
func Test_DeleteAuthUserShredsSecrets(t *testing.T) {
	db, secrets, authUsers, service := newShredFixture(t)
	authUsers.SetOnDeleted(service.DeleteUserByID)

	ua := &happydns.UserAuth{Email: "leaving@example.com"}
	helpers.DefinePassword(ua, "Leaving-Password-123")
	if err := db.CreateAuthUser(ua); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateOrUpdateUser(&happydns.User{Id: ua.Id, Email: ua.Email}); err != nil {
		t.Fatal(err)
	}

	sealFor(t, secrets, ua.Id)

	if err := authUsers.DeleteAuthUser(ua, "Leaving-Password-123"); err != nil {
		t.Fatalf("DeleteAuthUser: %v", err)
	}

	if _, err := db.GetSafeByOwner(ua.Id, secret.KindInstance); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("the safe of the deleted account is still there: %v", err)
	}
	if _, err := db.GetUser(ua.Id); !errors.Is(err, happydns.ErrUserNotFound) {
		t.Errorf("the deleted account is still there: %v", err)
	}
}
