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
	"context"
	"encoding/json"
)

// SecretEnvelope wraps encrypted or externally-stored secret data with metadata
// describing how it is protected. When stored in the database, the Provider
// field of ProviderMessage contains a JSON-serialized SecretEnvelope instead of
// raw credentials.
type SecretEnvelope struct {
	// Version allows future format evolution.
	Version int `json:"v"`

	// Method identifies which SecretBackend produced this envelope.
	Method string `json:"method"`

	// KeyID identifies the specific key or key version used.
	// For instance-key: a key fingerprint.
	// For vault/hsm/api: the secret path or reference.
	KeyID string `json:"key_id,omitempty"`

	// Nonce for symmetric encryption schemes.
	Nonce []byte `json:"nonce,omitempty"`

	// Ciphertext holds the encrypted secret data, or for external backends,
	// holds a JSON reference/pointer.
	Ciphertext json.RawMessage `json:"ciphertext"`
}

// SecretBackend encrypts/decrypts or stores/retrieves secret data.
// Each implementation handles one "method" string.
type SecretBackend interface {
	// Method returns the identifier string for this backend.
	Method() string

	// Seal takes plaintext secret JSON and returns a SecretEnvelope.
	// For external backends (Vault, HSM), this stores the secret externally
	// and returns an envelope containing a reference.
	Seal(ctx context.Context, userID, providerID Identifier, plaintext json.RawMessage) (*SecretEnvelope, error)

	// Open takes a SecretEnvelope and returns the plaintext JSON.
	// For external backends, this retrieves the secret from the external
	// store using the reference in the envelope.
	Open(ctx context.Context, envelope *SecretEnvelope) (json.RawMessage, error)

	// Delete removes any externally-stored secret associated with the given
	// envelope. No-op for encryption-only backends.
	Delete(ctx context.Context, envelope *SecretEnvelope) error
}
