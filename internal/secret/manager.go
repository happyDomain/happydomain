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
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"git.happydns.org/happyDomain/model"
)

// Policy says how new secrets are sealed. Existing ones open whatever the
// current policy: the sealed value names what opens it.
type Policy string

const (
	// PolicyPlaintext stores secrets in clear, as happyDomain always did.
	PolicyPlaintext Policy = "plaintext"

	// PolicyInstance encrypts secrets under a key held by the instance.
	PolicyInstance Policy = "instance"
)

var (
	// ErrRedactedSecret is returned when the placeholder the API sends in
	// place of a secret reaches sealing or opening: it should have been
	// merged with the stored value before.
	ErrRedactedSecret = errors.New("a redacted secret placeholder was not merged with the stored value")

	// ErrSealedFromClient is returned by CheckIncoming for a sealed value
	// sent by a client, which has no reason to send one: accepting it would
	// let a user paste a token taken from elsewhere.
	ErrSealedFromClient = errors.New("sealed values cannot be submitted")

	// errNoManager is returned by a nil Manager: failing is better than
	// guessing a policy for a component wired without one.
	errNoManager = errors.New("no secret manager configured")

	// ErrUnknownSafe is returned when a sealed value names a safe that does
	// not exist.
	ErrUnknownSafe = errors.New("unknown safe")

	// ErrInstanceOwner is returned when the safes of the instance itself
	// would be deleted as those of a user, whose identifier equals it.
	ErrInstanceOwner = errors.New("the safes of the instance itself are not deleted with a user")
)

// reservedPrefix starts every stored form happyDomain gives a meaning to,
// SealedSecretPrefix and those to come: no value starting with it is stored
// in clear, so that none is misread once a new format claims it.
const reservedPrefix = "hds:"

// Manager seals secrets before they are stored, and opens them where
// happyDomain uses them on behalf of their owner.
type Manager struct {
	policy Policy

	// safes is nil when no safe can be opened: no storage.
	safes SafeStorage

	// stores holds the store of each kind of safe configured, by kind.
	stores map[string]Store

	// locks holds one mutex per owner and kind, see ownerSafe.
	locks sync.Map
}

// Config tells a Manager how to seal new secrets and how to open stored ones.
type Config struct {
	// Policy says how new secrets are sealed.
	Policy Policy

	// InstanceKey opens the instance safes. Without it, their secrets
	// cannot be opened.
	InstanceKey *InstanceKey

	// Safes keeps the safes. Required by every store, and by a policy
	// other than plaintext.
	Safes SafeStorage

	// Stores lists the kinds of safe the administrator enabled: the others
	// are left out. "plaintext" is not a kind of safe but is listed too:
	// left out, it forbids the plaintext policy. Nil enables everything.
	Stores []string

	// factories replaces builtinStores, for tests.
	factories map[string]StoreFactory
}

// NewManager returns a Manager for cfg.
func NewManager(cfg Config) (*Manager, error) {
	m := &Manager{
		policy: cfg.Policy,
		safes:  cfg.Safes,
		stores: map[string]Store{},
	}

	factories := cfg.storeFactories()
	kinds := slices.Sorted(maps.Keys(factories))
	if cfg.Stores != nil {
		for _, kind := range cfg.Stores {
			if _, known := factories[kind]; !known && kind != string(PolicyPlaintext) {
				return nil, fmt.Errorf("unknown secret store %q", kind)
			}
		}
		kinds = slices.DeleteFunc(kinds, func(kind string) bool {
			return !slices.Contains(cfg.Stores, kind)
		})
	}

	for _, kind := range kinds {
		store, err := factories[kind](cfg)
		if errors.Is(err, ErrStoreUnavailable) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("secret store %q: %w", kind, err)
		}
		m.stores[kind] = store
	}

	if cfg.Policy == PolicyPlaintext {
		if cfg.Stores != nil && !slices.Contains(cfg.Stores, string(PolicyPlaintext)) {
			return nil, fmt.Errorf("the %q secret policy is not enabled: list it in the secret stores, or choose another policy", cfg.Policy)
		}
		return m, nil
	}

	if _, known := factories[string(cfg.Policy)]; !known {
		return nil, fmt.Errorf("unknown secret policy %q", cfg.Policy)
	}
	if cfg.Stores != nil && !slices.Contains(cfg.Stores, string(cfg.Policy)) {
		return nil, fmt.Errorf("the %q secret policy names a store that is not enabled", cfg.Policy)
	}
	store, ok := m.stores[string(cfg.Policy)]
	if !ok {
		return nil, fmt.Errorf("the %q secret policy needs its store, which is not configured", cfg.Policy)
	}
	if _, ok := store.(Provisioner); !ok {
		return nil, fmt.Errorf("the %q secret store does not create safes: it cannot be the policy", cfg.Policy)
	}
	if cfg.Safes == nil {
		return nil, fmt.Errorf("the %q secret policy requires a safe storage", cfg.Policy)
	}

	return m, nil
}

