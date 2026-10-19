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
