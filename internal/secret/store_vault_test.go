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
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"git.happydns.org/happyDomain/model"
)

// testVault is a fake Vault server: a KV v2 engine mounted at secret, and
// an AppRole "role" with the secret ID "sid".
type testVault struct {
	*httptest.Server

	mu       sync.Mutex
	tokens   map[string]bool
	data     map[string]map[string]any
	versions map[string]map[string]map[string]any
	logins   int
	renews   int
	lease    int
	last     string
}

func newTestVault(t *testing.T) *testVault {
	t.Helper()
	v := &testVault{tokens: map[string]bool{"user-token": true}, data: map[string]map[string]any{}, versions: map[string]map[string]map[string]any{}, lease: 3600}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/secret/data/", func(w http.ResponseWriter, r *http.Request) {
		v.mu.Lock()
		defer v.mu.Unlock()
		if !v.tokens[r.Header.Get("X-Vault-Token")] {
			http.Error(w, `{"errors":["permission denied"]}`, http.StatusForbidden)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/v1/secret/data/")
		v.last = path
		if strings.Contains(path, "forbidden") {
			// What the policy of the token does not grant.
			http.Error(w, `{"errors":["permission denied"]}`, http.StatusForbidden)
			return
		}
		data := v.data[path]
		if version := r.URL.Query().Get("version"); version != "" {
			data = v.versions[path][version]
		}
		if data == nil {
			http.Error(w, `{"errors":[]}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"data": data}})
	})
	mux.HandleFunc("POST /v1/auth/approle/login", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		v.mu.Lock()
		defer v.mu.Unlock()
		if body["role_id"] != "role" || body["secret_id"] != "sid" {
			http.Error(w, `{"errors":["invalid secret id"]}`, http.StatusBadRequest)
			return
		}
		v.logins++
		token := "instance-token-" + string(rune('a'+v.logins))
		v.tokens[token] = true
		json.NewEncoder(w).Encode(map[string]any{"auth": map[string]any{"client_token": token, "lease_duration": v.lease, "renewable": true}})
	})
	mux.HandleFunc("POST /v1/auth/token/renew-self", func(w http.ResponseWriter, r *http.Request) {
		v.mu.Lock()
		defer v.mu.Unlock()
		token := r.Header.Get("X-Vault-Token")
		if !v.tokens[token] {
			http.Error(w, `{"errors":["permission denied"]}`, http.StatusForbidden)
			return
		}
		v.renews++
		json.NewEncoder(w).Encode(map[string]any{"auth": map[string]any{"client_token": token, "lease_duration": v.lease, "renewable": true}})
	})
	mux.HandleFunc("GET /v1/auth/token/lookup-self", func(w http.ResponseWriter, r *http.Request) {
		v.mu.Lock()
		defer v.mu.Unlock()
		if !v.tokens[r.Header.Get("X-Vault-Token")] {
			http.Error(w, `{"errors":["permission denied"]}`, http.StatusForbidden)
			return
		}
		w.Write([]byte(`{"data":{}}`))
	})
	v.Server = httptest.NewServer(mux)
	t.Cleanup(v.Close)
	return v
}

func (v *testVault) put(path string, data map[string]any) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.data[path] = data
}

// revokeAll revokes every token handed out by the AppRole.
func (v *testVault) revokeAll() {
	v.mu.Lock()
	defer v.mu.Unlock()
	for token := range v.tokens {
		if strings.HasPrefix(token, "instance-token-") {
			delete(v.tokens, token)
		}
	}
}

func (v *testVault) counts() (logins, renews int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.logins, v.renews
}

func (v *testVault) config(t *testing.T) vaultConfig {
	t.Helper()
	sid := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(sid, []byte("sid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return vaultConfig{addr: v.URL, mount: "secret", prefix: "happydomain", appRoleMount: "approle", roleID: "role", secretIDFile: sid}
}

func newTestVaultStore(t *testing.T, v *testVault) *vaultStore {
	t.Helper()
	store, err := newVaultStoreWith(v.config(t))
	if err != nil {
		t.Fatal(err)
	}
	return store.(*vaultStore)
}

func vaultSafe(t *testing.T, store Creator, owner happydns.Identifier, kind string, params json.RawMessage) *happydns.Safe {
	t.Helper()
	safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: owner, Kind: kind}
	if err := store.Create(context.Background(), safe, params); err != nil {
		t.Fatalf("Create: %v", err)
	}
	return safe
}

func TestParseVaultReference(t *testing.T) {
	for ref, want := range map[string]vaultRef{
		"dns/ovh#app_secret":           {path: "dns/ovh", key: "app_secret"},
		"dns/ovh?version=3#app_secret": {path: "dns/ovh", version: 3, key: "app_secret"},
		"a#b#c":                        {path: "a#b", key: "c"},
		"one#k":                        {path: "one", key: "k"},
	} {
		got, err := parseVaultReference(ref)
		if err != nil || got != want {
			t.Errorf("parseVaultReference(%q) = %+v, %v; want %+v", ref, got, err, want)
		}
	}
	for _, ref := range []string{"", "dns/ovh", "dns/ovh#", "#key", "../x#k", "a/../b#k", "a//b#k", "/abs#k", "a/./b#k", "a?version=0#k", "a?version=x#k", "a?v=1#k", "a\x00#k", strings.Repeat("x", 2000) + "#k"} {
		if _, err := parseVaultReference(ref); err == nil {
			t.Errorf("parseVaultReference(%q) succeeded", ref)
		}
	}
}

func TestVaultStoreNeedsItsConfiguration(t *testing.T) {
	if _, err := newVaultStoreWith(vaultConfig{}); !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("newVaultStoreWith(no address) = %v, want ErrStoreUnavailable", err)
	}
	v := newTestVault(t)
	for name, change := range map[string]func(*vaultConfig){
		"no role":           func(c *vaultConfig) { c.roleID = "" },
		"no secret id":      func(c *vaultConfig) { c.secretIDFile = "" },
		"missing secret id": func(c *vaultConfig) { c.secretIDFile = filepath.Join(t.TempDir(), "nope") },
		"bad address":       func(c *vaultConfig) { c.addr = "vault:8200" },
		"missing CA":        func(c *vaultConfig) { c.caFile = filepath.Join(t.TempDir(), "nope") },
	} {
		c := v.config(t)
		change(&c)
		if _, err := newVaultStoreWith(c); err == nil || errors.Is(err, ErrStoreUnavailable) {
			t.Errorf("%s: newVaultStoreWith = %v, want a configuration error", name, err)
		}
	}
}

// At startup, credentials Vault refuses stop happyDomain; a Vault that
// cannot be reached does not.
func TestVaultStoreCheckStartup(t *testing.T) {
	v := newTestVault(t)
	if err := newTestVaultStore(t, v).CheckStartup(context.Background()); err != nil {
		t.Errorf("CheckStartup = %v", err)
	}

	c := v.config(t)
	c.roleID = "other"
	store, err := newVaultStoreWith(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.(StartupChecker).CheckStartup(context.Background()); err == nil {
		t.Error("CheckStartup accepted credentials Vault refuses")
	}

	c = v.config(t)
	down := httptest.NewServer(http.NotFoundHandler())
	c.addr = down.URL
	down.Close()
	store, err = newVaultStoreWith(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.(StartupChecker).CheckStartup(context.Background()); err != nil {
		t.Errorf("CheckStartup(unreachable) = %v, want happyDomain to start", err)
	}

	u, err := store.Unlock(context.Background(), &happydns.Safe{Owner: happydns.Identifier{0x01}, Kind: KindVault})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := u.Open(context.Background(), objectContext(), []byte("pa55w0rd/typed#in")); err == nil || strings.Contains(err.Error(), "pa55w0rd") {
		t.Errorf("Open(unreachable) = %v, want an error not repeating the reference", err)
	}
}

func TestVaultStoreOwnerPath(t *testing.T) {
	v := newTestVault(t)
	owner := happydns.Identifier{0x01}
	for prefix, want := range map[string]string{
		"":              owner.String(),
		"/happydomain/": "happydomain/" + owner.String(),
		"a/b":           "a/b/" + owner.String(),
	} {
		c := v.config(t)
		c.prefix = prefix
		store, err := newVaultStoreWith(c)
		if err != nil {
			t.Fatal(err)
		}
		if got := store.(*vaultStore).ownerPath(owner); got != want {
			t.Errorf("ownerPath(prefix %q) = %q, want %q", prefix, got, want)
		}
	}
}

func TestVaultStoreTrustsItsCA(t *testing.T) {
	v := newTestVault(t)
	tlsVault := httptest.NewTLSServer(v.Config.Handler)
	t.Cleanup(tlsVault.Close)
	c := v.config(t)
	c.addr = tlsVault.URL

	store, err := newVaultStoreWith(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.(*vaultStore).auth.get(context.Background()); err == nil {
		t.Error("logged in to a Vault whose certificate is not trusted")
	}

	dir := t.TempDir()
	c.caFile = filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(c.caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: tlsVault.Certificate().Raw}), 0o600); err != nil {
		t.Fatal(err)
	}
	if store, err = newVaultStoreWith(c); err != nil {
		t.Fatal(err)
	}
	if _, err := store.(*vaultStore).auth.get(context.Background()); err != nil {
		t.Errorf("get with the CA trusted: %v", err)
	}

	c.caFile = filepath.Join(dir, "not.pem")
	if err := os.WriteFile(c.caFile, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := newVaultStoreWith(c); err == nil {
		t.Error("newVaultStoreWith accepted a CA file holding no certificate")
	}
}

func TestVaultStoreReadsBelowTheOwnerPrefix(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	store := newTestVaultStore(t, v)
	owner := happydns.Identifier{0x01, 0x02}
	v.put("happydomain/"+owner.String()+"/dns/ovh", map[string]any{"app_secret": "s3cr3t", "port": 53, "empty": ""})
	v.put("happydomain/other/dns/ovh", map[string]any{"app_secret": "theirs"})

	u, err := store.Unlock(ctx, vaultSafe(t, store, owner, KindVault, json.RawMessage(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	sc := objectContext()
	payload, err := u.Seal(ctx, sc, []byte("dns/ovh#app_secret"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if strings.Contains(string(payload), "s3cr3t") {
		t.Error("the payload holds the secret")
	}
	if ref, err := store.Reference(payload); err != nil || ref != "dns/ovh#app_secret" {
		t.Errorf("Reference = %q, %v", ref, err)
	}
	if got, err := u.Open(ctx, sc, payload); err != nil || string(got) != "s3cr3t" {
		t.Errorf("Open = %q, %v", got, err)
	}
	if v.last != "happydomain/"+owner.String()+"/dns/ovh" {
		t.Errorf("read %q, want below the prefix of the owner", v.last)
	}

	for name, ref := range map[string]string{
		"missing secret":     "dns/nope#app_secret",
		"missing key":        "dns/ovh#nope",
		"not a string":       "dns/ovh#port",
		"empty string":       "dns/ovh#empty",
		"another owner":      "../other/dns/ovh#app_secret",
		"pa55w0rd":           "pa55w0rd-typed-in",
		"pa55w0rd as a path": "pa55w0rd/typed#in",
	} {
		_, err := u.Seal(ctx, sc, []byte(ref))
		if err == nil {
			t.Errorf("Seal(%s) succeeded", name)
		} else if strings.Contains(err.Error(), "pa55w0rd") {
			t.Errorf("Seal(%s): the error repeats the reference: %v", name, err)
		}
	}

	if hint := store.ReferenceHint(); !strings.Contains(hint, "secret/happydomain/") {
		t.Errorf("ReferenceHint = %q, want it naming the mount and prefix", hint)
	}
}

// Past its lease the token is replaced, not renewed, and so is one whose
// renewal Vault refuses; a token without a lease is kept as is.
func TestVaultStoreTokenExpiry(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	store := newTestVaultStore(t, v)
	now := time.Now()
	store.auth.now = func() time.Time { return now }

	if _, err := store.auth.get(ctx); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Hour)
	if _, err := store.auth.get(ctx); err != nil {
		t.Fatalf("get past the lease: %v", err)
	}
	if logins, renews := v.counts(); logins != 2 || renews != 0 {
		t.Errorf("past the lease: %d logins, %d renewals; want 2, 0", logins, renews)
	}

	now = now.Add(50 * time.Minute)
	v.revokeAll()
	if _, err := store.auth.get(ctx); err != nil {
		t.Fatalf("get after a refused renewal: %v", err)
	}
	if logins, renews := v.counts(); logins != 3 || renews != 0 {
		t.Errorf("after a refused renewal: %d logins, %d renewals; want 3, 0", logins, renews)
	}

	v.mu.Lock()
	v.lease = 0
	v.mu.Unlock()
	store.auth.drop(store.auth.token.Token)
	if _, err := store.auth.get(ctx); err != nil {
		t.Fatal(err)
	}
	now = now.Add(365 * 24 * time.Hour)
	if _, err := store.auth.get(ctx); err != nil {
		t.Fatal(err)
	}
	if logins, renews := v.counts(); logins != 4 || renews != 0 {
		t.Errorf("without a lease: %d logins, %d renewals; want 4, 0", logins, renews)
	}
}

func TestVaultStoreReadsAVersion(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	store := newTestVaultStore(t, v)
	owner := happydns.Identifier{0x01}
	path := "happydomain/" + owner.String() + "/k"
	v.put(path, map[string]any{"v": "latest"})
	v.versions[path] = map[string]map[string]any{"1": {"v": "first"}}

	u, err := store.Unlock(ctx, vaultSafe(t, store, owner, KindVault, json.RawMessage(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := u.Open(ctx, objectContext(), []byte("k?version=1#v")); err != nil || string(got) != "first" {
		t.Errorf("Open(version 1) = %q, %v", got, err)
	}
}

// The instance token is kept, renewed before it expires, and replaced by a
// new login once revoked.
func TestVaultStoreToken(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	store := newTestVaultStore(t, v)
	now := time.Now()
	store.auth.now = func() time.Time { return now }
	owner := happydns.Identifier{0x01}
	v.put("happydomain/"+owner.String()+"/k", map[string]any{"v": "x"})
	u, err := store.Unlock(ctx, vaultSafe(t, store, owner, KindVault, json.RawMessage(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	open := func() {
		t.Helper()
		if _, err := u.Open(ctx, objectContext(), []byte("k#v")); err != nil {
			t.Fatalf("Open: %v", err)
		}
	}

	open()
	open()
	if logins, renews := v.counts(); logins != 1 || renews != 0 {
		t.Errorf("after two reads: %d logins, %d renewals; want 1, 0", logins, renews)
	}

	now = now.Add(50 * time.Minute)
	open()
	if logins, renews := v.counts(); logins != 1 || renews != 1 {
		t.Errorf("near the end of the lease: %d logins, %d renewals; want 1, 1", logins, renews)
	}

	v.revokeAll()
	open()
	if logins, _ := v.counts(); logins != 2 {
		t.Errorf("after a revocation: %d logins, want 2", logins)
	}

	// Refused for a path, by the policy, with a live token: no new login.
	if _, err := u.Open(ctx, objectContext(), []byte("forbidden#v")); err == nil {
		t.Error("a forbidden secret opened")
	}
	if logins, _ := v.counts(); logins != 2 {
		t.Errorf("after a refusal of the policy: %d logins, want 2", logins)
	}
}

func TestVaultKindIsBuiltin(t *testing.T) {
	if _, ok := builtinStores[KindVault]; !ok {
		t.Fatalf("no builtin store %q", KindVault)
	}
	store := Store(newTestVaultStore(t, newTestVault(t)))
	if _, ok := store.(Creator); !ok {
		t.Error("vault is not a Creator")
	}
	if _, ok := store.(Referencing); !ok {
		t.Error("vault is not Referencing")
	}
	if _, ok := store.(StartupChecker); !ok {
		t.Error("vault does not check itself at startup")
	}
	if _, ok := store.(Interactive); ok {
		t.Error("vault is Interactive")
	}
	if !opensUnattended(store) {
		t.Error("vault does not open unattended")
	}
}

func TestVaultKeystoreThroughManager(t *testing.T) {
	ctx := context.Background()
	v := newTestVault(t)
	c := v.config(t)
	sc := objectContext()
	v.put("happydomain/"+sc.Owner.String()+"/dns/ovh", map[string]any{"app_secret": "s3cr3t"})
	m, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: newMemSafeStorage(), Defaults: newMemDefaults()}.WithStores(map[string]StoreFactory{
		KindVault: func(Config) (Store, error) { return newVaultStoreWith(c) },
	}))
	if err != nil {
		t.Fatal(err)
	}

	ks, err := m.CreateKeystore(ctx, sc.Owner, KindVault, "Vault", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	obj := &managedObject{ApiKey: happydns.NewSecret("dns/ovh#app_secret")}
	if err := m.SealObjectInto(ctx, sc, obj, Destination{Keystore: ks.Ref}); err != nil {
		t.Fatalf("SealObjectInto: %v", err)
	}
	shown := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := m.ShowReferences(ctx, sc, shown); err != nil || !shown.ApiKey.IsReference() {
		t.Fatalf("ShowReferences = %v, reference shown: %v", err, shown.ApiKey.IsReference())
	}
	back := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := m.OpenObject(ctx, sc, back); err != nil || back.ApiKey.Reveal() != "s3cr3t" {
		t.Errorf("OpenObject = %q, %v", back.ApiKey.Reveal(), err)
	}
}
