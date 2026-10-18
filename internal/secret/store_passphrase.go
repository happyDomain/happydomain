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
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/aead/subtle"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"golang.org/x/crypto/argon2"

	"git.happydns.org/happyDomain/model"
)

const (
	// KindPassphrase is a personal keystore whose key is wrapped by a
	// passphrase its owner chooses, and gives again to unlock it.
	KindPassphrase = "passphrase"

	// passphraseField names the passphrase in the forms of KindPassphrase,
	// passphraseConfirmField its confirmation at creation.
	passphraseField        = "passphrase"
	passphraseConfirmField = "passphrase-confirm"

	// minPassphraseLength is the length of the shortest passphrase, in
	// characters rather than bytes, so that accents do not count double.
	minPassphraseLength = 8
)

// phraseParams is what derives the wrapping key from a phrase, with
// Argon2id, kept in the keyring entry so that it can change without locking
// out older keystores.
type phraseParams struct {
	Salt    []byte `json:"salt"`
	Time    uint32 `json:"t"`
	Memory  uint32 `json:"m"`
	Threads uint8  `json:"p"`
}

// newPhraseParams returns the parameters of a new keyring entry: the second
// recommendation of RFC 9106, for when memory is constrained, and a new salt.
func newPhraseParams() (phraseParams, error) {
	p := phraseParams{Salt: make([]byte, 16), Time: 3, Memory: 64 * 1024, Threads: 4}
	_, err := rand.Read(p.Salt)
	return p, err
}

// check refuses parameters read from storage that would make a derivation
// exhaust the memory or time of the server, or that are too weak.
func (p phraseParams) check() error {
	switch {
	case len(p.Salt) < 16:
		return errors.New("salt too short")
	case p.Time < 1 || p.Time > 10:
		return fmt.Errorf("time cost %d out of bounds", p.Time)
	case p.Memory < 8*1024 || p.Memory > 1024*1024:
		return fmt.Errorf("memory cost %d KiB out of bounds", p.Memory)
	case p.Threads < 1 || p.Threads > 16:
		return fmt.Errorf("parallelism %d out of bounds", p.Threads)
	}
	return nil
}

func (p phraseParams) key(phrase string) []byte {
	return argon2.IDKey([]byte(phrase), p.Salt, p.Time, p.Memory, p.Threads, 32)
}

// phraseKeys wraps and unwraps the key of a safe under a phrase: what the
// keystores opening with a phrase share. They differ by where the phrase
// comes from.
type phraseKeys struct {
	kind  string
	field string
}

// wrap returns the keyring entry of dek, the key of safe, under phrase.
func (k phraseKeys) wrap(safe *happydns.Safe, dek *keyset.Handle, phrase string) (happydns.WrappedKeyset, error) {
	params, err := newPhraseParams()
	if err != nil {
		return happydns.WrappedKeyset{}, err
	}
	w, err := subtle.NewAESGCM(params.key(phrase))
	if err != nil {
		return happydns.WrappedKeyset{}, err
	}
	var buf bytes.Buffer
	if err := dek.WriteWithAssociatedData(keyset.NewBinaryWriter(&buf), w, wrapAssociatedData(safe)); err != nil {
		return happydns.WrappedKeyset{}, fmt.Errorf("unable to wrap the key of safe %s: %w", safe.Id.String(), err)
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return happydns.WrappedKeyset{}, err
	}
	return happydns.WrappedKeyset{KEK: k.kind, Params: raw, Blob: buf.Bytes()}, nil
}

// bind gives safe a new key, wrapped under phrase.
func (k phraseKeys) bind(safe *happydns.Safe, phrase string) error {
	if phrase == "" {
		return errors.New(k.field + " is required")
	}
	dek, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		return err
	}
	w, err := k.wrap(safe, dek, phrase)
	if err != nil {
		return err
	}
	safe.Keyring = []happydns.WrappedKeyset{w}
	return nil
}

// unwrap returns the key of safe, or ErrWrongUnlockMaterial when phrase does
// not open its entry.
func (k phraseKeys) unwrap(safe *happydns.Safe, phrase string) (*keyset.Handle, error) {
	if safe.Kind != k.kind {
		return nil, fmt.Errorf("safe %s is of kind %q, not %q", safe.Id.String(), safe.Kind, k.kind)
	}
	if phrase == "" {
		return nil, ErrWrongUnlockMaterial
	}
	for _, e := range safe.Keyring {
		if e.KEK != k.kind {
			continue
		}
		var params phraseParams
		if err := json.Unmarshal(e.Params, &params); err != nil {
			return nil, fmt.Errorf("safe %s has unreadable key parameters: %w", safe.Id.String(), err)
		}
		if err := params.check(); err != nil {
			return nil, fmt.Errorf("safe %s has unreadable key parameters: %w", safe.Id.String(), err)
		}
		w, err := subtle.NewAESGCM(params.key(phrase))
		if err != nil {
			return nil, err
		}
		dek, err := keyset.ReadWithAssociatedData(keyset.NewBinaryReader(bytes.NewReader(e.Blob)), w, wrapAssociatedData(safe))
		// AES-GCM cannot tell a wrong phrase from a tampered blob.
		if err != nil {
			return nil, ErrWrongUnlockMaterial
		}
		return dek, nil
	}
	return nil, fmt.Errorf("safe %s has no key wrapped by %q", safe.Id.String(), k.kind)
}

func (k phraseKeys) open(safe *happydns.Safe, phrase string) (UnlockedSafe, error) {
	dek, err := k.unwrap(safe, phrase)
	if err != nil {
		return nil, err
	}
	u, err := newAEADSafe(safe.Id, dek)
	if err != nil {
		return nil, err
	}
	return &phraseUnlocked{UnlockedSafe: u, dek: dek}, nil
}

