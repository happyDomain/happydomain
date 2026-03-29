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

// Internal package tests — allows direct access to the secretKey var.
package secret

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"testing"

	"git.happydns.org/happyDomain/model"
)

var (
	ctx        = context.Background()
	testUserID = happydns.Identifier([]byte("user-test"))
	testPrvdID = happydns.Identifier([]byte("provider-test"))
	plainData  = json.RawMessage(`{"server":"127.0.0.1","key":"secret"}`)
)

// validKey32 is a 32-byte (64-hex-char) key used across tests.
var validKey32 = bytes.Repeat([]byte{0xAB}, 32)
var validKeyHex = hex.EncodeToString(validKey32)

// --- PlaintextBackend ---

func TestPlaintextBackend_Method(t *testing.T) {
	b := &PlaintextBackend{}
	if b.Method() != "plaintext" {
		t.Errorf("expected method 'plaintext', got %q", b.Method())
	}
}

func TestPlaintextBackend_SealOpen(t *testing.T) {
	b := &PlaintextBackend{}

	env, err := b.Seal(ctx, testUserID, testPrvdID, plainData)
	if err != nil {
		t.Fatalf("Seal error: %v", err)
	}
	if env.Version != 1 {
		t.Errorf("expected Version=1, got %d", env.Version)
	}
	if env.Method != "plaintext" {
		t.Errorf("expected Method=plaintext, got %q", env.Method)
	}
	if !bytes.Equal(env.Ciphertext, plainData) {
		t.Errorf("Ciphertext mismatch: got %s", env.Ciphertext)
	}

	got, err := b.Open(ctx, env)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	if !bytes.Equal(got, plainData) {
		t.Errorf("expected %s, got %s", plainData, got)
	}
}

func TestPlaintextBackend_Delete(t *testing.T) {
	b := &PlaintextBackend{}
	env := &happydns.SecretEnvelope{Version: 1, Method: "plaintext", Ciphertext: plainData}
	if err := b.Delete(ctx, env); err != nil {
		t.Errorf("Delete should be a no-op, got error: %v", err)
	}
}

// --- InstanceKeyBackend ---

func withSecretKey(t *testing.T, keyHex string, fn func()) {
	t.Helper()
	old := secretKey
	secretKey = keyHex
	defer func() { secretKey = old }()
	fn()
}

func TestInstanceKeyBackend_Method(t *testing.T) {
	withSecretKey(t, validKeyHex, func() {
		b, err := NewInstanceKeyBackend()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Method() != "instance-key" {
			t.Errorf("expected method 'instance-key', got %q", b.Method())
		}
	})
}

func TestInstanceKeyBackend_SealOpen(t *testing.T) {
	withSecretKey(t, validKeyHex, func() {
		b, err := NewInstanceKeyBackend()
		if err != nil {
			t.Fatalf("NewInstanceKeyBackend error: %v", err)
		}

		env, err := b.Seal(ctx, testUserID, testPrvdID, plainData)
		if err != nil {
			t.Fatalf("Seal error: %v", err)
		}
		if env.Version != 1 {
			t.Errorf("expected Version=1, got %d", env.Version)
		}
		if env.Method != "instance-key" {
			t.Errorf("expected Method=instance-key, got %q", env.Method)
		}
		if len(env.Nonce) == 0 {
			t.Error("expected non-empty Nonce")
		}
		if bytes.Equal(env.Ciphertext, plainData) {
			t.Error("Ciphertext should not equal plaintext after encryption")
		}

		got, err := b.Open(ctx, env)
		if err != nil {
			t.Fatalf("Open error: %v", err)
		}
		if !bytes.Equal(got, plainData) {
			t.Errorf("round-trip failed: expected %s, got %s", plainData, got)
		}
	})
}

func TestInstanceKeyBackend_EachSealProducesDistinctCiphertext(t *testing.T) {
	withSecretKey(t, validKeyHex, func() {
		b, _ := NewInstanceKeyBackend()
		env1, _ := b.Seal(ctx, testUserID, testPrvdID, plainData)
		env2, _ := b.Seal(ctx, testUserID, testPrvdID, plainData)
		if bytes.Equal(env1.Nonce, env2.Nonce) {
			t.Error("two Seal calls should produce different nonces")
		}
	})
}

