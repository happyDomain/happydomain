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

package controller

import "testing"

func TestIsUnderDomain(t *testing.T) {
	tests := []struct {
		host     string
		domain   string
		expected bool
	}{
		{"example.com", "example.com.", true},
		{"example.com.", "example.com.", true},
		{"www.example.com", "example.com.", true},
		{"deep.sub.example.com.", "example.com.", true},
		{"WWW.Example.COM", "example.com.", true},
		{"www.example.com", "Example.com", true},

		// A raw byte suffix test would wrongly accept these.
		{"notexample.com", "example.com.", false},
		{"notexample.com.", "example.com.", false},
		{"wwwexample.com.", "www.example.com.", false},

		{"example.com.evil.net.", "example.com.", false},
		{"com.", "example.com.", false},
		{"other.net.", "example.com.", false},

		// The root is never a domain a user can own.
		{"example.com.", ".", false},
	}

	for _, tt := range tests {
		if got := isUnderDomain(tt.host, tt.domain); got != tt.expected {
			t.Errorf("isUnderDomain(%q, %q) = %v; want %v", tt.host, tt.domain, got, tt.expected)
		}
	}
}
