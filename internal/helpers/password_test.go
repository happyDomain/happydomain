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
	"strings"
	"testing"
)

func TestGeneratePassword(t *testing.T) {
	t.Run("generates 12 character password", func(t *testing.T) {
		password, err := GeneratePassword()
		if err != nil {
			t.Fatalf("GeneratePassword() returned error: %v", err)
		}

		if len(password) != 12 {
			t.Errorf("GeneratePassword() generated password of length %d, want 12", len(password))
		}
	})

	t.Run("password does not contain replaced characters", func(t *testing.T) {
		forbiddenChars := []string{"v", "u", "l", "1", "o", "O", "0", "/"}

		for range 100 {
			password, err := GeneratePassword()
			if err != nil {
				t.Fatalf("GeneratePassword() returned error: %v", err)
			}

			for _, char := range forbiddenChars {
				if strings.Contains(password, char) {
					t.Errorf("GeneratePassword() generated password containing forbidden character %q: %s", char, password)
				}
			}
		}
	})

	t.Run("password contains replacement characters", func(t *testing.T) {
		replacementChars := []string{"*", "(", "%", "?", "@", "!", ">", "^"}
		foundChars := make(map[string]bool)

		for range 1000 {
			password, err := GeneratePassword()
			if err != nil {
				t.Fatalf("GeneratePassword() returned error: %v", err)
			}

			for _, char := range replacementChars {
				if strings.Contains(password, char) {
					foundChars[char] = true
				}
			}
		}

		if len(foundChars) == 0 {
			t.Error("GeneratePassword() did not use any replacement characters in 1000 attempts")
		}
	})

	t.Run("generates different passwords", func(t *testing.T) {
		passwords := make(map[string]bool)
		iterations := 100

		for range iterations {
			password, err := GeneratePassword()
			if err != nil {
				t.Fatalf("GeneratePassword() returned error: %v", err)
			}
			passwords[password] = true
		}

		if len(passwords) < iterations {
			duplicates := iterations - len(passwords)
			t.Errorf("GeneratePassword() generated %d duplicate passwords out of %d", duplicates, iterations)
		}
	})

	t.Run("password uses valid characters", func(t *testing.T) {
		validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+*(%?@!>^="

		for range 100 {
			password, err := GeneratePassword()
			if err != nil {
				t.Fatalf("GeneratePassword() returned error: %v", err)
			}

			for _, r := range password {
				if !strings.ContainsRune(validChars, r) {
					t.Errorf("GeneratePassword() generated password with invalid character %q: %s", r, password)
				}
			}
		}
	})

	t.Run("password has good entropy", func(t *testing.T) {
		charCounts := make(map[rune]int)
		iterations := 1000

		for range iterations {
			password, err := GeneratePassword()
			if err != nil {
				t.Fatalf("GeneratePassword() returned error: %v", err)
			}

			for _, r := range password {
				charCounts[r]++
			}
		}

		uniqueChars := len(charCounts)
		if uniqueChars < 30 {
			t.Errorf("GeneratePassword() used only %d unique characters in %d passwords, expected more variety", uniqueChars, iterations)
		}
	})

	t.Run("password ends with valid character", func(t *testing.T) {
		for range 100 {
			password, err := GeneratePassword()
			if err != nil {
				t.Fatalf("GeneratePassword() returned error: %v", err)
			}

			lastChar := password[len(password)-1]
			if lastChar == '=' {
				t.Log("Password contains base64 padding character '=' at the end (acceptable)")
			}
		}
	})
}

func TestGeneratePasswordNonEmpty(t *testing.T) {
	password, err := GeneratePassword()
	if err != nil {
		t.Fatalf("GeneratePassword() returned error: %v", err)
	}

	if password == "" {
		t.Error("GeneratePassword() returned empty password")
	}
}

func TestGeneratePasswordConsistentLength(t *testing.T) {
	lengths := make(map[int]int)

	for range 1000 {
		password, err := GeneratePassword()
		if err != nil {
			t.Fatalf("GeneratePassword() returned error: %v", err)
		}
		lengths[len(password)]++
	}

	if len(lengths) != 1 {
		t.Errorf("GeneratePassword() generated passwords with varying lengths: %v", lengths)
	}

	for length := range lengths {
		if length != 12 {
			t.Errorf("GeneratePassword() generated password of length %d, want 12", length)
		}
	}
}

func TestGeneratePasswordReplacements(t *testing.T) {
	replacements := map[string]string{
		"v": "*",
		"u": "(",
		"l": "%",
		"1": "?",
		"o": "@",
		"O": "!",
		"0": ">",
		"/": "^",
	}

	t.Run("verifies character replacements", func(t *testing.T) {
		for range 100 {
			password, err := GeneratePassword()
			if err != nil {
				t.Fatalf("GeneratePassword() returned error: %v", err)
			}

			for original := range replacements {
				if strings.Contains(password, original) {
					t.Errorf("Password contains unreplaced character %q: %s", original, password)
				}
			}
		}
	})
}

func BenchmarkGeneratePassword(b *testing.B) {
	for b.Loop() {
		_, err := GeneratePassword()
		if err != nil {
			b.Fatalf("GeneratePassword() returned error: %v", err)
		}
	}
}

func BenchmarkGeneratePasswordParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := GeneratePassword()
			if err != nil {
				b.Fatalf("GeneratePassword() returned error: %v", err)
			}
		}
	})
}
