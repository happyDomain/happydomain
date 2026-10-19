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
	"time"

	"git.happydns.org/happyDomain/model"
)

const (
	// ScopeInstance is a keystore the instance creates for each owner, on
	// first use, from a Provisioner: only the administrator manages it.
	ScopeInstance = "instance"

	// ScopePersonal is a keystore its owner created, from a Creator.
	ScopePersonal = "personal"

	// maxKeystoreName is the longest name of a personal keystore.
	maxKeystoreName = 64
)

var (
	// ErrUnknownKeystore is returned for a reference naming no keystore of
	// the owner.
	ErrUnknownKeystore = errors.New("unknown keystore")

	// ErrKeystoreNotUnattended is returned when a keystore that needs its
	// owner is chosen for an object used in the background.
	ErrKeystoreNotUnattended = errors.New("this keystore does not open without its owner, and this object is used in the background")

	// ErrInstanceKeystore is returned when an owner would delete a keystore
	// the instance manages.
	ErrInstanceKeystore = errors.New("keystores of the instance are managed by the administrator")

	// ErrInvalidKeystore is returned for a keystore that cannot be created
	// as asked.
	ErrInvalidKeystore = errors.New("invalid keystore")

	// errNoDefaults is returned when no storage keeps the default keystores.
	errNoDefaults = errors.New("no storage for the default keystores")
)

// Keystore describes, to its owner, a place their secrets can be sealed in.
type Keystore struct {
	// Ref names the keystore: the kind of its store for a keystore of the
	// instance, which may not be created yet, the identifier of its safe for
	// a personal one.
	Ref string `json:"ref"`

	Kind string `json:"kind"`

	// Name is set for a personal keystore.
	Name string `json:"name,omitempty"`

	// Scope is ScopeInstance or ScopePersonal.
	Scope string `json:"scope"`

	// Unattended tells whether the keystore opens without its owner, as
	// objects used in the background need.
	Unattended bool `json:"unattended"`

	// Available tells whether the store of the keystore is configured here:
	// otherwise, what it holds does not open.
	Available bool `json:"available"`

	// Default tells whether new secrets go there when nothing is chosen.
	Default bool `json:"default"`

	// SafeId is the safe behind the keystore, empty for a keystore of the
	// instance not created yet.
	SafeId happydns.Identifier `json:"safeId,omitempty" swaggertype:"string"`

	CreatedAt *time.Time `json:"createdAt,omitempty"`
}

// KindInfo describes a kind of personal keystore an owner may create.
type KindInfo struct {
	Kind string `json:"kind"`

	// Unattended tells whether its keystores open without their owner.
	Unattended bool `json:"unattended"`

	// Form describes the parameters of its creation.
	Form []happydns.Field `json:"form"`
}

// isPersonal tells whether safe is a keystore its owner created: the safes
// the instance creates have no name.
func isPersonal(safe *happydns.Safe) bool {
	return safe.Name != ""
}

func (m *Manager) describe(safe *happydns.Safe) Keystore {
	store, available := m.stores[safe.Kind]
	ks := Keystore{
		Ref:        safe.Id.String(),
		Kind:       safe.Kind,
		Name:       safe.Name,
		Scope:      ScopePersonal,
		Unattended: available && opensUnattended(store),
		Available:  available,
		SafeId:     safe.Id,
	}
	if !safe.CreatedAt.IsZero() {
		createdAt := safe.CreatedAt
		ks.CreatedAt = &createdAt
	}
	if !isPersonal(safe) {
		ks.Ref = safe.Kind
		ks.Scope = ScopeInstance
	}
	return ks
}

// Keystores lists the keystores of owner: those of the instance, created or
// not, then their personal ones.
func (m *Manager) Keystores(ctx context.Context, owner happydns.Identifier) ([]Keystore, error) {
	if m == nil {
		return nil, errNoManager
	}

	var safes []*happydns.Safe
	if m.safes != nil {
		var err error
		if safes, err = m.safes.ListOwnerSafes(owner); err != nil {
			return nil, err
		}
	}
	slices.SortFunc(safes, func(a, b *happydns.Safe) int { return a.CreatedAt.Compare(b.CreatedAt) })

	instance := map[string]Keystore{}
	var personal []Keystore
	for _, safe := range safes {
		ks := m.describe(safe)
		if ks.Scope == ScopePersonal {
			personal = append(personal, ks)
		} else if _, seen := instance[ks.Ref]; !seen {
			instance[ks.Ref] = ks
		}
	}
	for kind, store := range m.stores {
		if _, ok := store.(Provisioner); !ok || m.safes == nil {
			continue
		}
		if _, seen := instance[kind]; !seen {
			instance[kind] = Keystore{Ref: kind, Kind: kind, Scope: ScopeInstance, Unattended: opensUnattended(store), Available: true}
		}
	}

	def, err := m.effectiveDefault(owner)
	if err != nil {
		return nil, err
	}

	list := make([]Keystore, 0, len(instance)+len(personal))
	for _, kind := range slices.Sorted(maps.Keys(instance)) {
		list = append(list, instance[kind])
	}
	list = append(list, personal...)
	for i := range list {
		list[i].Default = list[i].Ref == def
	}
	return list, nil
}

