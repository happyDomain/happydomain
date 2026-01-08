// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package happydns_test

import (
	"encoding/json"
	"testing"

	"git.happydns.org/happyDomain/model"
)

func TestNewRandomIdentifier(t *testing.T) {
	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatalf("NewRandomIdentifier() error = %v", err)
	}

	if len(id) != happydns.IDENTIFIER_LEN {
		t.Errorf("NewRandomIdentifier() length = %d; want %d", len(id), happydns.IDENTIFIER_LEN)
	}
}

func TestNewRandomIdentifierUniqueness(t *testing.T) {
	id1, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatalf("NewRandomIdentifier() error = %v", err)
	}

	id2, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatalf("NewRandomIdentifier() error = %v", err)
	}

	if id1.Equals(id2) {
		t.Error("NewRandomIdentifier() generated identical identifiers, expected unique values")
	}
}

func TestNewIdentifierFromString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "valid base64url",
			input:       "dGVzdGluZzEyMzQ1Ng",
			expectError: false,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: false,
		},
		{
			name:        "valid short string",
			input:       "YWJj",
			expectError: false,
		},
		{
			name:        "valid long string",
			input:       "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo",
			expectError: false,
		},
		{
			name:        "invalid character",
			input:       "invalid@character",
			expectError: true,
		},
		{
			name:        "padding not allowed in raw encoding",
			input:       "dGVzdA==",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := happydns.NewIdentifierFromString(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewIdentifierFromString() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewIdentifierFromString() error = %v", err)
			}

			if id == nil {
				t.Error("NewIdentifierFromString() returned nil identifier")
			}
		})
	}
}

func TestIdentifierIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		id       happydns.Identifier
		expected bool
	}{
		{
			name:     "empty identifier",
			id:       happydns.Identifier{},
			expected: true,
		},
		{
			name:     "nil identifier",
			id:       nil,
			expected: true,
		},
		{
			name:     "non-empty identifier",
			id:       happydns.Identifier{0x01, 0x02, 0x03},
			expected: false,
		},
		{
			name:     "single byte identifier",
			id:       happydns.Identifier{0x00},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.id.IsEmpty()
			if result != tt.expected {
				t.Errorf("IsEmpty() = %v; want %v", result, tt.expected)
			}
		})
	}
}

func TestIdentifierEquals(t *testing.T) {
	tests := []struct {
		name     string
		id1      happydns.Identifier
		id2      happydns.Identifier
		expected bool
	}{
		{
			name:     "equal identifiers",
			id1:      happydns.Identifier{0x01, 0x02, 0x03},
			id2:      happydns.Identifier{0x01, 0x02, 0x03},
			expected: true,
		},
		{
			name:     "different identifiers",
			id1:      happydns.Identifier{0x01, 0x02, 0x03},
			id2:      happydns.Identifier{0x04, 0x05, 0x06},
			expected: false,
		},
		{
			name:     "empty identifiers",
			id1:      happydns.Identifier{},
			id2:      happydns.Identifier{},
			expected: true,
		},
		{
			name:     "nil identifiers",
			id1:      nil,
			id2:      nil,
			expected: true,
		},
		{
			name:     "different lengths",
			id1:      happydns.Identifier{0x01, 0x02},
			id2:      happydns.Identifier{0x01, 0x02, 0x03},
			expected: false,
		},
		{
			name:     "one empty one not",
			id1:      happydns.Identifier{},
			id2:      happydns.Identifier{0x01},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.id1.Equals(tt.id2)
			if result != tt.expected {
				t.Errorf("Equals() = %v; want %v", result, tt.expected)
			}
		})
	}
}

