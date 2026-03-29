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

// PlaintextBackend is a no-op backend that wraps secrets in an envelope
// without encryption. Used for backward compatibility and as a default.
type PlaintextBackend struct{}

func (b *PlaintextBackend) Method() string {
	return "plaintext"
}

func (b *PlaintextBackend) Seal(_ context.Context, _, _ happydns.Identifier, plaintext json.RawMessage) (*happydns.SecretEnvelope, error) {
	return &happydns.SecretEnvelope{
		Version:    1,
		Method:     "plaintext",
		Ciphertext: plaintext,
	}, nil
}

func (b *PlaintextBackend) Open(_ context.Context, envelope *happydns.SecretEnvelope) (json.RawMessage, error) {
	return envelope.Ciphertext, nil
}

func (b *PlaintextBackend) Delete(_ context.Context, _ *happydns.SecretEnvelope) error {
	return nil
}
