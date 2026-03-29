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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"git.happydns.org/happyDomain/model"
)

var (
	secretKey string
)

func init() {
	flag.StringVar(&secretKey, "secret-key", "", "Hex-encoded 32-byte key for encrypting provider secrets (AES-256-GCM)")
}

// InstanceKeyBackend encrypts secrets using AES-256-GCM with an
// instance-wide secret key configured via the -secret-key flag.
type InstanceKeyBackend struct {
	key   []byte
	keyID string
}

// NewInstanceKeyBackend creates an InstanceKeyBackend from the flag-configured
// secret key. Returns an error if the key is not set or has an invalid length.
func NewInstanceKeyBackend() (*InstanceKeyBackend, error) {
	if secretKey == "" {
		return nil, fmt.Errorf("secret-key flag is required for instance-key secret method")
	}

	key, err := hex.DecodeString(secretKey)
	if err != nil {
		return nil, fmt.Errorf("secret-key must be hex-encoded: %w", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("secret-key must be exactly 32 bytes (64 hex chars) for AES-256, got %d bytes", len(key))
	}

	// KeyID is the first 8 hex chars of SHA-256(key) for identification.
	h := sha256.Sum256(key)
	keyID := hex.EncodeToString(h[:4])

	return &InstanceKeyBackend{
		key:   key,
		keyID: keyID,
	}, nil
}

func (b *InstanceKeyBackend) Method() string {
	return "instance-key"
}

func (b *InstanceKeyBackend) Seal(_ context.Context, _, _ happydns.Identifier, plaintext json.RawMessage) (*happydns.SecretEnvelope, error) {
	block, err := aes.NewCipher(b.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertextBytes := gcm.Seal(nil, nonce, plaintext, nil)

	ciphertextJSON, err := json.Marshal(ciphertextBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to encode ciphertext: %w", err)
	}

	return &happydns.SecretEnvelope{
		Version:    1,
		Method:     "instance-key",
		KeyID:      b.keyID,
		Nonce:      nonce,
		Ciphertext: ciphertextJSON,
	}, nil
}

func (b *InstanceKeyBackend) Open(_ context.Context, envelope *happydns.SecretEnvelope) (json.RawMessage, error) {
	if envelope.KeyID != "" && envelope.KeyID != b.keyID {
		return nil, fmt.Errorf("key ID mismatch: envelope has %q, current key is %q", envelope.KeyID, b.keyID)
	}

	block, err := aes.NewCipher(b.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	var ciphertextBytes []byte
	if err := json.Unmarshal(envelope.Ciphertext, &ciphertextBytes); err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	plaintext, err := gcm.Open(nil, envelope.Nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func (b *InstanceKeyBackend) Delete(_ context.Context, _ *happydns.SecretEnvelope) error {
	return nil
}
