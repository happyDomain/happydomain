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

package happydns

import (
	"encoding/json"
	"time"
)

// Safe protects the secrets of a user: it holds the key they are sealed
// with, wrapped by one or more key encryption keys. See
// docs/secret-management.md.
type Safe struct {
	Id    Identifier `json:"id"`
	Owner Identifier `json:"owner"`

	// Kind tells what protects the safe: "instance" for now.
	Kind string `json:"kind"`

	// Keyring holds the key of the safe, a Tink keyset, once per key
	// encryption key able to unwrap it.
	Keyring []WrappedKeyset `json:"keyring"`

	CreatedAt time.Time `json:"createdAt"`
}

// WrappedKeyset is the key of a safe, encrypted by one key encryption key.
type WrappedKeyset struct {
	// KEK names the key encryption key: "instance" for now.
	KEK string `json:"kek"`

	// Params holds what the key encryption key needs besides itself.
	Params json.RawMessage `json:"params,omitempty"`

	// Blob is the encrypted keyset.
	Blob []byte `json:"blob"`
}
