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
	"fmt"

	"git.happydns.org/happyDomain/model"
)

// Manager coordinates secret operations across multiple backends.
// It uses the "default" backend for new Seal operations when no specific
// method is requested, and can Open envelopes from any registered backend.
type Manager struct {
	defaultMethod string
	backends      map[string]happydns.SecretBackend
}

// NewManager creates a Manager with the given default backend and additional
// backends for opening legacy envelopes.
func NewManager(defaultBackend happydns.SecretBackend, others ...happydns.SecretBackend) *Manager {
	m := &Manager{
		defaultMethod: defaultBackend.Method(),
		backends:      make(map[string]happydns.SecretBackend),
	}
	m.backends[defaultBackend.Method()] = defaultBackend
	for _, b := range others {
		m.backends[b.Method()] = b
	}
	return m
}

// Seal encrypts or stores the plaintext using the specified method. If method
// is empty, the default backend is used.
func (m *Manager) Seal(ctx context.Context, method string, userID, providerID happydns.Identifier, plaintext json.RawMessage) (*happydns.SecretEnvelope, error) {
	if method == "" {
		method = m.defaultMethod
	}
	b, ok := m.backends[method]
	if !ok {
		return nil, fmt.Errorf("unknown secret method %q", method)
	}
	return b.Seal(ctx, userID, providerID, plaintext)
}

// Open decrypts or retrieves the plaintext from the given envelope.
// The backend is selected based on the envelope's Method field.
func (m *Manager) Open(ctx context.Context, envelope *happydns.SecretEnvelope) (json.RawMessage, error) {
	b, ok := m.backends[envelope.Method]
	if !ok {
		return nil, fmt.Errorf("unknown secret method %q", envelope.Method)
	}
	return b.Open(ctx, envelope)
}

// Delete cleans up any externally-stored secret for the given envelope.
func (m *Manager) Delete(ctx context.Context, envelope *happydns.SecretEnvelope) error {
	b, ok := m.backends[envelope.Method]
	if !ok {
		return nil
	}
	return b.Delete(ctx, envelope)
}

// Rotate re-encrypts an envelope: opens with the old backend, seals with the
// target method.
func (m *Manager) Rotate(ctx context.Context, targetMethod string, userID, providerID happydns.Identifier, envelope *happydns.SecretEnvelope) (*happydns.SecretEnvelope, error) {
	plaintext, err := m.Open(ctx, envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to open during rotation: %w", err)
	}
	newEnvelope, err := m.Seal(ctx, targetMethod, userID, providerID, plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to seal during rotation: %w", err)
	}
	_ = m.Delete(ctx, envelope)
	return newEnvelope, nil
}

// DefaultMethod returns the default secret method name.
func (m *Manager) DefaultMethod() string {
	return m.defaultMethod
}
