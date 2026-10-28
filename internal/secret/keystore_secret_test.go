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
	"testing"

	"git.happydns.org/happyDomain/model"
)

// vaultFixture is a Manager reaching a test Vault, the token of whose
// keystores is a secret like any other.
type vaultFixture struct {
	m        *Manager
	safes    *memSafeStorage
	checks   *memCheckStorage
	defaults *memDefaults
	vault    *testVault
}

func newVaultFixture(t *testing.T, policy Policy, key *InstanceKey) *vaultFixture {
	t.Helper()
	f := &vaultFixture{safes: newMemSafeStorage(), checks: &memCheckStorage{}, defaults: newMemDefaults(), vault: newTestVault(t)}
	f.vault.put("dns/ovh", map[string]any{"app_secret": "s3cr3t"})
	f.m = f.start(t, policy, key)
	return f
}

// start starts a Manager on the storage of f, as a restart of happyDomain.
func (f *vaultFixture) start(t *testing.T, policy Policy, key *InstanceKey) *Manager {
	t.Helper()
	m, err := Start(context.Background(), Config{Policy: policy, InstanceKey: key, Safes: f.safes, Checks: f.checks, Defaults: f.defaults, Outbound: loopbackGuard(t)})
	if err != nil {
		t.Fatalf("Start(%s): %v", policy, err)
	}
	return m
}

// createVault creates a Vault keystore of the owner of objectContext, its
// token going to the keystore ref names, or as for any secret when empty.
func (f *vaultFixture) createVault(t *testing.T, ctx context.Context, name, ref string) (Keystore, error) {
	t.Helper()
	return f.createVaultWith(t, ctx, name, ref, "user-token")
}

// createVaultWith is createVault, with token as the token parameter: a
// reference when ref keeps references.
func (f *vaultFixture) createVaultWith(t *testing.T, ctx context.Context, name, ref, token string) (Keystore, error) {
	t.Helper()
	if ref != "" {
		ctx = WithKeystore(ctx, ref)
	}
	return f.m.CreateKeystore(ctx, objectContext().Owner, KindVaultUser, name, vaultUserParams(f.vault.URL, token))
}

// sealInto seals a reference to the test secret into the Vault keystore ks.
func (f *vaultFixture) sealInto(t *testing.T, ctx context.Context, ks Keystore) string {
	t.Helper()
	obj := &managedObject{ApiKey: happydns.NewSecret("dns/ovh#app_secret")}
	if err := f.m.SealObjectInto(ctx, objectContext(), obj, Destination{Keystore: ks.Ref}); err != nil {
		t.Fatalf("SealObjectInto: %v", err)
	}
	return obj.ApiKey.Token()
}

// open opens token, a reference sealed in a Vault keystore, through m.
func open(t *testing.T, ctx context.Context, m *Manager, token string) (string, error) {
	t.Helper()
	back := &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := m.OpenObject(ctx, objectContext(), back); err != nil {
		return "", err
	}
	return back.ApiKey.Reveal(), nil
}

// The token goes where any secret goes: under the instance policy, into the
// instance safe of its owner, never in clear; under the plaintext one, as it
// is, without a keyset.
func TestVaultTokenFollowsThePolicy(t *testing.T) {
	ctx := context.Background()

	f := newVaultFixture(t, PolicyInstance, testInstanceKey(t))
	ks, err := f.createVault(t, ctx, "My Vault", "")
	if err != nil {
		t.Fatal(err)
	}
	safe, _ := f.safes.GetSafe(ks.SafeId)
	instance, err := f.safes.GetSafeByOwner(objectContext().Owner, KindInstance)
	if err != nil {
		t.Fatalf("no instance safe holds the token: %v", err)
	}
	if sv, err := ParseSealed(safe.Secret.Token()); err != nil || !sv.SafeId.Equals(instance.Id) {
		t.Errorf("the token is sealed in %v, %v; want the instance safe", sv.SafeId, err)
	}
	raw, _ := json.Marshal(safe)
	if bytes.Contains(raw, []byte("user-token")) {
		t.Error("the token is stored in clear")
	}
	if got, err := open(t, ctx, f.m, f.sealInto(t, ctx, ks)); err != nil || got != "s3cr3t" {
		t.Errorf("open = %q, %v", got, err)
	}

	p := newVaultFixture(t, PolicyPlaintext, nil)
	ks, err = p.createVault(t, ctx, "My Vault", "")
	if err != nil {
		t.Fatalf("CreateKeystore without a keyset: %v", err)
	}
	if got, err := open(t, ctx, p.m, p.sealInto(t, ctx, ks)); err != nil || got != "s3cr3t" {
		t.Errorf("open without a keyset = %q, %v", got, err)
	}
	if !ks.Unattended {
		t.Error("a Vault whose token is in clear does not open unattended")
	}
}

