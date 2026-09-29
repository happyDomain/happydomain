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
	"fmt"

	"github.com/tink-crypto/tink-go/v2/tink"

	"git.happydns.org/happyDomain/model"
)

// Counts tells how the secrets of some objects are stored.
type Counts struct {
	// Clear counts the secrets stored in clear.
	Clear int `json:"clear"`

	// Sealed counts the secrets that open, by kind of safe.
	Sealed map[string]int `json:"sealed"`

	// Unreadable counts the sealed secrets that do not open.
	Unreadable int `json:"unreadable"`

	// Undecodable counts the objects that could not be looked at, such as
	// a provider of a type this version no longer knows. They may hold
	// sealed secrets.
	Undecodable int `json:"undecodable"`
}

// ResealReport tells what a reseal of every object of a type did.
type ResealReport struct {
	ObjectType string `json:"objectType"`

	// Processed counts the objects looked at.
	Processed int `json:"processed"`

	// Changed counts the objects written back.
	Changed int `json:"changed"`

	// Failed counts the objects left as they were because of an error.
	Failed int `json:"failed"`

	Errors []string `json:"errors,omitempty"`
}

// Inspect adds how the secrets of obj are stored to c. It tries to open each
// sealed one, in a copy: obj is left untouched.
func (m *Manager) Inspect(ctx context.Context, sc SecretContext, obj any, c *Counts) error {
	if m == nil {
		return errNoManager
	}
	if err := sc.validateObject(); err != nil {
		return err
	}
	if c.Sealed == nil {
		c.Sealed = map[string]int{}
	}

	cp, err := clone(obj)
	if err != nil {
		return err
	}

	primitives := map[string]tink.AEAD{}
	// The kind of each safe met, by safe identifier: read once.
	kinds := map[string]string{}

	return Walk(cp, func(path string, s *happydns.Secret) error {
		switch {
		case s.IsEmpty():
			return nil
		case s.IsClear():
			c.Clear++
			return nil
		case !s.IsSealed():
			return fmt.Errorf("unexpected secret state in a stored object")
		}

		fsc := sc
		fsc.Field = path

		sv, err := ParseSealed(s.Token())
		if err != nil {
			c.Unreadable++
			return nil
		}
		if err := m.open(fsc, s, primitives); err != nil {
			c.Unreadable++
			return nil
		}

		kind, ok := kinds[sv.SafeId.String()]
		if !ok {
			safe, err := m.safes.store.GetSafe(sv.SafeId)
			if err != nil {
				return err
			}
			kind = safe.Kind
			kinds[sv.SafeId.String()] = kind
		}
		c.Sealed[kind]++
		return nil
	})
}

// ResealObject stores the secrets of obj, as read from storage, the way the
// current policy stores new ones: sealing clear values under the instance
// policy, opening sealed ones to store them in clear under the plaintext
// policy. It reports whether obj changed and has to be written back; it
// fails, leaving obj untouched, when a secret does not open.
func (m *Manager) ResealObject(ctx context.Context, sc SecretContext, obj any) (bool, error) {
	if m == nil {
		return false, errNoManager
	}

	cp, err := clone(obj)
	if err != nil {
		return false, err
	}

	changed := false
	err = Walk(cp, func(_ string, s *happydns.Secret) error {
		switch {
		case s.IsClear():
			if m.policy != PolicyPlaintext {
				changed = true
			}
		case s.IsSealed():
			if m.policy == PolicyPlaintext {
				changed = true
			}
		}
		return nil
	})
	if err != nil || !changed {
		return false, err
	}

	if m.policy == PolicyPlaintext {
		if err := m.OpenObject(ctx, sc, cp); err != nil {
			return false, err
		}
		// Back to clear, so that sealing stores the value itself.
		err := Walk(cp, func(_ string, s *happydns.Secret) error {
			if s.IsOpened() {
				*s = happydns.NewSecret(s.Reveal())
			}
			return nil
		})
		if err != nil {
			return false, err
		}
	}

	if err := m.SealObject(ctx, sc, cp); err != nil {
		return false, err
	}

	// Only now that everything went through: obj is never left half done.
	src, dst := reflectElem(cp), reflectElem(obj)
	dst.Set(src)
	return true, nil
}

// Policy is how new secrets are sealed.
func (m *Manager) Policy() Policy {
	if m == nil {
		return ""
	}
	return m.policy
}
