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
	"encoding/json"

	"git.happydns.org/happyDomain/model"
)

// TryParseEnvelope attempts to parse raw JSON as a SecretEnvelope.
// Returns the envelope and true if it looks like an envelope (has Version > 0),
// or nil and false if it's legacy plaintext.
func TryParseEnvelope(raw json.RawMessage) (*happydns.SecretEnvelope, bool) {
	var envelope happydns.SecretEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, false
	}
	if envelope.Version > 0 && envelope.Method != "" {
		return &envelope, true
	}
	return nil, false
}

// NewManagerFromConfig creates a Manager based on the given Options.
// It always registers the plaintext backend. If SecretMethod is "instance-key",
// it also creates and registers the InstanceKeyBackend.
func NewManagerFromConfig(cfg *happydns.Options) (*Manager, error) {
	plaintext := &PlaintextBackend{}

	method := cfg.SecretMethod
	if method == "" {
		method = "plaintext"
	}

	switch method {
	case "plaintext":
		return NewManager(plaintext), nil
	case "instance-key":
		ik, err := NewInstanceKeyBackend()
		if err != nil {
			return nil, err
		}
		return NewManager(ik, plaintext), nil
	default:
		return nil, &happydns.ValidationError{Msg: "unknown secret method: " + method}
	}
}