// Policy is how new secrets are sealed.
func (m *Manager) Policy() Policy {
	if m == nil {
		return ""
	}
	return m.policy
}

// SealObject seals every clear Secret of obj, a pointer to a struct, under
// the current policy. sc tells whose object it is; its Field is filled for
// each secret. Sealed and opened secrets are left as they are, a redacted one
// is an error.
//
// A sealed secret stays opened: obj can still be used, and is now storable.
func (m *Manager) SealObject(ctx context.Context, sc SecretContext, obj any) error {
	if m == nil {
		return errNoManager
	}
	if err := sc.validateObject(); err != nil {
		return err
	}

	// All the secrets of an object go to the same safe.
	x := &sealer{m: m, owner: sc.Owner}

	return Walk(obj, func(path string, s *happydns.Secret) error {
		fsc := sc
		fsc.Field = path
		return x.seal(ctx, fsc, s)
	})
}

// SealSecret seals s, a single secret, like SealObject does for each field.
// sc must name the field.
func (m *Manager) SealSecret(ctx context.Context, sc SecretContext, s *happydns.Secret) error {
	if m == nil {
		return errNoManager
	}
	if err := sc.Validate(); err != nil {
		return err
	}
	return (&sealer{m: m, owner: sc.Owner}).seal(ctx, sc, s)
}

// sealer seals the secrets of one owner, looking its safe up and unlocking
// it once.
type sealer struct {
	m        *Manager
	owner    happydns.Identifier
	safe     *happydns.Safe
	unlocked UnlockedSafe
}

func (x *sealer) seal(ctx context.Context, sc SecretContext, s *happydns.Secret) error {
	switch {
	case s.IsEmpty(), s.IsSealed(), s.IsOpened():
		return nil
	case s.IsRedacted():
		return ErrRedactedSecret
	}

	// Before any work: a safe may be created, or a store reached.
	if err := ctx.Err(); err != nil {
		return err
	}

	clear, ok := s.ClearForSealing()
	if !ok {
		return errors.New("secret in an unknown state")
	}

	if x.m.policy == PolicyPlaintext {
		if strings.HasPrefix(string(clear), reservedPrefix) {
			// Stored as is, it would read back as sealed, now or once a
			// format to come claims its prefix.
			return fmt.Errorf("a value starting with %q cannot be stored in clear", reservedPrefix)
		}
		if string(clear) == happydns.RedactedSecret {
			// Stored as is, it would read back as redacted.
			return errors.New("the redacted placeholder cannot be stored in clear")
		}
		s.SetOpened(string(clear), clear)
		return nil
	}

	if x.unlocked == nil {
		kind := string(x.m.policy)
		// Checked by NewManager.
		store := x.m.stores[kind].(Provisioner)

		safe, err := x.m.ownerSafe(ctx, x.owner, kind, store)
		if err != nil {
			return err
		}
		unlocked, err := store.Unlock(ctx, safe)
		if err != nil {
			return err
		}
		x.safe, x.unlocked = safe, unlocked
	}

	payload, err := x.unlocked.Seal(ctx, sc, clear)
	if err != nil {
		return err
	}
	s.SetOpened(FormatSealed(x.safe.Id, payload), clear)
	return nil
}

// OpenObject opens every sealed Secret of obj, a pointer to a struct. Clear
// ones, legacy plaintext, are left as they are. A secret that cannot be
// opened is an error.
func (m *Manager) OpenObject(ctx context.Context, sc SecretContext, obj any) error {
	if m == nil {
		return errNoManager
	}
	if err := sc.validateObject(); err != nil {
		return err
	}

	// Each safe is fetched and unlocked once per object, not once per field.
	unlocked := unlockedSafes{}

	return Walk(obj, func(path string, s *happydns.Secret) error {
		fsc := sc
		fsc.Field = path
		_, err := m.open(ctx, fsc, s, unlocked)
		return err
	})
}

