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

package provider_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase"
	"git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
)

// extractionWorld is an instance with providers, a personal keystore and a
// keystore of references, real storage, manager and usecases.
type extractionWorld struct {
	db       storage.Storage
	svc      *provider.Service
	keystore *usecase.KeystoreUsecase
	manager  *secret.Manager
	user     *happydns.User
	personal secret.Keystore
	refs     secret.Keystore
}

func newExtractionWorld(t *testing.T, allowExtraction bool) *extractionWorld {
	t.Helper()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	cfg := secrettest.WithReferenceStore(secrettest.WithPersonalStore(secret.Config{
		Policy:          secret.PolicyInstance,
		InstanceKey:     key,
		Safes:           db,
		Defaults:        db,
		Stores:          []string{secret.KindInstance},
		AllowExtraction: allowExtraction,
	}))
	m, err := secret.NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}

	w := &extractionWorld{db: db, manager: m, user: createTestUser(t, db, "extraction@example.com")}
	w.svc = provider.NewService(db, &mockValidator{}, nil, nil, m)
	w.keystore = usecase.NewKeystoreUsecase(m, map[string]usecase.SecretHolder{"provider": w.svc})
	if allowExtraction {
		w.keystore.WithExtraction()
	}

	if w.personal, err = w.keystore.Create(ctx, w.user.Id, secrettest.PersonalKind, "Mine", secrettest.PersonalParams()); err != nil {
		t.Fatal(err)
	}
	values := map[string]string{"ref-a": "s3cr3t-a", "ref-b": "s3cr3t-b", "ref-c": "s3cr3t-c"}
	if w.refs, err = w.keystore.Create(ctx, w.user.Id, secrettest.ReferenceKind, "My file", secrettest.ReferenceParams(values)); err != nil {
		t.Fatal(err)
	}
	return w
}