func TestInstanceKeyBackend_KeyIDMismatch(t *testing.T) {
	withSecretKey(t, validKeyHex, func() {
		b, _ := NewInstanceKeyBackend()
		env := &happydns.SecretEnvelope{
			Version:    1,
			Method:     "instance-key",
			KeyID:      "deadbeef",
			Nonce:      make([]byte, 12),
			Ciphertext: []byte("garbage"),
		}
		_, err := b.Open(ctx, env)
		if err == nil {
			t.Error("expected key ID mismatch error")
		}
	})
}

func TestInstanceKeyBackend_MissingKey(t *testing.T) {
	withSecretKey(t, "", func() {
		_, err := NewInstanceKeyBackend()
		if err == nil {
			t.Error("expected error when secret-key is empty")
		}
	})
}

func TestInstanceKeyBackend_WrongKeyLength(t *testing.T) {
	withSecretKey(t, hex.EncodeToString([]byte{1, 2, 3}), func() {
		_, err := NewInstanceKeyBackend()
		if err == nil {
			t.Error("expected error for non-32-byte key")
		}
	})
}

func TestInstanceKeyBackend_InvalidHex(t *testing.T) {
	withSecretKey(t, "not-hex!", func() {
		_, err := NewInstanceKeyBackend()
		if err == nil {
			t.Error("expected error for invalid hex key")
		}
	})
}

func TestInstanceKeyBackend_Delete(t *testing.T) {
	withSecretKey(t, validKeyHex, func() {
		b, _ := NewInstanceKeyBackend()
		env := &happydns.SecretEnvelope{Version: 1, Method: "instance-key"}
		if err := b.Delete(ctx, env); err != nil {
			t.Errorf("Delete should be a no-op, got error: %v", err)
		}
	})
}

// --- Manager ---

func TestManager_DefaultMethod(t *testing.T) {
	m := NewManager(&PlaintextBackend{})
	if m.DefaultMethod() != "plaintext" {
		t.Errorf("expected default method 'plaintext', got %q", m.DefaultMethod())
	}
}

func TestManager_Seal_UsesDefaultBackend(t *testing.T) {
	m := NewManager(&PlaintextBackend{})
	env, err := m.Seal(ctx, "", testUserID, testPrvdID, plainData)
	if err != nil {
		t.Fatalf("Seal error: %v", err)
	}
	if env.Method != "plaintext" {
		t.Errorf("expected Method=plaintext, got %q", env.Method)
	}
}

func TestManager_Seal_ExplicitMethod(t *testing.T) {
	withSecretKey(t, validKeyHex, func() {
		ik, _ := NewInstanceKeyBackend()
		m := NewManager(&PlaintextBackend{}, ik)
		env, err := m.Seal(ctx, "instance-key", testUserID, testPrvdID, plainData)
		if err != nil {
			t.Fatalf("Seal error: %v", err)
		}
		if env.Method != "instance-key" {
			t.Errorf("expected Method=instance-key, got %q", env.Method)
		}
	})
}

func TestManager_Seal_UnknownMethod(t *testing.T) {
	m := NewManager(&PlaintextBackend{})
	_, err := m.Seal(ctx, "vault", testUserID, testPrvdID, plainData)
	if err == nil {
		t.Error("expected error for unknown method")
	}
}

func TestManager_Open(t *testing.T) {
	m := NewManager(&PlaintextBackend{})
	env, _ := m.Seal(ctx, "plaintext", testUserID, testPrvdID, plainData)
	got, err := m.Open(ctx, env)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	if !bytes.Equal(got, plainData) {
		t.Errorf("expected %s, got %s", plainData, got)
	}
}

func TestManager_Open_UnknownMethod(t *testing.T) {
	m := NewManager(&PlaintextBackend{})
	env := &happydns.SecretEnvelope{Version: 1, Method: "unknown"}
	_, err := m.Open(ctx, env)
	if err == nil {
		t.Error("expected error for unknown method")
	}
}

