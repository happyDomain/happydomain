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
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
	authuserUC "git.happydns.org/happyDomain/internal/usecase/authuser"
	sessionUC "git.happydns.org/happyDomain/internal/usecase/session"
	"git.happydns.org/happyDomain/internal/usecase/user"
	"git.happydns.org/happyDomain/model"
)

type sealedBody struct {
	ApiKey happydns.Secret `json:"apikey"`
}

// Deleting a user deletes their safes: what was sealed in them no longer
// opens, backups included.
func Test_DeleteUserShredsSecrets(t *testing.T) {
	for name, del := range map[string]func(*user.Service, happydns.Identifier) error{
		"DeleteUserByID": (*user.Service).DeleteUserByID,
		"DeleteUser":     (*user.Service).DeleteUser,
	} {
		t.Run(name, func(t *testing.T) {
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

			u := &happydns.User{Id: happydns.Identifier("doomed"), Email: "doomed@example.com"}
			other := &happydns.User{Id: happydns.Identifier("survivor"), Email: "survivor@example.com"}
			for _, x := range []*happydns.User{u, other} {
				if err := db.CreateOrUpdateUser(x); err != nil {
					t.Fatal(err)
				}
			}

			seal := func(owner happydns.Identifier) string {
				body := &sealedBody{ApiKey: happydns.NewSecret("key of " + string(owner))}
				sc := secret.SecretContext{Owner: owner, ObjectType: "provider", ObjectId: "p"}
				if err := secrets.SealObject(context.Background(), sc, body); err != nil {
					t.Fatal(err)
				}
				return body.ApiKey.Token()
			}
			doomedToken := seal(u.Id)
			survivorToken := seal(other.Id)

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

// A safe record that does not decode, corrupted on disk, must not keep every
// account from being deleted.
func Test_DeleteUserGoesPastADamagedSafe(t *testing.T) {
	kvdb, _ := inmemory.NewInMemoryStorage()
	db, err := kv.NewKVDatabase(kvdb)
	if err != nil {
		t.Fatal(err)
	}
	if err := kvdb.Put("safe-damaged", "not a safe"); err != nil {
		t.Fatal(err)
	}

	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	secrets, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})
	sessions := &mockSessionCloser{}
	service := user.NewUserUsecases(db, nil, nil, sessions, secrets)

	u := &happydns.User{Id: happydns.Identifier("doomed"), Email: "doomed@example.com"}
	if err := db.CreateOrUpdateUser(u); err != nil {
		t.Fatal(err)
	}
	body := &sealedBody{ApiKey: happydns.NewSecret("key")}
	if err := secrets.SealObject(context.Background(), secret.SecretContext{Owner: u.Id, ObjectType: "provider", ObjectId: "p"}, body); err != nil {
		t.Fatal(err)
	}

	if err := service.DeleteUserByID(u.Id); err != nil {
		t.Fatalf("DeleteUserByID: %v", err)
	}
	if _, err := db.GetSafeByOwner(u.Id, secret.KindInstance); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("the deleted user's safe is still there: %v", err)
	}
	if len(sessions.closedUserIDs) != 1 {
		t.Errorf("sessions closed for %v, want the deleted user's", sessions.closedUserIDs)
	}
}

type failingShredder struct{}

func (failingShredder) DeleteOwnerSafes(happydns.Identifier) error {
	return errors.New("storage unavailable")
}

// The user record is gone once the safes are being deleted: a failure there
// must still log the user out, and be reported.
func Test_DeleteUserClosesSessionsWhenShreddingFails(t *testing.T) {
	for name, del := range map[string]func(*user.Service, happydns.Identifier) error{
		"DeleteUserByID": (*user.Service).DeleteUserByID,
		"DeleteUser":     (*user.Service).DeleteUser,
	} {
		t.Run(name, func(t *testing.T) {
			db, _ := inmemory.Instantiate()
			sessions := sessionUC.NewService(db)
			authUsers := authuserUC.NewAuthUserUsecases(&happydns.Options{}, &noopMailer{}, db, sessions)
			closer := &mockSessionCloser{}
			service := user.NewUserUsecases(db, nil, authUsers, closer, failingShredder{})

			u := &happydns.User{Id: happydns.Identifier("doomed"), Email: "doomed@example.com"}
			if err := db.CreateOrUpdateUser(u); err != nil {
				t.Fatal(err)
			}

			if err := del(service, u.Id); err == nil {
				t.Error("the failure to delete the safes was not reported")
			}
			if len(closer.closedUserIDs) != 1 {
				t.Errorf("sessions closed for %v, want the deleted user's", closer.closedUserIDs)
			}
		})
	}
}

// A user deleting their local account, wired the way the application wires
// it, deletes their safes too, and not only their credentials.
func Test_DeleteAuthUserShredsSecrets(t *testing.T) {
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
	authUsers.SetOnDeleted(service.DeleteUserByID)

	ua := &happydns.UserAuth{Email: "leaving@example.com"}
	helpers.DefinePassword(ua, "Leaving-Password-123")
	if err := db.CreateAuthUser(ua); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateOrUpdateUser(&happydns.User{Id: ua.Id, Email: ua.Email}); err != nil {
		t.Fatal(err)
	}

	body := &sealedBody{ApiKey: happydns.NewSecret("key")}
	if err := secrets.SealObject(context.Background(), secret.SecretContext{Owner: ua.Id, ObjectType: "provider", ObjectId: "p"}, body); err != nil {
		t.Fatal(err)
	}

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