// create stores a provider whose credential is apikey, in the keystore ref
// ("" for the default one).
func (w *extractionWorld) create(t *testing.T, ref, apikey string) happydns.Identifier {
	t.Helper()
	c := ctx
	if ref != "" {
		c = secret.WithKeystore(ctx, ref)
	}
	p, err := w.svc.CreateProvider(c, w.user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"`+apikey+`"}`))
	if err != nil {
		t.Fatalf("CreateProvider(%q): %v", ref, err)
	}
	return p.Id
}

// echoInto is the update a client does with ?keystore=ref, sending the
// credential back as the placeholder it received.
func (w *extractionWorld) echoInto(id happydns.Identifier, ref string) error {
	body := `{"host":"h","apikey":"` + happydns.RedactedSecret + `"}`
	msg := &happydns.ProviderMessage{ProviderMeta: happydns.ProviderMeta{Type: "SecretTestProvider", Comment: "secret test"}, Provider: json.RawMessage(body)}
	_, err := w.svc.UpdateProviderFromMessage(secret.WithKeystore(ctx, ref), id, w.user, msg)
	return err
}

// safeOf returns the safe the credential of the provider is sealed in, and
// the credential once opened.
func (w *extractionWorld) safeOf(t *testing.T, id happydns.Identifier) (happydns.Identifier, string) {
	t.Helper()
	msg, err := w.db.GetProvider(id)
	if err != nil {
		t.Fatal(err)
	}
	p, err := provider.ParseProvider(msg)
	if err != nil {
		t.Fatal(err)
	}
	body := p.Provider.(*SecretTestProvider)
	sealed, err := secret.ParseSealed(body.ApiKey.Token())
	if err != nil {
		t.Fatalf("credential not sealed: %v", err)
	}
	if err := w.manager.OpenObject(ctx, provider.SecretContext(p), p.Provider); err != nil {
		t.Fatalf("OpenObject: %v", err)
	}
	return sealed.SafeId, body.ApiKey.Reveal()
}

func (w *extractionWorld) safeIdOf(t *testing.T, ref string) happydns.Identifier {
	t.Helper()
	id, err := w.manager.KeystoreSafe(ctx, w.user.Id, ref)
	if err != nil {
		t.Fatalf("KeystoreSafe(%s): %v", ref, err)
	}
	return id
}

func status(err error) int {
	var he interface{ HTTPStatus() int }
	if errors.As(err, &he) {
		return he.HTTPStatus()
	}
	return 0
}

// Credentials that are not references move between keystores whether the
// administrator allows extraction or not, by migration and by choosing a
// keystore when saving.
func TestE2E_PlainSecretsMoveWhateverTheOption(t *testing.T) {
	for _, allow := range []bool{false, true} {
		name := "extraction disabled"
		if allow {
			name = "extraction enabled"
		}
		t.Run(name, func(t *testing.T) {
			w := newExtractionWorld(t, allow)
			a := w.create(t, "", "plain-a")
			b := w.create(t, "", "plain-b")
			instance := w.safeIdOf(t, secret.KindInstance)
			mine := w.safeIdOf(t, w.personal.Ref)

			// Migration out of the instance keystore, and back.
			if _, err := w.keystore.Migrate(ctx, w.user.Id, secret.KindInstance, w.personal.Ref); err != nil {
				t.Fatalf("Migrate(instance -> personal) = %v", err)
			}
			for _, tc := range []struct {
				id   happydns.Identifier
				want string
			}{{a, "plain-a"}, {b, "plain-b"}} {
				if safe, clear := w.safeOf(t, tc.id); !safe.Equals(mine) || clear != tc.want {
					t.Errorf("after migration: safe %s, secret %q; want the personal keystore and %q", safe, clear, tc.want)
				}
			}
			if _, err := w.keystore.Migrate(ctx, w.user.Id, w.personal.Ref, secret.KindInstance); err != nil {
				t.Fatalf("Migrate(personal -> instance) = %v", err)
			}
			if safe, clear := w.safeOf(t, a); !safe.Equals(instance) || clear != "plain-a" {
				t.Errorf("migrated back: safe %s, secret %q", safe, clear)
			}

			// ?keystore= on a save, both ways.
			if err := w.echoInto(a, w.personal.Ref); err != nil {
				t.Fatalf("save with keystore=personal = %v", err)
			}
			if safe, clear := w.safeOf(t, a); !safe.Equals(mine) || clear != "plain-a" {
				t.Errorf("saved into personal: safe %s, secret %q", safe, clear)
			}
			if err := w.echoInto(a, secret.KindInstance); err != nil {
				t.Fatalf("save with keystore=instance = %v", err)
			}
			if safe, clear := w.safeOf(t, a); !safe.Equals(instance) || clear != "plain-a" {
				t.Errorf("saved into instance: safe %s, secret %q", safe, clear)
			}
		})
	}
}

// A credential kept as a reference leaves its keystore only when the
// administrator allows it, whichever way the move is asked.
func TestE2E_ReferencesLeaveOnlyWhenExtractionIsEnabled(t *testing.T) {
	for _, allow := range []bool{false, true} {
		name := "extraction disabled"
		if allow {
			name = "extraction enabled"
		}
		t.Run(name, func(t *testing.T) {
			w := newExtractionWorld(t, allow)
			viaSave := w.create(t, w.refs.Ref, "ref-a")
			viaSavePersonal := w.create(t, w.refs.Ref, "ref-b")
			viaMigrate := w.create(t, w.refs.Ref, "ref-c")
			refsSafe := w.safeIdOf(t, w.refs.Ref)
			for _, id := range []happydns.Identifier{viaSave, viaSavePersonal, viaMigrate} {
				if safe, _ := w.safeOf(t, id); !safe.Equals(refsSafe) {
					t.Fatalf("provider created outside the keystore of references: %s", safe)
				}
			}

			errSave := w.echoInto(viaSave, secret.KindInstance)
			errSavePersonal := w.echoInto(viaSavePersonal, w.personal.Ref)
			_, errMigrate := w.keystore.Migrate(ctx, w.user.Id, w.refs.Ref, secret.KindInstance)

			if !allow {
				for what, err := range map[string]error{"save into instance": errSave, "save into personal": errSavePersonal} {
					if status(err) != http.StatusBadRequest || !strings.Contains(fmt.Sprint(err), secret.ErrExtractionForbidden.Error()) {
						t.Errorf("%s = %v, want it refused", what, err)
					}
				}
				if status(errMigrate) != http.StatusForbidden {
					t.Errorf("Migrate = %v, want a 403", errMigrate)
				}
				// Nothing was resolved: the credentials are where they were.
				for _, tc := range []struct {
					id   happydns.Identifier
					want string
				}{{viaSave, "s3cr3t-a"}, {viaSavePersonal, "s3cr3t-b"}, {viaMigrate, "s3cr3t-c"}} {
					if safe, clear := w.safeOf(t, tc.id); !safe.Equals(refsSafe) || clear != tc.want {
						t.Errorf("after refusals: safe %s, secret %q; want it still in the keystore of references", safe, clear)
					}
				}
				return
			}

			if errSave != nil || errSavePersonal != nil || errMigrate != nil {
				t.Fatalf("with extraction enabled: save %v, save %v, Migrate %v", errSave, errSavePersonal, errMigrate)
			}
			instance := w.safeIdOf(t, secret.KindInstance)
			mine := w.safeIdOf(t, w.personal.Ref)
			for _, tc := range []struct {
				id    happydns.Identifier
				safe  happydns.Identifier
				clear string
			}{{viaSave, instance, "s3cr3t-a"}, {viaSavePersonal, mine, "s3cr3t-b"}, {viaMigrate, instance, "s3cr3t-c"}} {
				if safe, clear := w.safeOf(t, tc.id); !safe.Equals(tc.safe) || clear != tc.clear {
					t.Errorf("extracted: safe %s, secret %q; want safe %s and %q", safe, clear, tc.safe, tc.clear)
				}
			}
		})
	}
}
