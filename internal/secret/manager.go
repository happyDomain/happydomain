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

	"github.com/tink-crypto/tink-go/v2/tink"

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

	// ErrUnopenable is returned, wrapping the reason, for a sealed value
	// that will never open where it is: its safe is gone or is another
	// owner's, it was moved from elsewhere, or it is malformed. Retrying
	// does not help, entering the secret again does. A failure that may
	// pass, such as the storage being down, is not one.
	ErrUnopenable = errors.New("secret cannot be opened")

	// ErrSafeUnavailable is returned, wrapping the reason, for a sealed value
	// whose safe cannot be opened on this instance as configured: no
	// instance keyset, one lacking the key that wrapped the safe, a damaged
	// safe. Only the administrator can repair it; entering the secret again
	// does not, since it would be sealed in the same safe. The value is not
	// lost: it opens again once the configuration is repaired.
	ErrSafeUnavailable = errors.New("the safe of this secret cannot be opened on this instance")

	// ErrUnknownOwner is returned when a secret would need a new safe for an
	// owner that does not exist, such as a deleted user.
	ErrUnknownOwner = errors.New("owner does not exist")

	// ErrInstanceOwner is returned when the safes of the instance itself
	// would be deleted as those of a user, whose identifier equals it.
	ErrInstanceOwner = errors.New("the safes of the instance itself are not deleted with a user")
)

// Manager seals secrets before they are stored, and opens them where
// happyDomain uses them on behalf of their owner.
type Manager struct {
	policy Policy

	// safes is nil when no safe can be opened: no instance keyset and no
	// storage.
	safes *safeRegistry
}

// Config tells a Manager how to seal new secrets and how to open stored ones.
type Config struct {
	// Policy says how new secrets are sealed.
	Policy Policy

	// InstanceKey opens the instance safes. Without it, their secrets
	// cannot be opened.
	InstanceKey *InstanceKey

	// Safes keeps the safes. Required with InstanceKey.
	Safes SafeStorage

	// Owners, when set, is checked before a safe is created, so that none
	// is created for a user that no longer exists.
	Owners OwnerStorage
}

// OwnerStorage tells whether a user exists.
type OwnerStorage interface {
	// GetUser returns the user, or happydns.ErrUserNotFound.
	GetUser(id happydns.Identifier) (*happydns.User, error)
}

// NewManager returns a Manager for cfg.
func NewManager(cfg Config) (*Manager, error) {
	switch cfg.Policy {
	case PolicyPlaintext:
	case PolicyInstance:
		if cfg.InstanceKey == nil {
			return nil, errors.New("the instance secret policy requires an instance keyset")
		}
	default:
		return nil, fmt.Errorf("unknown secret policy %q", cfg.Policy)
	}

	if cfg.InstanceKey != nil && cfg.Safes == nil {
		return nil, errors.New("an instance keyset requires a safe storage")
	}

	m := &Manager{policy: cfg.Policy}
	if cfg.Safes != nil {
		m.safes = newSafeRegistry(cfg.Safes, cfg.InstanceKey)
		m.safes.owners = cfg.Owners
	}
	return m, nil
}

// SealObject seals every clear Secret of obj, a pointer to a struct, under
// the current policy. sc tells whose object it is; its Field is filled for
// each secret. A redacted secret is an error.
//
// A secret already opened stays as it is when it was opened for this very
// field, and is sealed again otherwise: its token would not open where it now
// lives. A sealed one stays as it is only once it is known to open here.
// Under the plaintext policy, both are stored in clear instead, like new
// values: nothing written then needs a safe to open.
//
// A sealed secret stays opened: obj can still be used, and is now storable.
// On error, obj is left as it was.
func (m *Manager) SealObject(ctx context.Context, sc SecretContext, obj any) error {
	if m == nil {
		return errNoManager
	}
	if err := sc.validateObject(); err != nil {
		return err
	}

	// All the secrets of an object go to the same safe.
	return transform(obj, sc, m.newSealer(sc.Owner).seal)
}

