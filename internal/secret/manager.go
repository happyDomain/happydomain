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
	"strings"

	"git.happydns.org/happyDomain/model"
)

// Policy says how new secrets are sealed. Existing ones open whatever the
// current policy: the sealed value names what opens it.
type Policy string

const (
	// PolicyPlaintext stores secrets in clear, as happyDomain always did.
	PolicyPlaintext Policy = "plaintext"
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

// reservedPrefix starts every stored form happyDomain gives a meaning to,
// SealedSecretPrefix and those to come: no value starting with it is stored
// in clear, so that none is misread once a new format claims it.
const reservedPrefix = "hds:"

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

	return Walk(obj, func(path string, s *happydns.Secret) error {
		fsc := sc
		fsc.Field = path
		return m.seal(ctx, fsc, s)
	})
}

func (m *Manager) seal(_ context.Context, _ SecretContext, s *happydns.Secret) error {
	switch {
	case s.IsEmpty(), s.IsSealed(), s.IsOpened():
		return nil
	case s.IsRedacted():
		return ErrRedactedSecret
	}

	clear, ok := s.ClearForSealing()
	if !ok {
		return errors.New("secret in an unknown state")
	}

	switch m.policy {
	case PolicyPlaintext:
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

	return fmt.Errorf("unknown secret policy %q", m.policy)
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

	return Walk(obj, func(path string, s *happydns.Secret) error {
		fsc := sc
		fsc.Field = path
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
// obj is left as it was.
//
// The result holds secrets in clear: never store it.
func MarshalIncoming(obj any) ([]byte, error) {
	type saved struct {
		s    *happydns.Secret
		orig happydns.Secret
	}
	var changed []saved

	defer func() {
		for _, c := range changed {
			*c.s = c.orig
		}
	}()

	err := Walk(obj, func(_ string, s *happydns.Secret) error {
		clear, ok := s.ClearForSealing()
		if !ok {
			return nil
		}
		changed = append(changed, saved{s, *s})
		s.SetOpened(string(clear), clear)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(obj)
}