// OpenSecret opens s, a single secret, like OpenObject does for each field.
// sc must name the field.
func (m *Manager) OpenSecret(ctx context.Context, sc SecretContext, s *happydns.Secret) error {
	if m == nil {
		return errNoManager
	}
	if err := sc.Validate(); err != nil {
		return err
	}
	_, err := m.open(ctx, sc, s, unlockedSafes{})
	return err
}

// OpenCopy returns a copy of obj, a pointer to a struct, with every secret
// opened. obj is left untouched, so that it keeps carrying sealed values only.
func (m *Manager) OpenCopy(ctx context.Context, sc SecretContext, obj any) (any, error) {
	if m == nil {
		return nil, errNoManager
	}
	cp, err := clone(obj)
	if err != nil {
		return nil, err
	}

	if err := m.OpenObject(ctx, sc, cp); err != nil {
		return nil, err
	}

	return cp, nil
}

// unlockedSafes holds the safes unlocked while going through one object, by
// safe identifier.
type unlockedSafes map[string]unlockedSafe

type unlockedSafe struct {
	kind string
	UnlockedSafe
}

// open opens s, and returns the kind of the safe it was sealed in. A secret
// that did not need opening is of no kind.
func (m *Manager) open(ctx context.Context, sc SecretContext, s *happydns.Secret, unlocked unlockedSafes) (string, error) {
	switch {
	case s.IsEmpty(), s.IsClear(), s.IsOpened():
		return "", nil
	case s.IsRedacted():
		return "", ErrRedactedSecret
	}

	if err := ctx.Err(); err != nil {
		return "", err
	}

	token := s.Token()
	sv, err := ParseSealed(token)
	if err != nil {
		return "", err
	}

	u, ok := unlocked[sv.SafeId.String()]
	if !ok {
		if m.safes == nil {
			return "", fmt.Errorf("%w %s", ErrUnknownSafe, sv.SafeId.String())
		}

		safe, err := m.safes.GetSafe(sv.SafeId)
		if errors.Is(err, happydns.ErrSafeNotFound) {
			return "", fmt.Errorf("%w %s", ErrUnknownSafe, sv.SafeId.String())
		}
		if err != nil {
			return "", err
		}

		// A store encrypting the secrets binds them to their owner through
		// the associated data; for one that does not, this is what keeps a
		// user from opening the safe of another.
		if !safe.Owner.Equals(sc.Owner) {
			return "", fmt.Errorf("safe %s does not belong to the owner of this secret", sv.SafeId.String())
		}

		us, err := m.unlock(ctx, safe)
		if err != nil {
			return "", err
		}
		u = unlockedSafe{kind: safe.Kind, UnlockedSafe: us}
		unlocked[sv.SafeId.String()] = u
	}

	clear, err := u.Open(ctx, sc, sv.Payload)
	if err != nil {
		return "", fmt.Errorf("unable to open secret %s: %w", sc.Field, err)
	}

	s.SetOpened(token, clear)
	return u.kind, nil
}

// CheckIncoming refuses an object coming from a client that holds a sealed
// value. Call it on what the client sent, before merging stored values in.
func CheckIncoming(obj any) error {
	return Walk(obj, func(_ string, s *happydns.Secret) error {
		if s.IsSealed() || s.IsOpened() {
			return ErrSealedFromClient
		}
		return nil
	})
}

// MarshalIncoming encodes an object as a client sent it, clear secrets
// included, so that it can be decoded again by code expecting a request body.
// obj is left as it was.
//
// The result holds secrets in clear: never store it.
func MarshalIncoming(obj any) ([]byte, error) {
	// On a copy: the secrets of a map cannot be put back as they were
	// through the pointers a walk hands out.
	cp, err := clone(obj)
	if err != nil {
		return nil, err
	}

	err = Walk(cp, func(_ string, s *happydns.Secret) error {
		if clear, ok := s.ClearForSealing(); ok {
			s.SetOpened(string(clear), clear)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(cp)
}
