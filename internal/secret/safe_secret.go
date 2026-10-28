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
	"slices"

	"git.happydns.org/happyDomain/model"
)

// SafeSecretObjectType names, in secret contexts and reports, the safes
// keeping a secret of their own, see SecretKeeping.
const SafeSecretObjectType = "keystore"

// SafeSecret holds the secret of a safe, so that it is sealed, opened,
// inspected and moved as the secrets of any object are.
type SafeSecret struct {
	Secret happydns.Secret `json:"secret"`
}

// SafeSecretContext tells where the secret of safe belongs.
func SafeSecretContext(safe *happydns.Safe) SecretContext {
	return SecretContext{Owner: safe.Owner, ObjectType: SafeSecretObjectType, ObjectId: safe.Id.String()}
}

// sealSafeSecret seals the secret params holds, under the parameter keeper
// names, as the secret of safe, into the keystore dst resolves to: never
// into safe itself. It returns params with the secret opened again, as a
// reference resolves: what the store checks.
func (m *Manager) sealSafeSecret(ctx context.Context, safe *happydns.Safe, keeper SecretKeeping, params json.RawMessage, dst Destination) (json.RawMessage, error) {
	var p map[string]any
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("%w: invalid parameters", ErrInvalidKeystore)
	}
	clear, _ := p[keeper.SecretParam()].(string)
	if clear == "" {
		return nil, fmt.Errorf("%w: the %q parameter is required", ErrInvalidKeystore, keeper.SecretParam())
	}

	obj := &SafeSecret{Secret: happydns.NewSecret(clear)}
	dst.not = safe.Id.String()
	if err := m.SealObjectInto(ctx, SafeSecretContext(safe), obj, dst); err != nil {
		return nil, err
	}
	if err := m.refuseSafeCycle(safe, obj.Secret); err != nil {
		return nil, err
	}
	stored, err := storedForm(obj.Secret)
	if err != nil {
		return nil, err
	}
	safe.Secret = stored

	opened, err := m.openSafeSecret(ctx, safe, nil)
	if err != nil {
		return nil, err
	}
	p[keeper.SecretParam()] = string(opened)
	return json.Marshal(p)
}

// refuseSafeCycle refuses secret, sealed as the secret of safe, when the
// keystore it is sealed in is opened through safe: directly, or through the
// keystores keeping their own secret in between.
func (m *Manager) refuseSafeCycle(safe *happydns.Safe, secret happydns.Secret) error {
	id, ok := safeIdOf(secret)
	seen := map[string]bool{}
	for ok && m.safes != nil {
		if id.Equals(safe.Id) {
			return fmt.Errorf("%w: the keystore would open through the keystore whose secret it keeps", ErrInvalidKeystore)
		}
		if seen[id.String()] {
			// Planted, not stored by us: open reports it.
			return nil
		}
		seen[id.String()] = true
		next, err := m.safes.GetSafe(id)
		if errors.Is(err, happydns.ErrSafeNotFound) {
			return nil
		} else if err != nil {
			return err
		}
		id, ok = safeIdOf(next.Secret)
	}
	return nil
}

// storedForm returns s as storage keeps it: its value in clear is not. Under
// the plaintext policy the stored form reads back as clear, which cannot be
// encoded: s is then kept, opened, as it encodes to the raw value.
func storedForm(s happydns.Secret) (happydns.Secret, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return happydns.Secret{}, err
	}
	var stored happydns.Secret
	if err = json.Unmarshal(raw, &stored); err != nil {
		return happydns.Secret{}, err
	}
	if stored.IsClear() {
		return s, nil
	}
	return stored, nil
}

// openSafeSecret opens the secret of safe, through the safes already
// unlocked.
func (m *Manager) openSafeSecret(ctx context.Context, safe *happydns.Safe, unlocked unlockedSafes) ([]byte, error) {
	if unlocked == nil {
		unlocked = unlockedSafes{}
	}
	sc := SafeSecretContext(safe)
	sc.Field = "secret"
	s := safe.Secret
	if _, err := m.open(ctx, sc, &s, unlocked); err != nil {
		return nil, err
	}
	if s.IsEmpty() {
		return nil, fmt.Errorf("safe %s has no secret", safe.Id.String())
	}
	return []byte(s.Reveal()), nil
}

// errSafeCycle is returned for a safe whose secret is sealed, through
// others maybe, in itself.
var errSafeCycle = errors.New("the secret of a safe is sealed in a safe it opens")

// safeOpensUnattended tells whether safe opens without its owner: as its
// store does, and, for one keeping a secret, as the safe sealing it does.
func (m *Manager) safeOpensUnattended(safe *happydns.Safe) bool {
	seen := map[string]bool{}
	for {
		store, ok := m.stores[safe.Kind]
		if !ok || !opensUnattended(store) {
			return false
		}
		if _, ok := store.(SecretKeeping); !ok || !safe.Secret.IsSealed() && !safe.Secret.IsOpened() {
			return true
		}
		sv, err := ParseSealed(safe.Secret.Token())
		if err != nil {
			// Kept as it is, under the plaintext policy.
			return true
		}
		seen[safe.Id.String()] = true
		if seen[sv.SafeId.String()] || m.safes == nil {
			return false
		}
		if safe, err = m.safes.GetSafe(sv.SafeId); err != nil {
			return false
		}
	}
}