// SealSecret seals s, a single secret, like SealObject does for each field.
// sc must name the field. On error, s is left as it was.
func (m *Manager) SealSecret(ctx context.Context, sc SecretContext, s *happydns.Secret) error {
	if m == nil {
		return errNoManager
	}
	if err := sc.Validate(); err != nil {
		return err
	}

	val := *s
	if err := m.newSealer(sc.Owner).seal(sc, &val); err != nil {
		return err
	}
	*s = val
	return nil
}

// transform calls f on a copy of every Secret of obj, and writes the copies
// back only once every call succeeded: obj is never left half done.
func transform(obj any, sc SecretContext, f func(SecretContext, *happydns.Secret) error) error {
	type change struct {
		dst *happydns.Secret
		val happydns.Secret
	}
	var changes []change

	err := Walk(obj, func(path string, s *happydns.Secret) error {
		fsc := sc
		fsc.Field = path

		// A Secret is never modified in place: its methods replace it, so
		// this copy is independent.
		val := *s
		if err := f(fsc, &val); err != nil {
			return err
		}
		changes = append(changes, change{s, val})
		return nil
	})
	if err != nil {
		return err
	}

	for _, c := range changes {
		*c.dst = c.val
	}
	return nil
}

// sealer seals the secrets of one owner, looking its safe up once.
type sealer struct {
	m         *Manager
	owner     happydns.Identifier
	safe      *happydns.Safe
	primitive tink.AEAD

	// primitives opens the sealed values met, by safe identifier.
	primitives map[string]tink.AEAD

	// keepUnopenable keeps as they are the sealed values that will never
	// open, rather than failing: a reseal goes past them.
	keepUnopenable bool
}

// newSealer returns a sealer for the secrets of owner, sharing nothing with
// another one.
func (m *Manager) newSealer(owner happydns.Identifier) *sealer {
	return &sealer{m: m, owner: owner, primitives: map[string]tink.AEAD{}}
}

func (x *sealer) seal(sc SecretContext, s *happydns.Secret) error {
	switch {
	case s.IsEmpty():
		return nil
	case s.IsRedacted():
		return ErrRedactedSecret
	case s.IsSealed():
		// Keep it only if it opens here.
		probe := *s
		if err := x.m.open(sc, &probe, x.primitives); x.keepUnopenable && errors.Is(err, ErrUnopenable) {
			return nil
		} else if err != nil {
			return err
		}
		if x.m.policy != PolicyPlaintext {
			return nil
		}
		// Under the plaintext policy, stored in clear like a new value:
		// once nothing is found sealed, no write brings back a value only
		// a safe opens, and the safes can be dropped.
		*s = happydns.NewSecret(probe.Reveal())
	case s.IsOpened():
		if s.Binding() == sc.binding() && (x.m.policy != PolicyPlaintext || !IsSealed(s.Token())) {
			return nil
		}
		// Opened elsewhere, or sealed while the policy is now plaintext:
		// store its value again the way new values are, for where it now
		// lives.
		*s = happydns.NewSecret(s.Reveal())
	}
	if s.IsEmpty() {
		return nil
	}

	clear, ok := s.ClearForSealing()
	if !ok {
		return errors.New("secret in an unknown state")
	}

	switch x.m.policy {
	case PolicyPlaintext:
		if IsSealed(string(clear)) {
			// Stored as is, it would read back as sealed.
			return fmt.Errorf("a value starting with %q cannot be stored in clear", happydns.SealedSecretPrefix)
		}
		if string(clear) == happydns.RedactedSecret {
			// Stored as is, it would read back as redacted.
			return errors.New("the redacted placeholder cannot be stored in clear")
		}
		s.SetOpened(string(clear), clear, sc.binding())
		return nil

	case PolicyInstance:
		if x.safe == nil {
			var err error
			if x.safe, err = x.m.safes.instanceSafe(x.owner); err != nil {
				return err
			}
			if x.primitive, err = x.m.safes.aead(x.safe); err != nil {
				return err
			}
		}

		ct, err := x.primitive.Encrypt(clear, AssociatedData(x.safe.Id, sc))
		if err != nil {
			return err
		}
		s.SetOpened(FormatSealed(x.safe.Id, ct), clear, sc.binding())
		return nil
	}

	return fmt.Errorf("unknown secret policy %q", x.m.policy)
}

