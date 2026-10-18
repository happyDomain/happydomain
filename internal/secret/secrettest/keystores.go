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

package secrettest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
)

// PersonalKind is a personal keystore its owner creates giving a passphrase,
// and that needs them to open: it keeps no secret, as only a test can afford.
const PersonalKind = "test-personal"

// WithPersonalStore returns cfg knowing PersonalKind.
func WithPersonalStore(cfg secret.Config) secret.Config {
	return cfg.WithStores(map[string]secret.StoreFactory{
		PersonalKind: func(secret.Config) (secret.Store, error) { return personalStore{}, nil },
	})
}

// PersonalParams returns the parameters creating a keystore of PersonalKind.
func PersonalParams() json.RawMessage {
	return json.RawMessage(`{"passphrase":"test"}`)
}

type personalStore struct{}

func (personalStore) Unlock(_ context.Context, safe *happydns.Safe) (secret.UnlockedSafe, error) {
	if safe.Kind != PersonalKind {
		return nil, errors.New("not a personal test safe")
	}
	return personalUnlocked{id: safe.Id}, nil
}

func (personalStore) CreateForm() []happydns.Field {
	return []happydns.Field{{Id: "passphrase", Type: "string", Secret: true, Required: true}}
}

func (personalStore) Create(_ context.Context, safe *happydns.Safe, params json.RawMessage) error {
	var p struct {
		Passphrase string `json:"passphrase"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	if p.Passphrase == "" {
		return errors.New("a passphrase is required")
	}
	safe.Keyring = []happydns.WrappedKeyset{{KEK: PersonalKind}}
	return nil
}

type personalUnlocked struct{ id happydns.Identifier }

// The payload carries the associated data in clear, then the value: enough
// to tell a secret opened at the wrong place.
func (u personalUnlocked) Seal(_ context.Context, sc secret.SecretContext, clear []byte) ([]byte, error) {
	return append([]byte(hex.EncodeToString(secret.AssociatedData(u.id, sc))+":"), clear...), nil
}

func (u personalUnlocked) Open(_ context.Context, sc secret.SecretContext, payload []byte) ([]byte, error) {
	ad, clear, ok := bytes.Cut(payload, []byte(":"))
	if !ok || string(ad) != hex.EncodeToString(secret.AssociatedData(u.id, sc)) {
		return nil, errors.New("payload sealed elsewhere")
	}
	return clear, nil
}

// Defaults keeps the default keystores in memory.
type Defaults struct {
	mu   sync.Mutex
	refs map[string]string
}

func NewDefaults() *Defaults {
	return &Defaults{refs: map[string]string{}}
}

func (d *Defaults) GetDefaultKeystore(owner happydns.Identifier) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	ref, ok := d.refs[owner.String()]
	if !ok {
		return "", happydns.ErrNotFound
	}
	return ref, nil
}

func (d *Defaults) PutDefaultKeystore(owner happydns.Identifier, ref string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.refs[owner.String()] = ref
	return nil
}

func (d *Defaults) ListDefaultKeystores() (map[string]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make(map[string]string, len(d.refs))
	for owner, ref := range d.refs {
		out[owner] = ref
	}
	return out, nil
}

func (d *Defaults) DeleteDefaultKeystore(owner happydns.Identifier) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.refs, owner.String())
	return nil
}

// PassphraseKind is a personal keystore opening with a passphrase its owner
// gives, at creation and to unlock it: {"passphrase": "..."}.
const PassphraseKind = "test-passphrase"

// PasswordKind is a personal keystore bound to the login password of its
// owner: {"password": "..."}.
const PasswordKind = "test-password"

// WithInteractiveStores returns cfg knowing PassphraseKind and PasswordKind.
func WithInteractiveStores(cfg secret.Config) secret.Config {
	return cfg.WithStores(map[string]secret.StoreFactory{
		PassphraseKind: func(secret.Config) (secret.Store, error) {
			return interactiveStore{kind: PassphraseKind, field: "passphrase"}, nil
		},
		PasswordKind: func(secret.Config) (secret.Store, error) {
			return passwordStore{interactiveStore{kind: PasswordKind, field: secret.PasswordParam}}, nil
		},
	})
}

// Material returns the JSON object holding value under field, to create or
// unlock a keystore of PassphraseKind ("passphrase") or PasswordKind
// ("password").
func Material(field, value string) json.RawMessage {
	b, _ := json.Marshal(map[string]string{field: value})
	return b
}

func digest(s string) []byte {
	d := sha256.Sum256([]byte(s))
	return d[:]
}

// interactiveStore keeps the digest of what opens its safes in their
// keyring, as only a test can afford.
type interactiveStore struct{ kind, field string }

func (s interactiveStore) Unlock(context.Context, *happydns.Safe) (secret.UnlockedSafe, error) {
	return nil, errors.New("an interactive safe is not opened without its owner")
}

func (s interactiveStore) UnlockForm() []happydns.Field {
	return []happydns.Field{{Id: s.field, Type: "string", Secret: true, Required: true}}
}

func (s interactiveStore) CreateForm() []happydns.Field { return s.UnlockForm() }

func (s interactiveStore) read(raw json.RawMessage) (string, error) {
	var p map[string]string
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", err
	}
	if p[s.field] == "" {
		return "", errors.New(s.field + " is required")
	}
	return p[s.field], nil
}

func (s interactiveStore) Create(_ context.Context, safe *happydns.Safe, params json.RawMessage) error {
	v, err := s.read(params)
	if err != nil {
		return err
	}
	safe.Keyring = []happydns.WrappedKeyset{{KEK: s.kind, Blob: digest(v)}}
	return nil
}

func (s interactiveStore) open(safe *happydns.Safe, v string) (secret.UnlockedSafe, error) {
	for _, w := range safe.Keyring {
		if w.KEK == s.kind && bytes.Equal(w.Blob, digest(v)) {
			return personalUnlocked{id: safe.Id}, nil
		}
	}
	return nil, secret.ErrWrongUnlockMaterial
}

func (s interactiveStore) UnlockWith(_ context.Context, safe *happydns.Safe, material json.RawMessage) (secret.UnlockedSafe, error) {
	v, err := s.read(material)
	if err != nil {
		return nil, secret.ErrWrongUnlockMaterial
	}
	return s.open(safe, v)
}

// passwordStore is no Creator: the Manager creates its keystores from the
// login password alone, through Bind.
type passwordStore struct{ inner interactiveStore }

func (s passwordStore) Unlock(ctx context.Context, safe *happydns.Safe) (secret.UnlockedSafe, error) {
	return s.inner.Unlock(ctx, safe)
}

func (s passwordStore) UnlockForm() []happydns.Field { return s.inner.UnlockForm() }

func (s passwordStore) UnlockWith(ctx context.Context, safe *happydns.Safe, material json.RawMessage) (secret.UnlockedSafe, error) {
	return s.inner.UnlockWith(ctx, safe, material)
}

func (s passwordStore) Bind(_ context.Context, safe *happydns.Safe, password string) error {
	safe.Keyring = []happydns.WrappedKeyset{{KEK: s.inner.kind, Blob: digest(password)}}
	return nil
}

func (s passwordStore) UnlockWithPassword(_ context.Context, safe *happydns.Safe, password string) (secret.UnlockedSafe, error) {
	return s.inner.open(safe, password)
}

func (s passwordStore) Rebind(_ context.Context, safe *happydns.Safe, old, new string) error {
	if _, err := s.inner.open(safe, old); err != nil {
		return err
	}
	for i := range safe.Keyring {
		if safe.Keyring[i].KEK == s.inner.kind {
			safe.Keyring[i].Blob = digest(new)
		}
	}
	return nil
}

// ReferenceKind is a personal keystore keeping no secret, opening without its
// owner: they give, at creation, the secrets it resolves by reference, as a
// file of theirs would: {"values": {"<reference>": "<secret>"}}.
const ReferenceKind = "test-reference"

// WithReferenceStore returns cfg knowing ReferenceKind.
func WithReferenceStore(cfg secret.Config) secret.Config {
	return cfg.WithStores(map[string]secret.StoreFactory{
		ReferenceKind: func(secret.Config) (secret.Store, error) { return referenceStore{}, nil },
	})
}

// ReferenceParams returns the parameters creating a keystore of ReferenceKind
// resolving the references of values.
func ReferenceParams(values map[string]string) json.RawMessage {
	b, _ := json.Marshal(map[string]any{"values": values})
	return b
}

type referenceStore struct{}

func (referenceStore) OpensUnattended() bool { return true }

func (referenceStore) ReferenceHint() string { return "the key of the secret in your file" }

func (referenceStore) Reference(payload []byte) (string, error) {
	if len(payload) == 0 {
		return "", errors.New("empty reference")
	}
	return string(payload), nil
}

func (referenceStore) CreateForm() []happydns.Field {
	return []happydns.Field{{Id: "values", Type: "string", Required: true}}
}

func (referenceStore) Create(_ context.Context, safe *happydns.Safe, params json.RawMessage) error {
	var p struct {
		Values map[string]string `json:"values"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	if len(p.Values) == 0 {
		return errors.New("no values")
	}
	b, err := json.Marshal(p.Values)
	if err != nil {
		return err
	}
	safe.Keyring = []happydns.WrappedKeyset{{KEK: ReferenceKind, Params: b}}
	return nil
}

func (referenceStore) Unlock(_ context.Context, safe *happydns.Safe) (secret.UnlockedSafe, error) {
	if safe.Kind != ReferenceKind || len(safe.Keyring) == 0 {
		return nil, errors.New("not a reference test safe")
	}
	var values map[string]string
	if err := json.Unmarshal(safe.Keyring[0].Params, &values); err != nil {
		return nil, err
	}
	return referenceUnlocked(values), nil
}

type referenceUnlocked map[string]string

func (u referenceUnlocked) Seal(_ context.Context, _ secret.SecretContext, ref []byte) ([]byte, error) {
	if _, ok := u[string(ref)]; !ok {
		return nil, errors.New("unknown reference")
	}
	return ref, nil
}

func (u referenceUnlocked) Open(_ context.Context, _ secret.SecretContext, payload []byte) ([]byte, error) {
	v, ok := u[string(payload)]
	if !ok {
		return nil, errors.New("unknown reference")
	}
	return []byte(v), nil
}
