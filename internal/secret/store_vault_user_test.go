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

func newTestVaultUserStore(t *testing.T, key *InstanceKey) *vaultUserStore {
	t.Helper()
	store, err := newVaultUserStore(Config{InstanceKey: key, Outbound: loopbackGuard(t)})
	if err != nil {
		t.Fatal(err)
	}
	return store.(*vaultUserStore)
}

func TestVaultUserStoreNeedsTheInstanceKeyset(t *testing.T) {
	if _, err := newVaultUserStore(Config{}); !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("newVaultUserStore(no keyset) = %v, want ErrStoreUnavailable", err)
	}
	if _, err := newVaultUserStore(Config{InstanceKey: testInstanceKey(t), Stores: []string{KindVaultUser}}); !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("newVaultUserStore(instance store left out) = %v, want ErrStoreUnavailable", err)
	}
	if _, err := newVaultUserStore(Config{InstanceKey: testInstanceKey(t), Stores: []string{KindInstance, KindVaultUser}}); err != nil {
		t.Errorf("newVaultUserStore = %v", err)
	}
}

func TestVaultUserStoreSealsTheToken(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	v.put("dns/ovh", map[string]any{"app_secret": "s3cr3t"})
	store := newTestVaultUserStore(t, testInstanceKey(t))
	owner := happydns.Identifier{0x01}

	safe := vaultSafe(t, store, owner, KindVaultUser, vaultUserParams(v.URL, "user-token"))
	if len(safe.Keyring) != 1 || safe.Keyring[0].KEK != KindVaultUser {
		t.Fatalf("keyring = %+v", safe.Keyring)
	}
	if bytes.Contains(safe.Keyring[0].Blob, []byte("user-token")) || bytes.Contains(safe.Keyring[0].Params, []byte("user-token")) {
		t.Error("the token is stored in clear")
	}

	u, err := store.Unlock(ctx, safe)
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	payload, err := u.Seal(ctx, objectContext(), []byte("dns/ovh#app_secret"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if ref, err := store.Reference(payload); err != nil || ref != "dns/ovh#app_secret" {
		t.Errorf("Reference = %q, %v", ref, err)
	}
	if got, err := u.Open(ctx, objectContext(), payload); err != nil || string(got) != "s3cr3t" {
		t.Errorf("Open = %q, %v", got, err)
	}

	// The token is bound to its safe: moved to another, it does not open.
	moved := *safe
	moved.Id = happydns.Identifier{0x06}
	if _, err := store.Unlock(ctx, &moved); err == nil {
		t.Error("the token of a safe opened another safe")
	}
	stolen := *safe
	stolen.Owner = happydns.Identifier{0x02}
	if _, err := store.Unlock(ctx, &stolen); err == nil {
		t.Error("the token of a safe opened for another owner")
	}

	// Under another keyset, it does not open.
	if _, err := newTestVaultUserStore(t, testInstanceKey(t)).Unlock(ctx, safe); err == nil {
		t.Error("the token opened under another keyset")
	}
}

func TestVaultUserStoreCreate(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	v.addToken("limited-user-token", map[string]any{"num_uses": 5})
	v.addToken("fixed-user-token", map[string]any{"renewable": false})
	v.addToken("batch-user-token", map[string]any{"renewable": false, "type": "batch"})
	v.addToken("forever-user-token", map[string]any{"ttl": 0, "renewable": false})
	store := newTestVaultUserStore(t, testInstanceKey(t))

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
	guarded, err := newVaultUserStore(Config{InstanceKey: testInstanceKey(t)})
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
	store := newTestVaultUserStore(t, testInstanceKey(t))
	now := time.Now()
	store.now = func() time.Time { return now }
	safe := vaultSafe(t, store, happydns.Identifier{0x01}, KindVaultUser, vaultUserParams(v.URL, "user-token"))

	unlock := func(want int, why string) {
		t.Helper()
		if _, err := store.Unlock(ctx, safe); err != nil {
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
	if _, err := store.Unlock(ctx, other); err != nil {
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
	store := newTestVaultUserStore(t, testInstanceKey(t))
	now := time.Now()
	store.now = func() time.Time { return now }
	safe := vaultSafe(t, store, happydns.Identifier{0x01}, KindVaultUser, vaultUserParams(v.URL, "forever-user-token"))

	for range 3 {
		if _, err := store.Unlock(ctx, safe); err != nil {
			t.Fatal(err)
		}
		now = now.Add(24 * time.Hour)
	}
	if got := v.renewAttempts(); got != 1 {
		t.Errorf("%d renewals asked, want 1", got)
	}
}

// The token follows the rotation of the instance keyset, and the key status
// tells which instance key it is sealed with.
func TestVaultUserStoreRewrap(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	v.put("k", map[string]any{"v": "x"})
	h, err := GenerateInstanceKeyset()
	if err != nil {
		t.Fatal(err)
	}
	key, _ := NewInstanceKey(h)
	safe := vaultSafe(t, newTestVaultUserStore(t, key), happydns.Identifier{0x01}, KindVaultUser, vaultUserParams(v.URL, "user-token"))

	rh, err := RotateInstanceKeyset(h)
	if err != nil {
		t.Fatal(err)
	}
	rotated, _ := NewInstanceKey(rh)
	store := newTestVaultUserStore(t, rotated)
	if id, err := store.WrappingKey(safe.Keyring[0]); err != nil || id == rotated.PrimaryKeyId() {
		t.Fatalf("WrappingKey before the rewrap = %d, %v; want the previous key", id, err)
	}
	if store.Lender() != KindInstance {
		t.Errorf("Lender = %q, want the instance store", store.Lender())
	}

	w := safe.Keyring[0]
	changed, err := store.Rewrap(ctx, safe, &w)
	if err != nil || !changed {
		t.Fatalf("Rewrap = %v, %v", changed, err)
	}
	if id, _ := store.WrappingKey(w); id != rotated.PrimaryKeyId() {
		t.Errorf("WrappingKey after the rewrap = %d, want the primary key %d", id, rotated.PrimaryKeyId())
	}
	if changed, err := store.Rewrap(ctx, safe, &w); err != nil || changed {
		t.Errorf("Rewrap again = %v, %v; want nothing to do", changed, err)
	}

	safe.Keyring[0] = w
	u, err := store.Unlock(ctx, safe)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := u.Open(ctx, objectContext(), []byte("k#v")); err != nil || string(got) != "x" {
		t.Errorf("Open after the rewrap = %q, %v", got, err)
	}
}

// The address is checked at each request too, not only at creation: a
// keystore whose Vault the guard no longer allows does not read.
func TestVaultUserStoreGuardsEachRequest(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	v.put("dns/ovh", map[string]any{"app_secret": "s3cr3t"})
	key := testInstanceKey(t)
	safe := vaultSafe(t, newTestVaultUserStore(t, key), happydns.Identifier{0x01}, KindVaultUser, vaultUserParams(v.URL, "user-token"))

	guarded, err := newVaultUserStore(Config{InstanceKey: key})
	if err != nil {
		t.Fatal(err)
	}
	before := v.renewAttempts()
	u, err := guarded.(*vaultUserStore).Unlock(ctx, safe)
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
	store := newTestVaultUserStore(t, testInstanceKey(t))
	good := vaultSafe(t, store, happydns.Identifier{0x01}, KindVaultUser, vaultUserParams(v.URL, "user-token"))

	otherKind := *good
	otherKind.Kind = KindInstance
	noEntry := *good
	noEntry.Keyring = []happydns.WrappedKeyset{{KEK: KindInstance, Blob: good.Keyring[0].Blob}}
	badParams := *good
	badParams.Keyring = []happydns.WrappedKeyset{{KEK: KindVaultUser, Params: json.RawMessage(`nope`), Blob: good.Keyring[0].Blob}}
	badBlob := *good
	badBlob.Keyring = []happydns.WrappedKeyset{{KEK: KindVaultUser, Params: good.Keyring[0].Params, Blob: []byte("garbage")}}

	for name, safe := range map[string]*happydns.Safe{
		"another kind":      &otherKind,
		"no vault-user key": &noEntry,
		"bad parameters":    &badParams,
		"bad token blob":    &badBlob,
	} {
		if _, err := store.Unlock(ctx, safe); err == nil {
			t.Errorf("Unlock(%s) succeeded", name)
		}
	}
}

// At startup, a keyset that is not the one the database was used with is
// refused: the tokens would not open.
func TestVaultUserStoreCheckStartup(t *testing.T) {
	ctx := context.Background()
	checks := &memCheckStorage{}
	key := testInstanceKey(t)
	store, err := newVaultUserStore(Config{InstanceKey: key, Checks: checks})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.(StartupChecker).CheckStartup(ctx); err != nil {
		t.Errorf("CheckStartup = %v", err)
	}

	other, err := newVaultUserStore(Config{InstanceKey: testInstanceKey(t), Checks: checks})
	if err != nil {
		t.Fatal(err)
	}
	if err := other.(StartupChecker).CheckStartup(ctx); !errors.Is(err, ErrWrongInstanceKey) {
		t.Errorf("CheckStartup(another keyset) = %v, want ErrWrongInstanceKey", err)
	}

	if err := newTestVaultUserStore(t, key).CheckStartup(ctx); err != nil {
		t.Errorf("CheckStartup(no check storage) = %v", err)
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
	store := newTestVaultUserStore(t, testInstanceKey(t))
	safe := vaultSafe(t, store, happydns.Identifier{0x01}, KindVaultUser,
		json.RawMessage(`{"address":"`+srv.URL+`","token":"user-token","mount":"kv/team","namespace":"ops"}`))

	u, err := store.Unlock(ctx, safe)
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
	store := newTestVaultUserStore(t, testInstanceKey(t))
	now := time.Now()
	store.now = func() time.Time { return now }
	safe := vaultSafe(t, store, happydns.Identifier{0x01}, KindVaultUser, vaultUserParams(v.URL, "user-token"))

	for range 3 {
		if _, err := store.Unlock(ctx, safe); err != nil {
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
	store := Store(newTestVaultUserStore(t, testInstanceKey(t)))
	for name, ok := range map[string]bool{
		"Creator":        is[Creator](store),
		"Referencing":    is[Referencing](store),
		"Rewrapper":      is[Rewrapper](store),
		"KeyBorrower":    is[KeyBorrower](store),
		"StartupChecker": is[StartupChecker](store),
	} {
		if !ok {
			t.Errorf("vault-user is not a %s", name)
		}
	}
	if is[Interactive](store) {
		t.Error("vault-user is Interactive")
	}
	if is[KeyReporter](store) {
		t.Error("vault-user is a KeyReporter: the instance keys would be listed twice")
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