// OpenObject opens every sealed Secret of obj, a pointer to a struct. Clear
// ones, legacy plaintext, are left as they are. A secret that cannot be
// opened is an error, and obj is then left as it was.
func (m *Manager) OpenObject(ctx context.Context, sc SecretContext, obj any) error {
	if m == nil {
		return errNoManager
	}
	if err := sc.validateObject(); err != nil {
		return err
	}

	// The primitives of the safes met so far, by safe identifier.
	primitives := map[string]tink.AEAD{}

	return transform(obj, sc, func(fsc SecretContext, s *happydns.Secret) error {
		return m.open(fsc, s, primitives)
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
	return m.open(sc, s, map[string]tink.AEAD{})
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

func (m *Manager) open(sc SecretContext, s *happydns.Secret, primitives map[string]tink.AEAD) error {
	switch {
	case s.IsEmpty(), s.IsClear(), s.IsOpened():
		return nil
	case s.IsRedacted():
		return ErrRedactedSecret
	}

	token := s.Token()
	sv, err := ParseSealed(token)
	if err != nil {
		return unopenable(err)
	}

	primitive, ok := primitives[sv.SafeId.String()]
	if !ok {
		if m.safes == nil {
			// Without safe storage, whether the safe exists is unknown:
			// the configuration is what is missing.
			return safeUnavailable(fmt.Errorf("safe %s: no safe storage configured", sv.SafeId.String()))
		}

		safe, err := m.safes.store.GetSafe(sv.SafeId)
		if errors.Is(err, happydns.ErrSafeNotFound) {
			return unopenable(fmt.Errorf("%w %s", ErrUnknownSafe, sv.SafeId.String()))
		}
		if err != nil {
			// Maybe the storage is down: not known to be for good.
			return err
		}

		// The associated data already binds the owner; this only makes the
		// error clearer.
		if !safe.Owner.Equals(sc.Owner) {
			return unopenable(fmt.Errorf("safe %s does not belong to the owner of this secret", sv.SafeId.String()))
		}

		if primitive, err = m.safes.aead(safe); err != nil {
			// The keyset is missing or lacks the key of this safe, or the
			// safe is damaged: sealing again would hit the same safe.
			return safeUnavailable(err)
		}
		primitives[sv.SafeId.String()] = primitive
	}

	clear, err := primitive.Decrypt(sv.Payload, AssociatedData(sv.SafeId, sc))
	if err != nil {
		return unopenable(fmt.Errorf("unable to open secret %s: %w", sc.Field, err))
	}

	s.SetOpened(token, clear, sc.binding())
	return nil
}

// unopenable marks err as the reason a sealed value will never open.
func unopenable(err error) error {
	return fmt.Errorf("%w: %w", ErrUnopenable, err)
}

// safeUnavailable marks err as the reason the safe of a sealed value cannot
// be opened on this instance.
func safeUnavailable(err error) error {
	return fmt.Errorf("%w: %w", ErrSafeUnavailable, err)
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
// obj is left untouched: the clear secrets are marked encodable in a copy.
//
// The result holds secrets in clear: never store it.
func MarshalIncoming(obj any) ([]byte, error) {
	cp, err := clone(obj)
	if err != nil {
		return nil, err
	}

	err = Walk(cp, func(_ string, s *happydns.Secret) error {
		if clear, ok := s.ClearForSealing(); ok {
			s.SetOpened(string(clear), clear, "")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(cp)
}
