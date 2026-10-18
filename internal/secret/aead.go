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

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/tink"

	"git.happydns.org/happyDomain/model"
)

// aeadSafe seals the secrets of a safe with its key, binding each to its
// place through AssociatedData.
type aeadSafe struct {
	id        happydns.Identifier
	primitive tink.AEAD
}

func newAEADSafe(id happydns.Identifier, dek *keyset.Handle) (UnlockedSafe, error) {
	primitive, err := aead.New(dek)
	if err != nil {
		return nil, err
	}
	return &aeadSafe{id: id, primitive: primitive}, nil
}

func (u *aeadSafe) Seal(_ context.Context, sc SecretContext, clear []byte) ([]byte, error) {
	return u.primitive.Encrypt(clear, AssociatedData(u.id, sc))
}

func (u *aeadSafe) Open(_ context.Context, sc SecretContext, payload []byte) ([]byte, error) {
	return u.primitive.Decrypt(payload, AssociatedData(u.id, sc))
}
