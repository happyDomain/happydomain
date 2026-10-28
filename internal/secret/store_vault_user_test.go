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

package secret

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

// loopbackGuard lets the stores reach the fake Vault, on the loopback.
func loopbackGuard(t *testing.T) *netguard.Guard {
	t.Helper()
	g, err := netguard.New("outbound", "-outbound-allowed-target", []string{"127.0.0.0/8", "::1/128"})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func vaultUserParams(addr, token string) json.RawMessage {
	b, _ := json.Marshal(map[string]string{vaultAddressParam: addr, vaultTokenParam: token})
	return b
}

func newTestVaultUserStore(t *testing.T) *vaultUserStore {
	t.Helper()
	store, err := newVaultUserStore(Config{Outbound: loopbackGuard(t)})
	if err != nil {
		t.Fatal(err)
	}
	return store.(*vaultUserStore)
}

func TestVaultUserStoreCreate(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	v.addToken("limited-user-token", map[string]any{"num_uses": 5})
	v.addToken("fixed-user-token", map[string]any{"renewable": false})
	v.addToken("batch-user-token", map[string]any{"renewable": false, "type": "batch"})
	v.addToken("forever-user-token", map[string]any{"ttl": 0, "renewable": false})
	store := newTestVaultUserStore(t)

	for name, params := range map[string]json.RawMessage{
		"no address":        vaultUserParams("", "user-token"),
		"bad address":       vaultUserParams("vault:8200", "user-token"),
		"no token":          vaultUserParams(v.URL, ""),
		"bad token":         vaultUserParams(v.URL, "wrong-user-token"),
		"limited uses":      vaultUserParams(v.URL, "limited-user-token"),
		"expiring for good": vaultUserParams(v.URL, "fixed-user-token"),
		"batch token":       vaultUserParams(v.URL, "batch-user-token"),
		"bad mount":         json.RawMessage(`{"address":"` + v.URL + `","token":"user-token","mount":"../sys"}`),
		"malformed":         json.RawMessage(`nope`),
	} {
		safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: happydns.Identifier{0x01}, Kind: KindVaultUser}
		if err := store.Create(ctx, safe, params); err == nil {
			t.Errorf("Create(%s) succeeded", name)
		} else if strings.Contains(err.Error(), "user-token") {
			t.Errorf("Create(%s): the error repeats the token: %v", name, err)
		}
	}

	// A token never expiring is accepted, though Vault does not renew it.
	forever := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: happydns.Identifier{0x01}, Kind: KindVaultUser}
	if err := store.Create(ctx, forever, vaultUserParams(v.URL, "forever-user-token")); err != nil {
		t.Errorf("Create(token never expiring) = %v", err)
	}

	// The address is checked as the endpoint of a provider is: by default,
	// no address that is not public.
	guarded, err := newVaultUserStore(Config{})
	if err != nil {
		t.Fatal(err)
	}
	safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: happydns.Identifier{0x01}, Kind: KindVaultUser}
	if err := guarded.(Creator).Create(ctx, safe, vaultUserParams(v.URL, "user-token")); !errors.Is(err, netguard.ErrBlocked) {
		t.Errorf("Create(loopback address) = %v, want ErrBlocked", err)
	}
}

