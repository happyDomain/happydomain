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

package helpers

import (
	"testing"
)

func TestGenUsername(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "simple email",
			email:    "john@example.com",
			expected: "John",
		},
		{
			name:     "email with dot",
			email:    "john.doe@example.com",
			expected: "John Doe",
		},
		{
			name:     "email with multiple dots",
			email:    "john.peter.doe@example.com",
			expected: "John Peter Doe",
		},
		{
			name:     "email with plus addressing",
			email:    "john+tag@example.com",
			expected: "John",
		},
		{
			name:     "email with plus and dot",
			email:    "john.doe+tag@example.com",
			expected: "John Doe",
		},
		{
			name:     "email with multiple dots and plus",
			email:    "first.middle.last+label@example.com",
			expected: "First Middle Last",
		},
		{
			name:     "single character username",
			email:    "a@example.com",
			expected: "a",
		},
		{
			name:     "two character username",
			email:    "ab@example.com",
			expected: "Ab",
		},
		{
			name:     "username with hyphen",
			email:    "john-doe@example.com",
			expected: "John-Doe",
		},
		{
			name:     "username with underscore",
			email:    "john_doe@example.com",
			expected: "John_Doe",
		},
		{
			name:     "username with mixed separators",
			email:    "john.doe-smith_jr@example.com",
			expected: "John Doe-Smith_Jr",
		},
		{
			name:     "lowercase username",
			email:    "lowercase@example.com",
			expected: "Lowercase",
		},
		{
			name:     "username with numbers",
			email:    "john123@example.com",
			expected: "John123",
		},
		{
			name:     "username with dots and numbers",
			email:    "john.doe123@example.com",
			expected: "John Doe123",
		},
		{
			name:     "username starting with dot",
			email:    ".john@example.com",
			expected: " John",
		},
		{
			name:     "username ending with dot",
			email:    "john.@example.com",
			expected: "John ",
		},
		{
			name:     "username with consecutive dots",
			email:    "john..doe@example.com",
			expected: "John  Doe",
		},
		{
			name:     "all uppercase",
			email:    "JOHN@example.com",
			expected: "JOHN",
		},
		{
			name:     "all uppercase with dots",
			email:    "JOHN.DOE@example.com",
			expected: "JOHN DOE",
		},
		{
			name:     "mixed case preserved except first letter after separator",
			email:    "JoHn.DoE@example.com",
			expected: "JoHn DoE",
		},
		{
			name:     "unicode characters",
			email:    "josé.garcía@example.com",
			expected: "José García",
		},
		{
			name:     "special characters",
			email:    "user!name@example.com",
			expected: "User!Name",
		},
		{
			name:     "plus at start",
			email:    "+tag@example.com",
			expected: "+Tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenUsername(tt.email)
			if result != tt.expected {
				t.Errorf("GenUsername(%q) = %q, want %q", tt.email, result, tt.expected)
			}
		})
	}
}

func TestGenUsernameEdgeCases(t *testing.T) {
	t.Run("panic on missing @ sign", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("GenUsername should panic on email without @ sign")
			}
		}()
		GenUsername("notanemail")
	})

	t.Run("@ at start returns empty string", func(t *testing.T) {
		result := GenUsername("@example.com")
		if result != "" {
			t.Errorf("GenUsername(@example.com) = %q, want empty string", result)
		}
	})
}

func BenchmarkGenUsername(b *testing.B) {
	testCases := []string{
		"john@example.com",
		"john.doe@example.com",
		"first.middle.last@example.com",
		"john+tag@example.com",
		"john.doe+tag@example.com",
	}

	for _, email := range testCases {
		b.Run(email, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				GenUsername(email)
			}
		})
	}
}