func TestIdentifierString(t *testing.T) {
	tests := []struct {
		name     string
		id       happydns.Identifier
		expected string
	}{
		{
			name:     "empty identifier",
			id:       happydns.Identifier{},
			expected: "",
		},
		{
			name:     "single byte",
			id:       happydns.Identifier{0x42},
			expected: "Qg",
		},
		{
			name:     "multiple bytes",
			id:       happydns.Identifier{0xde, 0xad, 0xbe, 0xef},
			expected: "3q2-7w",
		},
		{
			name:     "standard identifier length",
			id:       happydns.Identifier{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10},
			expected: "ASNFZ4mrze_-3LqYdlQyEA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.id.String()
			if result != tt.expected {
				t.Errorf("String() = %q; want %q", result, tt.expected)
			}
		})
	}
}

func TestIdentifierMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		id       happydns.Identifier
		expected string
	}{
		{
			name:     "empty identifier",
			id:       happydns.Identifier{},
			expected: `""`,
		},
		{
			name:     "single byte",
			id:       happydns.Identifier{0x42},
			expected: `"Qg"`,
		},
		{
			name:     "multiple bytes",
			id:       happydns.Identifier{0xde, 0xad, 0xbe, 0xef},
			expected: `"3q2-7w"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.id.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("MarshalJSON() = %s; want %s", string(result), tt.expected)
			}
		})
	}
}

func TestIdentifierUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    happydns.Identifier
		expectError bool
	}{
		{
			name:        "empty string",
			input:       `""`,
			expected:    happydns.Identifier{},
			expectError: false,
		},
		{
			name:        "single byte",
			input:       `"Qg"`,
			expected:    happydns.Identifier{0x42},
			expectError: false,
		},
		{
			name:        "multiple bytes",
			input:       `"3q2-7w"`,
			expected:    happydns.Identifier{0xde, 0xad, 0xbe, 0xef},
			expectError: false,
		},
		{
			name:        "missing opening quote",
			input:       `3q2-7w"`,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "missing closing quote",
			input:       `"3q2-7w`,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "no quotes",
			input:       `3q2-7w`,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "invalid base64url character",
			input:       `"invalid@char"`,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "empty input",
			input:       ``,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "only quotes",
			input:       `"`,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id happydns.Identifier
			err := id.UnmarshalJSON([]byte(tt.input))

			if tt.expectError {
				if err == nil {
					t.Errorf("UnmarshalJSON() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}

			if !id.Equals(tt.expected) {
				t.Errorf("UnmarshalJSON() = %v; want %v", id, tt.expected)
			}
		})
	}
}

func TestIdentifierRoundTrip(t *testing.T) {
	tests := []happydns.Identifier{
		{},
		{0x00},
		{0xff},
		{0xde, 0xad, 0xbe, 0xef},
		{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10},
	}

	for _, original := range tests {
		t.Run("", func(t *testing.T) {
			marshaled, err := original.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}

			var unmarshaled happydns.Identifier
			err = unmarshaled.UnmarshalJSON(marshaled)
			if err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}

			if !original.Equals(unmarshaled) {
				t.Errorf("Round trip mismatch: got %v, want %v", unmarshaled, original)
			}
		})
	}
}

func TestIdentifierStringRoundTrip(t *testing.T) {
	original := happydns.Identifier{0xde, 0xad, 0xbe, 0xef, 0xca, 0xfe, 0xba, 0xbe}

	str := original.String()
	recovered, err := happydns.NewIdentifierFromString(str)
	if err != nil {
		t.Fatalf("NewIdentifierFromString() error = %v", err)
	}

	if !original.Equals(recovered) {
		t.Errorf("String round trip mismatch: got %v, want %v", recovered, original)
	}
}

func TestIdentifierJSONIntegration(t *testing.T) {
	type testStruct struct {
		ID happydns.Identifier `json:"id"`
	}

	original := testStruct{
		ID: happydns.Identifier{0xca, 0xfe, 0xba, 0xbe},
	}

	marshaled, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled testStruct
	err = json.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !original.ID.Equals(unmarshaled.ID) {
		t.Errorf("JSON integration mismatch: got %v, want %v", unmarshaled.ID, original.ID)
	}
}