// The token is renewed when the keystore is unlocked, once two thirds of its
// lease are gone, and not again before; a renewal that fails does not stop
// the keystore from opening, and is tried again later.
func TestVaultUserStoreRenewsAtUnlock(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	store := newTestVaultUserStore(t)
	now := time.Now()
	store.now = func() time.Time { return now }
	safe := vaultSafe(t, store, happydns.Identifier{0x01}, KindVaultUser, vaultUserParams(v.URL, "user-token"))

	unlock := func(want int, why string) {
		t.Helper()
		if _, err := store.UnlockWithSecret(ctx, safe, []byte("user-token")); err != nil {
			t.Fatalf("Unlock (%s): %v", why, err)
		}
		if got := v.renewAttempts(); got != want {
			t.Errorf("%s: %d renewals asked, want %d", why, got, want)
		}
	}

	unlock(1, "first unlock")
	unlock(1, "unlocked again at once")
	now = now.Add(39 * time.Minute)
	unlock(1, "before two thirds of the lease")
	now = now.Add(2 * time.Minute)
	unlock(2, "after two thirds of the lease")

	// Each safe is renewed on its own.
	other := &happydns.Safe{Id: happydns.Identifier{0x07}, Owner: happydns.Identifier{0x02}, Kind: KindVaultUser}
	if err := store.Create(ctx, other, vaultUserParams(v.URL, "user-token")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UnlockWithSecret(ctx, other, []byte("user-token")); err != nil {
		t.Fatal(err)
	}
	if got := v.renewAttempts(); got != 3 {
		t.Errorf("another safe: %d renewals asked, want 3", got)
	}

	// Vault no longer knows the token: the renewal fails, the keystore
	// opens all the same, and the renewal is not asked again at once.
	v.revoke("user-token")
	now = now.Add(41 * time.Minute)
	unlock(4, "token revoked")
	v.addToken("user-token", nil)
	unlock(4, "unlocked again after a failure")
	now = now.Add(vaultRenewRetry)
	unlock(5, "after the retry delay")
	if _, renews := v.counts(); renews != 4 {
		t.Errorf("%d renewals granted, want 4", renews)
	}
}

// A token Vault does not renew, such as one never expiring, is not asked to
// again.
func TestVaultUserStoreStopsRenewingWhatVaultDoesNot(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	v.addToken("forever-user-token", map[string]any{"ttl": 0, "renewable": false})
	store := newTestVaultUserStore(t)
	now := time.Now()
	store.now = func() time.Time { return now }
	safe := vaultSafe(t, store, happydns.Identifier{0x01}, KindVaultUser, vaultUserParams(v.URL, "forever-user-token"))

	for range 3 {
		if _, err := store.UnlockWithSecret(ctx, safe, []byte("forever-user-token")); err != nil {
			t.Fatal(err)
		}
		now = now.Add(24 * time.Hour)
	}
	if got := v.renewAttempts(); got != 1 {
		t.Errorf("%d renewals asked, want 1", got)
	}
}

// The address is checked at each request too, not only at creation: a
// keystore whose Vault the guard no longer allows does not read.
func TestVaultUserStoreGuardsEachRequest(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	v.put("dns/ovh", map[string]any{"app_secret": "s3cr3t"})
	safe := vaultSafe(t, newTestVaultUserStore(t), happydns.Identifier{0x01}, KindVaultUser, vaultUserParams(v.URL, "user-token"))

	guarded, err := newVaultUserStore(Config{})
	if err != nil {
		t.Fatal(err)
	}
	before := v.renewAttempts()
	u, err := guarded.(*vaultUserStore).UnlockWithSecret(ctx, safe, []byte("user-token"))
	if err != nil {
		t.Fatalf("Unlock = %v, want the keystore to open, the renewal failing quietly", err)
	}
	if _, err := u.Open(ctx, objectContext(), []byte("dns/ovh#app_secret")); !errors.Is(err, netguard.ErrBlocked) {
		t.Errorf("Open(loopback address) = %v, want ErrBlocked", err)
	}
	if got := v.renewAttempts(); got != before {
		t.Errorf("the renewal reached a Vault the guard refuses: %d attempts, want %d", got, before)
	}
}

func TestVaultUserStoreUnlockRefusesMalformedSafes(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	store := newTestVaultUserStore(t)
	good := vaultSafe(t, store, happydns.Identifier{0x01}, KindVaultUser, vaultUserParams(v.URL, "user-token"))
	if _, err := store.UnlockWithSecret(ctx, good, []byte("user-token")); err != nil {
		t.Fatalf("UnlockWithSecret = %v", err)
	}
	if bytes.Contains(good.Keyring[0].Params, []byte("user-token")) || len(good.Keyring[0].Blob) != 0 {
		t.Error("the keyring keeps the token: the Manager seals it as the secret of the safe")
	}
	if _, err := store.Unlock(ctx, good); err == nil {
		t.Error("Unlock opened without the token")
	}

	otherKind := *good
	otherKind.Kind = KindInstance
	noEntry := *good
	noEntry.Keyring = []happydns.WrappedKeyset{{KEK: KindInstance}}
	badParams := *good
	badParams.Keyring = []happydns.WrappedKeyset{{KEK: KindVaultUser, Params: json.RawMessage(`nope`)}}

	for name, safe := range map[string]*happydns.Safe{
		"another kind":      &otherKind,
		"no vault-user key": &noEntry,
		"bad parameters":    &badParams,
	} {
		if _, err := store.UnlockWithSecret(ctx, safe, []byte("user-token")); err == nil {
			t.Errorf("UnlockWithSecret(%s) succeeded", name)
		}
	}
	if _, err := store.UnlockWithSecret(ctx, good, nil); err == nil {
		t.Error("UnlockWithSecret(no token) succeeded")
	}
}

// The mount and namespace given at creation are those read from.
func TestVaultUserStoreReadsItsMountAndNamespace(t *testing.T) {
	ctx := context.Background()
	var gotPath, gotNS string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/token/lookup-self":
			w.Write([]byte(`{"data":{"ttl":0,"renewable":false,"num_uses":0}}`))
		case "/v1/auth/token/renew-self":
			http.Error(w, `{"errors":["lease is not renewable"]}`, http.StatusBadRequest)
		default:
			gotPath, gotNS = r.URL.Path, r.Header.Get("X-Vault-Namespace")
			w.Write([]byte(`{"data":{"data":{"k":"v"}}}`))
		}
	}))
	t.Cleanup(srv.Close)
	store := newTestVaultUserStore(t)
	safe := vaultSafe(t, store, happydns.Identifier{0x01}, KindVaultUser,
		json.RawMessage(`{"address":"`+srv.URL+`","token":"user-token","mount":"kv/team","namespace":"ops"}`))

	u, err := store.UnlockWithSecret(ctx, safe, []byte("user-token"))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := u.Open(ctx, objectContext(), []byte("dns/ovh#k")); err != nil || string(got) != "v" {
		t.Fatalf("Open = %q, %v", got, err)
	}
	if gotPath != "/v1/kv/team/data/dns/ovh" || gotNS != "ops" {
		t.Errorf("read %q in namespace %q, want /v1/kv/team/data/dns/ovh in ops", gotPath, gotNS)
	}
}