func TestManager_Rotate(t *testing.T) {
	withSecretKey(t, validKeyHex, func() {
		ik, _ := NewInstanceKeyBackend()
		// Start with plaintext, rotate to instance-key
		m := NewManager(&PlaintextBackend{}, ik)

		origEnv, _ := m.Seal(ctx, "plaintext", testUserID, testPrvdID, plainData)
		rotated, err := m.Rotate(ctx, "instance-key", testUserID, testPrvdID, origEnv)
		if err != nil {
			t.Fatalf("Rotate error: %v", err)
		}
		if rotated.Method != "instance-key" {
			t.Errorf("expected rotated Method=instance-key, got %q", rotated.Method)
		}

		// The rotated envelope should decrypt back to original plaintext.
		got, err := m.Open(ctx, rotated)
		if err != nil {
			t.Fatalf("Open after rotate error: %v", err)
		}
		if !bytes.Equal(got, plainData) {
			t.Errorf("expected %s after rotate, got %s", plainData, got)
		}
	})
}

// --- TryParseEnvelope ---

func TestTryParseEnvelope_ValidEnvelope(t *testing.T) {
	raw, _ := json.Marshal(happydns.SecretEnvelope{
		Version:    1,
		Method:     "plaintext",
		Ciphertext: plainData,
	})
	env, ok := TryParseEnvelope(raw)
	if !ok {
		t.Fatal("expected TryParseEnvelope to return true for valid envelope")
	}
	if env.Method != "plaintext" {
		t.Errorf("expected Method=plaintext, got %q", env.Method)
	}
}

func TestTryParseEnvelope_LegacyPlaintext(t *testing.T) {
	// Raw JSON credentials without envelope wrapper
	raw := json.RawMessage(`{"server":"127.0.0.1"}`)
	_, ok := TryParseEnvelope(raw)
	if ok {
		t.Error("expected TryParseEnvelope to return false for legacy plaintext")
	}
}

func TestTryParseEnvelope_ZeroVersion(t *testing.T) {
	// v=0 should be treated as legacy
	raw, _ := json.Marshal(map[string]interface{}{
		"v":          0,
		"method":     "plaintext",
		"ciphertext": "{}",
	})
	_, ok := TryParseEnvelope(raw)
	if ok {
		t.Error("expected TryParseEnvelope to return false when version is 0")
	}
}

func TestTryParseEnvelope_MissingMethod(t *testing.T) {
	raw, _ := json.Marshal(map[string]interface{}{
		"v":          1,
		"ciphertext": "{}",
	})
	_, ok := TryParseEnvelope(raw)
	if ok {
		t.Error("expected TryParseEnvelope to return false when method is missing")
	}
}

func TestTryParseEnvelope_InvalidJSON(t *testing.T) {
	_, ok := TryParseEnvelope(json.RawMessage(`not json`))
	if ok {
		t.Error("expected TryParseEnvelope to return false for invalid JSON")
	}
}

// --- NewManagerFromConfig ---

func TestNewManagerFromConfig_Plaintext(t *testing.T) {
	cfg := &happydns.Options{SecretMethod: "plaintext"}
	m, err := NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.DefaultMethod() != "plaintext" {
		t.Errorf("expected default method 'plaintext', got %q", m.DefaultMethod())
	}
}

func TestNewManagerFromConfig_Empty_DefaultsToPlaintext(t *testing.T) {
	cfg := &happydns.Options{}
	m, err := NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.DefaultMethod() != "plaintext" {
		t.Errorf("expected default method 'plaintext', got %q", m.DefaultMethod())
	}
}

func TestNewManagerFromConfig_InstanceKey(t *testing.T) {
	withSecretKey(t, validKeyHex, func() {
		cfg := &happydns.Options{SecretMethod: "instance-key"}
		m, err := NewManagerFromConfig(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.DefaultMethod() != "instance-key" {
			t.Errorf("expected default method 'instance-key', got %q", m.DefaultMethod())
		}
		// Plaintext should also be registered as fallback for legacy data.
		env := &happydns.SecretEnvelope{Version: 1, Method: "plaintext", Ciphertext: plainData}
		got, err := m.Open(ctx, env)
		if err != nil {
			t.Fatalf("expected plaintext backend to be registered as fallback, got error: %v", err)
		}
		if !bytes.Equal(got, plainData) {
			t.Errorf("expected %s, got %s", plainData, got)
		}
	})
}

func TestNewManagerFromConfig_UnknownMethod(t *testing.T) {
	cfg := &happydns.Options{SecretMethod: "magic"}
	_, err := NewManagerFromConfig(cfg)
	if err == nil {
		t.Error("expected error for unknown secret method")
	}
}