// SafeSecrets administers the secrets of the safes as those of any object
// are, see SecretsUsecase and KeystoreUsecase.
type SafeSecrets struct {
	m *Manager
}

// SafeSecrets returns the holder of the secrets of the safes.
func (m *Manager) SafeSecrets() *SafeSecrets {
	return &SafeSecrets{m: m}
}

// withSecret lists the safes keeping a secret; a record that does not
// decode is passed to damaged.
func (h *SafeSecrets) withSecret(damaged func(key string, err error)) ([]*happydns.Safe, error) {
	if h.m == nil || h.m.safes == nil {
		return nil, nil
	}
	var safes []*happydns.Safe
	err := eachSafe(h.m.safes, func(safe *happydns.Safe) {
		if !safe.Secret.IsEmpty() {
			safes = append(safes, safe)
		}
	}, damaged)
	return safes, err
}

// update runs fn on the safe id as stored now, under the lock of the
// keystores of its owner, and stores it when fn tells it changed.
func (h *SafeSecrets) update(id happydns.Identifier, fn func(safe *happydns.Safe, obj *SafeSecret) (bool, error)) (bool, error) {
	safe, err := h.m.safes.GetSafe(id)
	if err != nil {
		return false, err
	}
	mu := h.m.ownerLock("keystores:" + safe.Owner.String())
	mu.Lock()
	defer mu.Unlock()

	// Read again under the lock: an update may have stored another token.
	if safe, err = h.m.safes.GetSafe(id); err != nil {
		return false, err
	}
	obj := &SafeSecret{Secret: safe.Secret}
	changed, err := fn(safe, obj)
	if err != nil || !changed {
		return false, err
	}
	if safe.Secret, err = storedForm(obj.Secret); err != nil {
		return false, err
	}
	return true, h.m.safes.UpdateSafe(safe)
}

// InspectSecrets tells how the secrets of the safes are stored.
func (h *SafeSecrets) InspectSecrets(ctx context.Context) (Counts, error) {
	var counts Counts
	safes, err := h.withSecret(nil)
	for _, safe := range safes {
		obj := &SafeSecret{Secret: safe.Secret}
		if err := h.m.Inspect(ctx, SafeSecretContext(safe), obj, &counts); err != nil {
			counts.Undecodable++
		}
	}
	return counts, err
}

// ResealSecrets stores the secrets of the safes the way the current policy
// stores new ones. A safe that fails is reported and skipped.
func (h *SafeSecrets) ResealSecrets(ctx context.Context) (ResealReport, error) {
	report := ResealReport{ObjectType: SafeSecretObjectType}
	safes, err := h.withSecret(nil)
	if err != nil {
		return report, err
	}
	for _, safe := range safes {
		report.Processed++
		changed, err := h.update(safe.Id, func(safe *happydns.Safe, obj *SafeSecret) (bool, error) {
			return h.m.ResealObject(ctx, SafeSecretContext(safe), obj)
		})
		if err != nil && !errors.Is(err, happydns.ErrSafeNotFound) {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("keystore %s: %s", safe.Id.String(), err.Error()))
			continue
		}
		if changed {
			report.Changed++
		}
	}
	return report, nil
}

// OwnerObjects lists the keystores of owner keeping a sealed secret, with
// the safes it is sealed in.
func (h *SafeSecrets) OwnerObjects(_ context.Context, owner happydns.Identifier) ([]ObjectSafes, error) {
	if h.m == nil || h.m.safes == nil {
		return nil, nil
	}
	safes, err := h.m.safes.ListOwnerSafes(owner)
	if err != nil {
		return nil, err
	}
	var objects []ObjectSafes
	for _, safe := range safes {
		ids, err := SafesOf(&SafeSecret{Secret: safe.Secret})
		if err != nil || len(ids) == 0 {
			continue
		}
		objects = append(objects, ObjectSafes{ObjectType: SafeSecretObjectType, ObjectId: safe.Id.String(), Label: safe.Name, Safes: ids})
	}
	return objects, nil
}

// MoveOwnerSecrets moves into the keystore to the secrets of the keystores of
// owner sealed in the safe from: never into the keystore a secret opens. A
// safe that fails is reported and skipped.
func (h *SafeSecrets) MoveOwnerSecrets(ctx context.Context, owner, from happydns.Identifier, to string) (ResealReport, error) {
	report := ResealReport{ObjectType: SafeSecretObjectType}
	objects, err := h.OwnerObjects(ctx, owner)
	if err != nil {
		return report, err
	}
	for _, o := range objects {
		if !slices.ContainsFunc(o.Safes, from.Equals) {
			continue
		}
		report.Processed++
		id, err := happydns.NewIdentifierFromString(o.ObjectId)
		if err == nil {
			_, err = h.update(id, func(safe *happydns.Safe, obj *SafeSecret) (bool, error) {
				if !safe.Owner.Equals(owner) {
					return false, errors.New("keystore of another owner")
				}
				return true, h.m.SealObjectInto(ctx, SafeSecretContext(safe), obj, Destination{Keystore: to, not: safe.Id.String()})
			})
		}
		if err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("keystore %s: %s", o.ObjectId, err.Error()))
			continue
		}
		report.Changed++
	}
	return report, nil
}