// CreatableKinds lists the kinds of personal keystore owners may create here.
func (m *Manager) CreatableKinds() []KindInfo {
	if m == nil {
		return nil
	}
	var kinds []KindInfo
	for _, kind := range slices.Sorted(maps.Keys(m.stores)) {
		if c, ok := m.stores[kind].(Creator); ok {
			kinds = append(kinds, KindInfo{Kind: kind, Unattended: opensUnattended(c), Form: c.CreateForm()})
		}
	}
	return kinds
}

// CreateKeystore creates a personal keystore of kind for owner, named name,
// from params, the parameters its store needs.
func (m *Manager) CreateKeystore(ctx context.Context, owner happydns.Identifier, kind, name string, params json.RawMessage) (Keystore, error) {
	if m == nil || m.safes == nil {
		return Keystore{}, errNoManager
	}

	creator, ok := m.stores[kind].(Creator)
	if !ok {
		return Keystore{}, fmt.Errorf("%w: no store here creates keystores of kind %q", ErrInvalidKeystore, kind)
	}

	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return Keystore{}, fmt.Errorf("%w: a keystore needs a name", ErrInvalidKeystore)
	case len(name) > maxKeystoreName:
		return Keystore{}, fmt.Errorf("%w: a keystore name holds at most %d bytes", ErrInvalidKeystore, maxKeystoreName)
	}

	// One mutex per owner, so that two requests do not both take a name.
	mu := m.ownerLock("keystores:" + owner.String())
	mu.Lock()
	defer mu.Unlock()

	safes, err := m.safes.ListOwnerSafes(owner)
	if err != nil {
		return Keystore{}, err
	}
	for _, safe := range safes {
		if strings.EqualFold(safe.Name, name) {
			return Keystore{}, fmt.Errorf("%w: a keystore named %q already exists", ErrInvalidKeystore, safe.Name)
		}
	}

	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		return Keystore{}, err
	}
	safe := &happydns.Safe{
		Id:        id,
		Owner:     owner,
		Kind:      kind,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	if len(params) == 0 {
		params = json.RawMessage("{}")
	}
	if err := creator.Create(ctx, safe, params); err != nil {
		return Keystore{}, fmt.Errorf("%w: %w", ErrInvalidKeystore, err)
	}
	if err := m.safes.CreateSafe(safe); err != nil {
		return Keystore{}, err
	}
	return m.describe(safe), nil
}

// keystoreSafe returns the safe ref names among those of owner: nil, without
// error, for a keystore of the instance not created yet.
func (m *Manager) keystoreSafe(owner happydns.Identifier, ref string) (*happydns.Safe, error) {
	if m.safes == nil {
		return nil, fmt.Errorf("%w %q", ErrUnknownKeystore, ref)
	}

	// A keystore of the instance is named after its kind, configured here
	// or not: what a kind left out holds can still be moved out of it.
	safes, err := m.safes.ListOwnerSafes(owner)
	if err != nil {
		return nil, err
	}
	var found *happydns.Safe
	for _, safe := range safes {
		if !isPersonal(safe) && safe.Kind == ref && (found == nil || safe.CreatedAt.Before(found.CreatedAt)) {
			found = safe
		}
	}
	if found != nil {
		return found, nil
	}
	if _, ok := m.stores[ref].(Provisioner); ok {
		return nil, nil
	}

	for _, safe := range safes {
		// The identifier of a safe tells nothing of whose it is: one of
		// another owner is as unknown as one that does not exist.
		if isPersonal(safe) && safe.Id.String() == ref {
			return safe, nil
		}
	}
	return nil, fmt.Errorf("%w %q", ErrUnknownKeystore, ref)
}

// KeystoreSafe returns the identifier of the safe behind ref, a keystore of
// owner, or happydns.ErrSafeNotFound for a keystore of the instance not
// created yet.
func (m *Manager) KeystoreSafe(_ context.Context, owner happydns.Identifier, ref string) (happydns.Identifier, error) {
	if m == nil {
		return nil, errNoManager
	}
	safe, err := m.keystoreSafe(owner, ref)
	if err != nil {
		return nil, err
	}
	if safe == nil {
		return nil, happydns.ErrSafeNotFound
	}
	return safe.Id, nil
}

// DefaultKeystore returns the keystore owner chose for new secrets, empty
// when they chose none.
func (m *Manager) DefaultKeystore(owner happydns.Identifier) (string, error) {
	if m == nil {
		return "", errNoManager
	}
	if m.defaults == nil {
		return "", nil
	}
	ref, err := m.defaults.GetDefaultKeystore(owner)
	if errors.Is(err, happydns.ErrNotFound) {
		return "", nil
	}
	return ref, err
}

// effectiveDefault returns where new secrets of owner go when nothing is
// chosen: the keystore they chose, else the one of the policy; empty under
// the plaintext policy.
func (m *Manager) effectiveDefault(owner happydns.Identifier) (string, error) {
	def, err := m.DefaultKeystore(owner)
	if err != nil || def != "" {
		return def, err
	}
	if m.policy == PolicyPlaintext {
		return "", nil
	}
	return string(m.policy), nil
}
