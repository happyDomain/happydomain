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

	"git.happydns.org/happyDomain/model"
)

// secretFromValue reads a stored value the way UnmarshalJSON does: sealed
// when it carries the prefix, clear otherwise.
func secretFromValue(value string) (happydns.Secret, error) {
	var s happydns.Secret
	b, err := json.Marshal(value)
	if err != nil {
		return s, err
	}
	err = s.UnmarshalJSON(b)
	return s, err
}

// SealValue returns value, a secret stored in a map rather than in a
// struct, sealed as SealValueInto does without any keystore chosen.
func (m *Manager) SealValue(ctx context.Context, sc SecretContext, value string) (string, error) {
	return m.SealValueInto(ctx, sc, value, Destination{})
}

// SealValueInto returns value, a secret stored in a map rather than in a
// struct, sealed into the keystore dst resolves to: dst.Current lists the
// safes the other values of the object use. A value already sealed there is
// returned as is, one sealed elsewhere moved, the placeholder refused. sc
// must name the field.
func (m *Manager) SealValueInto(ctx context.Context, sc SecretContext, value string, dst Destination) (string, error) {
	if value == happydns.RedactedSecret {
		return "", ErrRedactedSecret
	}
	s, err := secretFromValue(value)
	if err != nil {
		return "", err
	}
	if err := m.SealSecretInto(ctx, sc, &s, dst); err != nil {
		return "", err
	}
	if s.IsSealed() {
		return value, nil
	}
	return s.Token(), nil
}

// OpenValue returns value, as stored, in clear. sc must name the field.
func (m *Manager) OpenValue(ctx context.Context, sc SecretContext, value string) (string, error) {
	if !IsSealed(value) {
		return value, nil
	}
	s, err := secretFromValue(value)
	if err != nil {
		return "", err
	}
	if err := m.OpenSecret(ctx, sc, &s); err != nil {
		return "", err
	}
	return s.Reveal(), nil
}

// InspectValue adds how value, as stored, is protected to c.
func (m *Manager) InspectValue(ctx context.Context, sc SecretContext, value string, c *Counts) error {
	if m == nil {
		return errNoManager
	}
	if err := sc.Validate(); err != nil {
		return err
	}
	s, err := secretFromValue(value)
	if err != nil {
		return err
	}
	return m.inspectOne(ctx, sc, &s, unlockedSafes{}, c)
}

// ResealValue returns value, as stored, the way the current policy stores new
// secrets, and whether that differs from what is stored. The placeholder,
// stored by mistake, stands for no value: it is refused.
func (m *Manager) ResealValue(ctx context.Context, sc SecretContext, value string) (string, bool, error) {
	if m == nil {
		return "", false, errNoManager
	}
	if value == happydns.RedactedSecret {
		return "", false, ErrRedactedSecret
	}

	sealed := IsSealed(value)
	switch {
	case value == "":
		return value, false, nil
	case sealed && m.policy != PolicyPlaintext, !sealed && m.policy == PolicyPlaintext:
		return value, false, nil
	}

	clear, err := m.OpenValue(ctx, sc, value)
	if err != nil {
		return "", false, err
	}
	out, err := m.SealValue(ctx, sc, clear)
	if err != nil {
		return "", false, err
	}
	return out, true, nil
}