// A renewal Vault grants without a lease is not asked again either.
func TestVaultUserStoreStopsRenewingWithoutLease(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	v.lease = 0
	store := newTestVaultUserStore(t)
	now := time.Now()
	store.now = func() time.Time { return now }
	safe := vaultSafe(t, store, happydns.Identifier{0x01}, KindVaultUser, vaultUserParams(v.URL, "user-token"))

	for range 3 {
		if _, err := store.UnlockWithSecret(ctx, safe, []byte("user-token")); err != nil {
			t.Fatal(err)
		}
		now = now.Add(24 * time.Hour)
	}
	if got := v.renewAttempts(); got != 1 {
		t.Errorf("%d renewals asked, want 1", got)
	}
}

func TestVaultUserKindIsBuiltin(t *testing.T) {
	if _, ok := builtinStores[KindVaultUser]; !ok {
		t.Fatalf("no builtin store %q", KindVaultUser)
	}
	store := Store(newTestVaultUserStore(t))
	for name, ok := range map[string]bool{
		"Creator":       is[Creator](store),
		"Updater":       is[Updater](store),
		"Referencing":   is[Referencing](store),
		"SecretKeeping": is[SecretKeeping](store),
	} {
		if !ok {
			t.Errorf("vault-user is not a %s", name)
		}
	}
	if is[Interactive](store) {
		t.Error("vault-user is Interactive")
	}
	if is[KeyReporter](store) || is[Rewrapper](store) {
		t.Error("vault-user has keys of its own: its token is sealed as any secret")
	}
	if is[KEK](store) {
		t.Error("vault-user is a KEK: its entry would be taken for a copy of a key")
	}
	if !opensUnattended(store) {
		t.Error("vault-user does not open unattended")
	}
}

func is[T any](store Store) bool {
	_, ok := store.(T)
	return ok
}

// Through the Manager: created, sealed into, opened, rewrapped after a
// rotation, and counted under the instance keys.
func TestVaultUserKeystoreThroughManager(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	v.put("dns/ovh", map[string]any{"app_secret": "s3cr3t"})
	h, _ := GenerateInstanceKeyset()
	key, _ := NewInstanceKey(h)
	safes := newMemSafeStorage()
	guard := loopbackGuard(t)
	manager := func(key *InstanceKey) *Manager {
		m, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: key, Safes: safes, Defaults: newMemDefaults(), Outbound: guard, Stores: []string{KindInstance, KindVaultUser}})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	m := manager(key)
	sc := objectContext()

	ks, err := m.CreateKeystore(ctx, sc.Owner, KindVaultUser, "My Vault", vaultUserParams(v.URL, "user-token"))
	if err != nil {
		t.Fatal(err)
	}
	obj := &managedObject{ApiKey: happydns.NewSecret("dns/ovh#app_secret")}
	if err := m.SealObjectInto(ctx, sc, obj, Destination{Keystore: ks.Ref}); err != nil {
		t.Fatalf("SealObjectInto: %v", err)
	}

	rh, _ := RotateInstanceKeyset(h)
	rotated, _ := NewInstanceKey(rh)
	m = manager(rotated)
	report, err := m.Rewrap(ctx)
	if err != nil || report.Changed != 1 {
		t.Errorf("Rewrap = %+v, %v; want the vault-user safe rewrapped", report, err)
	}
	status, err := m.KeyStatus()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, k := range status {
		if k.KEK == KindVaultUser {
			t.Errorf("KeyStatus lists %+v under vault-user", k)
		}
		if k.KEK == KindInstance && k.Id == rotated.PrimaryKeyId() && k.Safes == 1 {
			found = true
		}
	}
	if !found {
		t.Errorf("KeyStatus = %+v, want the vault-user safe counted under the primary instance key", status)
	}

	back := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := m.OpenObject(ctx, sc, back); err != nil || back.ApiKey.Reveal() != "s3cr3t" {
		t.Errorf("OpenObject = %q, %v", back.ApiKey.Reveal(), err)
	}
}

