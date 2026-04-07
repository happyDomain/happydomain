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

package domaininfo

import "testing"

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNil  bool
		wantVal  string
	}{
		{"https URL", "https://example.com", false, "https://example.com"},
		{"http URL", "http://example.com", false, "http://example.com"},
		{"https with path", "https://registrar.example.com/panel", false, "https://registrar.example.com/panel"},
		{"javascript scheme", "javascript:alert(1)", true, ""},
		{"data scheme", "data:text/html,<h1>hi</h1>", true, ""},
		{"ftp scheme", "ftp://example.com", true, ""},
		{"empty string", "", true, ""},
		{"no scheme", "example.com", true, ""},
		{"scheme-relative", "//example.com", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeURL(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("sanitizeURL(%q) = %q, want nil", tt.input, *got)
				}
			} else {
				if got == nil {
					t.Fatalf("sanitizeURL(%q) = nil, want %q", tt.input, tt.wantVal)
				}
				if *got != tt.wantVal {
					t.Errorf("sanitizeURL(%q) = %q, want %q", tt.input, *got, tt.wantVal)
				}
			}
		})
	}
}
