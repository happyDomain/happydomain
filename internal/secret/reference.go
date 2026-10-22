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
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

var (
	// ErrReferencesOnly is returned when a secret would move into a
	// keystore keeping references: the secret would be taken for one.
	ErrReferencesOnly = errors.New("this keystore keeps references to secrets kept outside HappyDomain, not secrets: enter a reference")

	// ErrReferenceAsSecret is returned when the reference a keystore keeps
	// would be sealed, sent back as shown, into one keeping secrets.
	ErrReferenceAsSecret = errors.New("this is the reference the previous keystore kept, not a secret: enter the secret itself")

	// ErrExtractionForbidden is returned when a secret would leave a
	// keystore of references to be sealed in happyDomain, which the
	// administrator did not allow.
	ErrExtractionForbidden = errors.New("secrets cannot be extracted from a keystore of references on this instance")

	// ErrUnresolvedReference is returned when a keystore keeping
	// references refuses the one entered, with its reason, which never
	// repeats the reference.
	ErrUnresolvedReference = errors.New("this reference does not resolve")
)

// IsReferenceError tells whether err refuses what the owner entered in a
// secret field, as a reference or in place of one: theirs to fix, its reason
// is told them.
func IsReferenceError(err error) bool {
	return errors.Is(err, ErrUnresolvedReference) || errors.Is(err, ErrReferencesOnly) || errors.Is(err, ErrReferenceAsSecret) || errors.Is(err, ErrExtractionForbidden)
}

// referencing returns the store of kind when it keeps references.
func (m *Manager) referencing(kind string) (Referencing, bool) {
	r, ok := m.stores[kind].(Referencing)
	return r, ok
}

// KeepsReferences tells whether ref, a keystore of owner, keeps references to
// secrets kept elsewhere. A keystore of the instance not created yet holds
// nothing, so it does not.
func (m *Manager) KeepsReferences(_ context.Context, owner happydns.Identifier, ref string) (bool, error) {
	if m == nil {
		return false, errNoManager
	}
	safe, err := m.keystoreSafe(owner, ref)
	if err != nil {
		return false, err
	}
	if safe == nil {
		return false, nil
	}
	_, ok := m.referencing(safe.Kind)
	return ok, nil
}

// checkNotReference returns ErrExtractionForbidden when the safe id is the one
// of a keystore keeping references.
func (m *Manager) checkNotReference(id happydns.Identifier, sc SecretContext) error {
	if m.safes == nil {
		return nil
	}
	safe, err := m.safes.GetSafe(id)
	if errors.Is(err, happydns.ErrSafeNotFound) {
		// Nothing left to resolve: opening the secret fails on its own.
		return nil
	}
	if err != nil {
		return err
	}
	if _, ok := m.referencing(safe.Kind); ok {
		return fmt.Errorf("%w: %s", ErrExtractionForbidden, sc.Field)
	}
	return nil
}

// CheckMigrationTarget returns an error when the secrets of owner cannot be
// migrated into ref, one of their keystores: as for CheckKeystore, or when
// it keeps references, which no secret stands for.
func (m *Manager) CheckMigrationTarget(_ context.Context, owner happydns.Identifier, ref string) error {
	if m == nil {
		return errNoManager
	}
	x, err := m.target(owner, ref, false)
	if err != nil {
		return err
	}
	if _, ok := m.referencing(x.kind); ok {
		return fmt.Errorf("%w: secrets cannot be migrated into keystore %q", ErrReferencesOnly, ref)
	}
	return nil
}

// referenceSafes caches the safes looked up while reading references: a walk
// shares the lookups of the map it is given.
type referenceSafes map[string]*happydns.Safe

// referenceOf returns the reference s, as stored, holds, when it is sealed in
// a keystore of sc.Owner keeping references, and the kind of that keystore. It
// never resolves the reference.
func (m *Manager) referenceOf(sc SecretContext, s happydns.Secret, safes referenceSafes) (ref, kind string, ok bool, err error) {
	if m.safes == nil || (!s.IsSealed() && !s.IsOpened()) {
		return "", "", false, nil
	}
	sv, err := ParseSealed(s.Token())
	if err != nil {
		return "", "", false, nil
	}

	safe, seen := safes[sv.SafeId.String()]
	if !seen {
		safe, err = m.safes.GetSafe(sv.SafeId)
		if errors.Is(err, happydns.ErrSafeNotFound) {
			safe, err = nil, nil
		}
		if err != nil {
			return "", "", false, err
		}
		safes[sv.SafeId.String()] = safe
	}
	// A reference is shown only to the owner of its keystore: to anyone
	// else, it says where their secrets are kept.
	if safe == nil || !safe.Owner.Equals(sc.Owner) {
		return "", "", false, nil
	}

	store, isRef := m.referencing(safe.Kind)
	if !isRef {
		return "", "", false, nil
	}
	ref, err = store.Reference(sv.Payload)
	if err != nil {
		return "", safe.Kind, false, err
	}
	return ref, safe.Kind, true, nil
}

// ShowReferences makes every secret of obj, a pointer to a struct as read
// from storage, sealed in a keystore of sc.Owner keeping references, show
// that reference: see happydns.Secret.SetReference. The others are left for
// the redaction. No reference is resolved.
//
// Only on an object answering a request, never stored again.
func (m *Manager) ShowReferences(ctx context.Context, sc SecretContext, obj any) error {
	if m == nil {
		return errNoManager
	}
	if err := sc.validateObject(); err != nil {
		return err
	}
	safes := referenceSafes{}
	return Walk(obj, func(_ string, s *happydns.Secret) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		ref, _, ok, err := m.referenceOf(sc, *s, safes)
		if err != nil || !ok {
			// One that does not read stays withheld.
			return nil
		}
		s.SetReference(ref)
		return nil
	})
}

// ReferencesOf returns the references the secrets of obj, a pointer to a
// struct as read from storage, keep, by field: see Destination.References.
// obj is left untouched.
func (m *Manager) ReferencesOf(ctx context.Context, sc SecretContext, obj any) (map[string]string, error) {
	if m == nil {
		return nil, errNoManager
	}
	if err := sc.validateObject(); err != nil {
		return nil, err
	}
	cp, err := clone(obj)
	if err != nil {
		return nil, err
	}

	refs := map[string]string{}
	safes := referenceSafes{}
	err = Walk(cp, func(path string, s *happydns.Secret) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		ref, _, ok, err := m.referenceOf(sc, *s, safes)
		if err != nil {
			return err
		}
		if ok {
			refs[path] = ref
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return refs, nil
}