// Its owner replaces the token of a keystore, as when Vault no longer knows
// it: the secrets it holds, references, open again with the new one.
func TestVaultUserKeystoreUpdate(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	v.put("dns/ovh", map[string]any{"app_secret": "s3cr3t"})
	safes := newMemSafeStorage()
	m, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: safes, Defaults: newMemDefaults(), Outbound: loopbackGuard(t), Stores: []string{KindInstance, KindVaultUser}})
	if err != nil {
		t.Fatal(err)
	}
	sc := objectContext()
	ks, err := m.CreateKeystore(ctx, sc.Owner, KindVaultUser, "My Vault", json.RawMessage(`{"address":"`+v.URL+`","token":"user-token","mount":"secret","namespace":"team"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(ks.UpdateForm) != 1 || ks.UpdateForm[0].Id != vaultTokenParam || !ks.UpdateForm[0].Secret {
		t.Errorf("UpdateForm = %+v, want the token alone", ks.UpdateForm)
	}
	obj := &managedObject{ApiKey: happydns.NewSecret("dns/ovh#app_secret")}
	if err := m.SealObjectInto(ctx, sc, obj, Destination{Keystore: ks.Ref}); err != nil {
		t.Fatal(err)
	}
	open := func() error {
		back := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
		if err := m.OpenObject(ctx, sc, back); err != nil {
			return err
		}
		if back.ApiKey.Reveal() != "s3cr3t" {
			return errors.New("opened to " + back.ApiKey.Reveal())
		}
		return nil
	}

	v.revoke("user-token")
	if err := open(); err == nil {
		t.Fatal("a revoked token still reads")
	}
	before, _ := safes.GetSafe(ks.SafeId)
	for name, params := range map[string]json.RawMessage{
		"no token":  json.RawMessage(`{}`),
		"no body":   nil,
		"bad token": vaultUserParams("", "wrong-user-token"),
		"malformed": json.RawMessage(`nope`),
	} {
		if _, err := m.UpdateKeystore(ctx, sc.Owner, ks.Ref, params); !errors.Is(err, ErrInvalidKeystore) {
			t.Errorf("UpdateKeystore(%s) = %v, want ErrInvalidKeystore", name, err)
		} else if strings.Contains(err.Error(), "user-token") {
			t.Errorf("UpdateKeystore(%s): the error repeats the token: %v", name, err)
		}
	}
	if after, _ := safes.GetSafe(ks.SafeId); after.Secret.Token() != before.Secret.Token() {
		t.Error("a refused update changed the token")
	}

	v.addToken("new-user-token", nil)
	attempts := v.renewAttempts()
	// Only the token changes: an address given along is not taken.
	updated, err := m.UpdateKeystore(ctx, sc.Owner, ks.Ref, vaultUserParams("http://elsewhere.example", "new-user-token"))
	if err != nil {
		t.Fatalf("UpdateKeystore = %v", err)
	}
	if updated.Ref != ks.Ref || updated.Name != "My Vault" {
		t.Errorf("UpdateKeystore = %+v", updated)
	}
	stored, _ := safes.GetSafe(ks.SafeId)
	if stored.Secret.Token() == before.Secret.Token() || strings.Contains(stored.Secret.Token(), "new-user-token") {
		t.Error("the new token is not sealed in place of the old one")
	}
	if !bytes.Equal(stored.Keyring[0].Params, before.Keyring[0].Params) {
		t.Errorf("the location changed: %s, was %s", stored.Keyring[0].Params, before.Keyring[0].Params)
	}
	if err := open(); err != nil {
		t.Errorf("open with the new token: %v", err)
	}
	// The first use of the new token renews it, without waiting for the
	// schedule of the old one.
	if got := v.renewAttempts(); got != attempts+1 {
		t.Errorf("%d renewals asked since the update, want 1", got-attempts)
	}
}
