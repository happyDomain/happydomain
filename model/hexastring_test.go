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
	"testing"

	"git.happydns.org/happyDomain/model"
)

func TestHexaStringMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    happydns.HexaString
		expected string
	}{
		{
			name:     "empty byte slice",
			input:    happydns.HexaString{},
			expected: `""`,
		},
		{
			name:     "single byte",
			input:    happydns.HexaString{0x42},
			expected: `"42"`,
		},
		{
			name:     "multiple bytes",
			input:    happydns.HexaString{0xde, 0xad, 0xbe, 0xef},
			expected: `"deadbeef"`,
		},
		{
			name:     "zero bytes",
			input:    happydns.HexaString{0x00, 0x00, 0x00},
			expected: `"000000"`,
		},
		{
			name:     "mixed case result",
			input:    happydns.HexaString{0xff, 0xaa, 0x11},
			expected: `"ffaa11"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.input.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("MarshalJSON() = %s; want %s", string(result), tt.expected)
			}
		})
	}
}

func TestHexaStringUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    happydns.HexaString
		expectError bool
	}{
		{
			name:        "empty string",
			input:       `""`,
			expected:    happydns.HexaString{},
			expectError: false,
		},
		{
			name:        "single byte",
			input:       `"42"`,
			expected:    happydns.HexaString{0x42},
			expectError: false,
		},
		{
			name:        "multiple bytes",
			input:       `"deadbeef"`,
			expected:    happydns.HexaString{0xde, 0xad, 0xbe, 0xef},
			expectError: false,
		},
		{
			name:        "uppercase hex",
			input:       `"DEADBEEF"`,
			expected:    happydns.HexaString{0xde, 0xad, 0xbe, 0xef},
			expectError: false,
		},
		{
			name:        "mixed case hex",
			input:       `"DeAdBeEf"`,
			expected:    happydns.HexaString{0xde, 0xad, 0xbe, 0xef},
			expectError: false,
		},
		{
			name:        "zero bytes",
			input:       `"000000"`,
			expected:    happydns.HexaString{0x00, 0x00, 0x00},
			expectError: false,
		},
		{
			name:        "missing opening quote",
			input:       `deadbeef"`,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "missing closing quote",
			input:       `"deadbeef`,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "no quotes",
			input:       `deadbeef`,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "invalid hex character",
			input:       `"deadbeeg"`,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "odd length hex string",
			input:       `"abc"`,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "empty input",
			input:       ``,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hs happydns.HexaString
			err := hs.UnmarshalJSON([]byte(tt.input))

			if tt.expectError {
				if err == nil {
					t.Errorf("UnmarshalJSON() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}

			if len(hs) != len(tt.expected) {
				t.Errorf("UnmarshalJSON() length = %d; want %d", len(hs), len(tt.expected))
				return
			}

			for i := range hs {
				if hs[i] != tt.expected[i] {
					t.Errorf("UnmarshalJSON() byte[%d] = %x; want %x", i, hs[i], tt.expected[i])
				}
			}
		})
	}
}

func TestHexaStringRoundTrip(t *testing.T) {
	tests := []happydns.HexaString{
		{},
		{0x00},
		{0xff},
		{0xde, 0xad, 0xbe, 0xef},
		{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef},
	}

	for _, original := range tests {
		t.Run("", func(t *testing.T) {
			marshaled, err := original.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}

			var unmarshaled happydns.HexaString
			err = unmarshaled.UnmarshalJSON(marshaled)
			if err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}

			if len(original) != len(unmarshaled) {
				t.Errorf("Round trip length mismatch: got %d, want %d", len(unmarshaled), len(original))
				return
			}

			for i := range original {
				if original[i] != unmarshaled[i] {
					t.Errorf("Round trip byte[%d] mismatch: got %x, want %x", i, unmarshaled[i], original[i])
				}
			}
		})
	}
}