// Its owner keeps the token in a keystore only they open: the Vault then
// opens in their session only, and is not offered to what runs in the
// background.
func TestVaultTokenInALoginPasswordKeystore(t *testing.T) {
	f := newVaultFixture(t, PolicyInstance, testInstanceKey(t))
	owner := objectContext().Owner
	pw, _ := json.Marshal(map[string]string{PasswordParam: "hunter2"})
	login, err := f.m.CreateKeystore(context.Background(), owner, KindLoginPassword, "Login", pw)
	if err != nil {
		t.Fatal(err)
	}

	// Locked, the token cannot be sealed there: the owner is asked to
	// unlock it.
	if _, err := f.createVault(t, context.Background(), "My Vault", login.Ref); !isLocked(err, login.Ref) {
		t.Fatalf("CreateKeystore into a locked keystore = %v, want locked", err)
	}

	s1 := scoped("s1")
	f.m.UnlockAtLogin(s1, owner, "hunter2")
	ks, err := f.createVault(t, s1, "My Vault", login.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if ks.Unattended {
		t.Error("a Vault whose token needs its owner opens unattended")
	}
	safe, _ := f.safes.GetSafe(ks.SafeId)
	if sv, err := ParseSealed(safe.Secret.Token()); err != nil || sv.SafeId.String() != login.Ref {
		t.Errorf("the token is sealed in %v, %v; want the login-password keystore", sv.SafeId, err)
	}

	token := f.sealInto(t, s1, ks)
	if got, err := open(t, s1, f.m, token); err != nil || got != "s3cr3t" {
		t.Errorf("open in the session = %q, %v", got, err)
	}
	if _, err := open(t, context.Background(), f.m, token); !isLocked(err, login.Ref) {
		t.Errorf("open in the background = %v, want the login-password keystore locked", err)
	}
	obj := &managedObject{ApiKey: happydns.NewSecret("dns/ovh#app_secret")}
	if err := f.m.SealObjectInto(s1, objectContext(), obj, Destination{Keystore: ks.Ref, Unattended: true}); !errors.Is(err, ErrKeystoreNotUnattended) {
		t.Errorf("SealObjectInto unattended = %v, want ErrKeystoreNotUnattended", err)
	}
}

// The token cannot be sealed in the keystore it opens, chosen or by default.
func TestVaultTokenNotInItself(t *testing.T) {
	ctx := context.Background()
	f := newVaultFixture(t, PolicyPlaintext, nil)
	owner := objectContext().Owner
	ks, err := f.createVault(t, ctx, "My Vault", "")
	if err != nil {
		t.Fatal(err)
	}

	update := vaultUserParams("", "user-token")
	if _, err := f.m.UpdateKeystore(WithKeystore(ctx, ks.Ref), owner, ks.Ref, update); !errors.Is(err, ErrInvalidKeystore) {
		t.Errorf("UpdateKeystore into itself = %v, want ErrInvalidKeystore", err)
	}

	// Its default, it gives way to the policy.
	if err := f.m.SetDefaultKeystore(ctx, owner, ks.Ref); err != nil {
		t.Fatal(err)
	}
	if _, err := f.m.UpdateKeystore(ctx, owner, ks.Ref, update); err != nil {
		t.Fatalf("UpdateKeystore with itself as default = %v", err)
	}
	if safe, _ := f.safes.GetSafe(ks.SafeId); safe.Secret.IsSealed() {
		t.Errorf("the token went to a keystore: %s", safe.Secret.Token())
	}
	if got, err := open(t, ctx, f.m, f.sealInto(t, ctx, ks)); err != nil || got != "s3cr3t" {
		t.Errorf("open = %q, %v", got, err)
	}
}

// The token is a secret as any: kept in a keystore of references, its
// parameter is a reference, which resolves to the token.
func TestVaultTokenAsAReference(t *testing.T) {
	ctx := context.Background()
	f := newVaultFixture(t, PolicyPlaintext, nil)
	f.vault.put("tokens/b", map[string]any{"token": "user-token"})
	a, err := f.createVault(t, ctx, "A", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.createVaultWith(t, ctx, "Bad", a.Ref, "tokens/b#nope"); err == nil {
		t.Error("CreateKeystore with a reference that does not resolve succeeded")
	}
	b, err := f.createVaultWith(t, ctx, "B", a.Ref, "tokens/b#token")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := open(t, ctx, f.m, f.sealInto(t, ctx, b)); err != nil || got != "s3cr3t" {
		t.Errorf("open through B = %q, %v", got, err)
	}
}

// Two Vaults whose tokens are each in the other do not open, rather than
// loop.
func TestVaultTokensInACycle(t *testing.T) {
	ctx := context.Background()
	f := newVaultFixture(t, PolicyPlaintext, nil)
	f.vault.put("tokens/b", map[string]any{"token": "user-token"})
	a, err := f.createVault(t, ctx, "A", "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.createVaultWith(t, ctx, "B", a.Ref, "tokens/b#token")
	if err != nil {
		t.Fatal(err)
	}

	// Planted, as no keystore lets it happen, see TestVaultTokenNotInACycle.
	safeA, _ := f.safes.GetSafe(a.SafeId)
	safeB, _ := f.safes.GetSafe(b.SafeId)
	safeA.Secret = happydns.Secret{}
	safeA.Secret.SetSealed(FormatSealed(safeB.Id, []byte("tokens/b#token")))
	if err := f.safes.UpdateSafe(safeA); err != nil {
		t.Fatal(err)
	}

	obj := &managedObject{ApiKey: happydns.NewSecret("dns/ovh#app_secret")}
	if err := f.m.SealObjectInto(ctx, objectContext(), obj, Destination{Keystore: a.Ref}); !errors.Is(err, errSafeCycle) {
		t.Errorf("SealObjectInto a Vault in a cycle = %v, want errSafeCycle", err)
	}
}

// Reseal brings the token to the policy, as any secret: once back to clear,
// the safes are dropped, and the Vault opens without the keyset.
func TestVaultTokenBackToClear(t *testing.T) {
	ctx := context.Background()
	key := testInstanceKey(t)
	f := newVaultFixture(t, PolicyInstance, key)
	ks, err := f.createVault(t, ctx, "My Vault", "")
	if err != nil {
		t.Fatal(err)
	}
	token := f.sealInto(t, ctx, ks)

	plaintext := f.start(t, PolicyPlaintext, key)
	safe, _ := f.safes.GetSafe(ks.SafeId)
	obj := &SafeSecret{Secret: safe.Secret}
	if changed, err := plaintext.ResealObject(ctx, SafeSecretContext(safe), obj); err != nil || !changed {
		t.Fatalf("ResealObject = %v, %v", changed, err)
	}
	safe.Secret = obj.Secret
	if err := f.safes.UpdateSafe(safe); err != nil {
		t.Fatal(err)
	}
	if n, err := plaintext.DropSafes(ctx, f.checks); err != nil || n != 1 {
		t.Fatalf("DropSafes = %d, %v", n, err)
	}

	if got, err := open(t, ctx, f.start(t, PolicyPlaintext, nil), token); err != nil || got != "s3cr3t" {
		t.Errorf("open without the keyset = %q, %v", got, err)
	}
}

// The tokens are objects as any for the administration of secrets: counted,
// resealed, listed under the keystore holding them, and moved.
func TestSafeSecretsHolder(t *testing.T) {
	ctx := context.Background()
	key := testInstanceKey(t)
	f := newVaultFixture(t, PolicyInstance, key)
	owner := objectContext().Owner
	ks, err := f.createVault(t, ctx, "My Vault", "")
	if err != nil {
		t.Fatal(err)
	}
	instance, _ := f.safes.GetSafeByOwner(owner, KindInstance)

	h := f.m.SafeSecrets()
	if c, err := h.InspectSecrets(ctx); err != nil || c.Sealed[KindInstance] != 1 || c.Clear != 0 {
		t.Errorf("InspectSecrets = %+v, %v; want the token sealed under instance", c, err)
	}
	objects, err := h.OwnerObjects(ctx, owner)
	if err != nil || len(objects) != 1 || objects[0].ObjectType != SafeSecretObjectType || objects[0].ObjectId != ks.Ref || objects[0].Label != "My Vault" || !objects[0].Safes[0].Equals(instance.Id) {
		t.Errorf("OwnerObjects = %+v, %v; want the Vault keystore, in the instance safe", objects, err)
	}

	// Moved, the token opens from its new keystore; not into itself.
	if report, err := h.MoveOwnerSecrets(ctx, owner, instance.Id, ks.Ref); err != nil || report.Failed != 1 {
		t.Errorf("MoveOwnerSecrets into itself = %+v, %v; want it failed", report, err)
	}
	pw, _ := json.Marshal(map[string]string{PasswordParam: "hunter2"})
	login, err := f.m.CreateKeystore(ctx, owner, KindLoginPassword, "Login", pw)
	if err != nil {
		t.Fatal(err)
	}
	s1 := scoped("s1")
	f.m.UnlockAtLogin(s1, owner, "hunter2")
	if report, err := h.MoveOwnerSecrets(s1, owner, instance.Id, login.Ref); err != nil || report.Changed != 1 {
		t.Fatalf("MoveOwnerSecrets = %+v, %v", report, err)
	}
	token := f.sealInto(t, s1, ks)
	if _, err := open(t, ctx, f.m, token); !isLocked(err, login.Ref) {
		t.Errorf("open in the background = %v, want locked", err)
	}
	if got, err := open(t, s1, f.m, token); err != nil || got != "s3cr3t" {
		t.Errorf("open in the session = %q, %v", got, err)
	}

	// A keystore of its owner, it is left where it is by a reseal.
	plaintext := f.start(t, PolicyPlaintext, key)
	if report, err := plaintext.SafeSecrets().ResealSecrets(ctx); err != nil || report.Changed != 0 {
		t.Errorf("ResealSecrets = %+v, %v; want the token left in its keystore", report, err)
	}

	// Back in the instance safe, it is resealed in clear.
	if report, err := h.MoveOwnerSecrets(s1, owner, login.SafeId, KindInstance); err != nil || report.Changed != 1 {
		t.Fatalf("MoveOwnerSecrets back = %+v, %v", report, err)
	}
	if report, err := plaintext.SafeSecrets().ResealSecrets(ctx); err != nil || report.Changed != 1 {
		t.Errorf("ResealSecrets = %+v, %v; want the token in clear", report, err)
	}
	if c, err := plaintext.SafeSecrets().InspectSecrets(ctx); err != nil || c.Clear != 1 {
		t.Errorf("InspectSecrets = %+v, %v; want the token in clear", c, err)
	}
}

// A keystore keeping a secret tells it, so that its owner is offered where
// to keep it.
func TestKeepsSecretIsDescribed(t *testing.T) {
	f := newVaultFixture(t, PolicyPlaintext, nil)
	ks, err := f.createVault(t, context.Background(), "My Vault", "")
	if err != nil {
		t.Fatal(err)
	}
	if !ks.KeepsSecret {
		t.Error("the Vault keystore does not tell it keeps a secret")
	}
	for _, k := range f.m.CreatableKinds() {
		if k.KeepsSecret != (k.Kind == KindVaultUser) {
			t.Errorf("kind %s: KeepsSecret = %v", k.Kind, k.KeepsSecret)
		}
	}
}

// threeVaults creates C, whose token is in the instance safe, A, whose token
// is a reference kept in C, and B, whose token is a reference kept in A.
func threeVaults(t *testing.T) (f *vaultFixture, a, b, c Keystore) {
	t.Helper()
	ctx := context.Background()
	f = newVaultFixture(t, PolicyInstance, testInstanceKey(t))
	f.vault.put("tokens/a", map[string]any{"token": "user-token"})
	f.vault.put("tokens/b", map[string]any{"token": "user-token"})
	var err error
	if c, err = f.createVault(t, ctx, "C", ""); err != nil {
		t.Fatal(err)
	}
	if a, err = f.createVaultWith(t, ctx, "A", c.Ref, "tokens/a#token"); err != nil {
		t.Fatal(err)
	}
	if b, err = f.createVaultWith(t, ctx, "B", a.Ref, "tokens/b#token"); err != nil {
		t.Fatal(err)
	}
	return
}

// sealedIn tells the safe the token of the Vault keystore ks is sealed in.
func (f *vaultFixture) sealedIn(t *testing.T, ks Keystore) happydns.Identifier {
	t.Helper()
	safe, err := f.safes.GetSafe(ks.SafeId)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := safeIdOf(safe.Secret)
	return id
}

// A token goes in no keystore opened through the keystore it belongs to, the
// chain of the keystores in between included.
func TestVaultTokenNotInACycle(t *testing.T) {
	ctx := context.Background()
	owner := objectContext().Owner

	// B is opened through A: A cannot go in B.
	f, a, b, c := threeVaults(t)
	before := f.sealedIn(t, a)
	_, err := f.m.UpdateKeystore(WithKeystore(ctx, b.Ref), owner, a.Ref, vaultUserParams(f.vault.URL, "tokens/b#token"))
	if !errors.Is(err, ErrInvalidKeystore) {
		t.Errorf("UpdateKeystore of A into B = %v, want ErrInvalidKeystore", err)
	}
	if got := f.sealedIn(t, a); !got.Equals(before) {
		t.Errorf("the token of A moved to %v despite the refusal", got)
	}

	// C is opened through B, then A: C cannot go in B either.
	_, err = f.m.UpdateKeystore(WithKeystore(ctx, b.Ref), owner, c.Ref, vaultUserParams(f.vault.URL, "tokens/b#token"))
	if !errors.Is(err, ErrInvalidKeystore) {
		t.Errorf("UpdateKeystore of C into B = %v, want ErrInvalidKeystore", err)
	}

	// Without a cycle, a chain is fine: B in C directly.
	if _, err := f.m.UpdateKeystore(WithKeystore(ctx, c.Ref), owner, b.Ref, vaultUserParams(f.vault.URL, "tokens/b#token")); err != nil {
		t.Errorf("UpdateKeystore of B into C = %v", err)
	}
	if got, err := open(t, ctx, f.m, f.sealInto(t, ctx, b)); err != nil || got != "s3cr3t" {
		t.Errorf("open through B = %q, %v", got, err)
	}
}

// A failing storage is reported, not taken for the end of the chain.
func TestVaultTokenCycleCheckStorageError(t *testing.T) {
	ctx := context.Background()
	owner := objectContext().Owner
	cs := &failingSafes{memSafeStorage: newMemSafeStorage()}
	f := &vaultFixture{safes: cs.memSafeStorage, checks: &memCheckStorage{}, defaults: newMemDefaults(), vault: newTestVault(t)}
	f.vault.put("tokens/a", map[string]any{"token": "user-token"})
	m, err := Start(ctx, Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: cs, Checks: f.checks, Defaults: f.defaults, Outbound: loopbackGuard(t)})
	if err != nil {
		t.Fatal(err)
	}
	f.m = m
	c, err := f.createVault(t, ctx, "C", "")
	if err != nil {
		t.Fatal(err)
	}
	a, err := f.createVaultWith(t, ctx, "A", c.Ref, "tokens/a#token")
	if err != nil {
		t.Fatal(err)
	}
	cs.failAfter = 1
	if _, err := m.UpdateKeystore(WithKeystore(ctx, c.Ref), owner, a.Ref, vaultUserParams(f.vault.URL, "tokens/a#token")); err == nil {
		t.Error("UpdateKeystore with a failing storage succeeded")
	}
}

// failingSafes fails every read from the failAfter-th on, when set.
type failingSafes struct {
	*memSafeStorage
	failAfter int
	reads     int
}

func (s *failingSafes) GetSafe(id happydns.Identifier) (*happydns.Safe, error) {
	s.reads++
	if s.failAfter > 0 && s.reads >= s.failAfter {
		return nil, errors.New("storage down")
	}
	return s.memSafeStorage.GetSafe(id)
}
