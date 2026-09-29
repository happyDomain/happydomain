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
)

// Manager seals secrets before they are stored, and opens them where
// happyDomain uses them on behalf of their owner.
type Manager struct {
	policy Policy
}

// NewManager returns a Manager sealing new secrets under policy.
func NewManager(policy Policy) (*Manager, error) {
	switch policy {
	case PolicyPlaintext:
	default:
		return nil, fmt.Errorf("unknown secret policy %q", policy)
	}

	return &Manager{policy: policy}, nil
}

// SealObject seals every clear Secret of obj, a pointer to a struct, under
// the current policy. sc tells whose object it is; its Field is filled for
// each secret. A redacted secret is an error.
//
// A secret already opened stays as it is when it was opened for this very
// field, and is sealed again otherwise: its token would not open where it now
// lives. A sealed one stays as it is only once it is known to open here.
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

	return transform(obj, sc, func(fsc SecretContext, s *happydns.Secret) error {
		return m.seal(ctx, fsc, s)
	})
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

func (m *Manager) seal(ctx context.Context, sc SecretContext, s *happydns.Secret) error {
	switch {
	case s.IsEmpty():
		return nil
	case s.IsRedacted():
		return ErrRedactedSecret
	case s.IsSealed():
		// Its value is unknown, so it cannot be sealed again: keep it only
		// if it opens here.
		probe := *s
		return m.open(ctx, sc, &probe)
	case s.IsOpened():
		if s.Binding() == sc.binding() {
			return nil
		}
		// Opened elsewhere: seal its value again for where it now lives.
		*s = happydns.NewSecret(s.Reveal())
		if s.IsEmpty() {
			return nil
		}
	}

	clear, ok := s.ClearForSealing()
	if !ok {
		return errors.New("secret in an unknown state")
	}

	switch m.policy {
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
	}

	return fmt.Errorf("unknown secret policy %q", m.policy)
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

	return transform(obj, sc, func(fsc SecretContext, s *happydns.Secret) error {
		return m.open(ctx, fsc, s)
	})
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

func (m *Manager) open(_ context.Context, _ SecretContext, s *happydns.Secret) error {
	switch {
	case s.IsEmpty(), s.IsClear(), s.IsOpened():
		return nil
	case s.IsRedacted():
		return ErrRedactedSecret
	}

	sv, err := ParseSealed(s.Token())
	if err != nil {
		return err
	}

	return fmt.Errorf("%w %s", ErrUnknownSafe, sv.SafeId.String())
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