// phraseUnlocked is a safe opened with its phrase, keeping its key, so that
// it can be wrapped again under another phrase while it is open.
type phraseUnlocked struct {
	UnlockedSafe
	dek *keyset.Handle
}

// readFields reads the fields of a JSON object following a form.
func readFields(raw json.RawMessage) (map[string]string, error) {
	var p map[string]string
	err := json.Unmarshal(raw, &p)
	return p, err
}

// unlockForm is the form opening a safe with its phrase.
func (k phraseKeys) unlockForm(label string) []happydns.Field {
	return []happydns.Field{{Id: k.field, Type: "string", Label: label, Secret: true, Required: true}}
}

// unlockWith opens safe with the phrase material holds, or fails with
// ErrWrongUnlockMaterial.
func (k phraseKeys) unlockWith(safe *happydns.Safe, material json.RawMessage) (UnlockedSafe, error) {
	p, err := readFields(material)
	if err != nil {
		return nil, ErrWrongUnlockMaterial
	}
	return k.open(safe, p[k.field])
}

// errNeedsOwner is returned by an interactive store asked to open a safe
// without its owner.
var errNeedsOwner = errors.New("an interactive safe is not opened without its owner")

// phraseStore keeps personal keystores, each locked by a passphrase its
// owner chooses: forgotten, what the keystore holds is lost.
type phraseStore struct{ keys phraseKeys }

func newPassphraseStore(Config) (Store, error) { return newPhraseStore(), nil }

func newPhraseStore() *phraseStore {
	return &phraseStore{phraseKeys{kind: KindPassphrase, field: passphraseField}}
}

func (s *phraseStore) Unlock(context.Context, *happydns.Safe) (UnlockedSafe, error) {
	return nil, errNeedsOwner
}

func (s *phraseStore) CreateForm() []happydns.Field {
	return []happydns.Field{
		{
			Id:          passphraseField,
			Type:        "string",
			Label:       "Passphrase",
			Description: fmt.Sprintf("At least %d characters. Nobody can recover it: forgotten, the credentials this keystore holds are lost.", minPassphraseLength),
			Secret:      true,
			Required:    true,
		},
		{Id: passphraseConfirmField, Type: "string", Label: "Confirm the passphrase", Secret: true, Required: true},
	}
}

func (s *phraseStore) UnlockForm() []happydns.Field { return s.keys.unlockForm("Passphrase") }

// Create locks safe with the passphrase params holds, once confirmed.
func (s *phraseStore) Create(_ context.Context, safe *happydns.Safe, params json.RawMessage) error {
	p, err := readFields(params)
	if err != nil {
		return err
	}
	phrase := p[passphraseField]
	switch {
	case len([]rune(phrase)) < minPassphraseLength:
		return fmt.Errorf("the passphrase must be at least %d characters long", minPassphraseLength)
	case phrase != p[passphraseConfirmField]:
		return errors.New("the passphrase and its confirmation are different")
	}
	return s.keys.bind(safe, phrase)
}

func (s *phraseStore) UnlockWith(_ context.Context, safe *happydns.Safe, material json.RawMessage) (UnlockedSafe, error) {
	return s.keys.unlockWith(safe, material)
}

// KindLoginPassword is a personal keystore whose key is wrapped by the login
// password of its owner: unlocked at login, and bound again when the
// password changes.
const KindLoginPassword = "login-password"

// loginStore keeps the personal keystores bound to the login password of
// their owner. After a password reset, one stays locked: it opens with the
// password it is bound to.
type loginStore struct{ keys phraseKeys }

func newLoginPasswordStore(Config) (Store, error) { return newLoginStore(), nil }

func newLoginStore() *loginStore {
	return &loginStore{phraseKeys{kind: KindLoginPassword, field: PasswordParam}}
}

func (s *loginStore) Unlock(context.Context, *happydns.Safe) (UnlockedSafe, error) {
	return nil, errNeedsOwner
}

func (s *loginStore) UnlockForm() []happydns.Field { return passwordForm() }

func (s *loginStore) UnlockWith(_ context.Context, safe *happydns.Safe, material json.RawMessage) (UnlockedSafe, error) {
	return s.keys.unlockWith(safe, material)
}

func (s *loginStore) Bind(_ context.Context, safe *happydns.Safe, password string) error {
	return s.keys.bind(safe, password)
}

func (s *loginStore) UnlockWithPassword(_ context.Context, safe *happydns.Safe, password string) (UnlockedSafe, error) {
	return s.keys.open(safe, password)
}

// RebindUnlocked wraps under new the key of safe, taken from u, opened by this
// store, without the password safe is bound to: after a reset, the session
// that unlocked it can still make it follow the new password.
func (s *loginStore) RebindUnlocked(_ context.Context, safe *happydns.Safe, u UnlockedSafe, new string) error {
	if new == "" {
		return errors.New("the new password is required")
	}
	pu, ok := u.(*phraseUnlocked)
	if !ok {
		return fmt.Errorf("safe %s was not opened by this store", safe.Id.String())
	}
	w, err := s.keys.wrap(safe, pu.dek, new)
	if err != nil {
		return err
	}
	safe.Keyring = []happydns.WrappedKeyset{w}
	return nil
}

// Rebind wraps again the key of safe under new, keeping the key: what is
// sealed stays readable.
func (s *loginStore) Rebind(_ context.Context, safe *happydns.Safe, old, new string) error {
	if new == "" {
		return errors.New("the new password is required")
	}
	dek, err := s.keys.unwrap(safe, old)
	if err != nil {
		return err
	}
	w, err := s.keys.wrap(safe, dek, new)
	if err != nil {
		return err
	}
	safe.Keyring = []happydns.WrappedKeyset{w}
	return nil
}
