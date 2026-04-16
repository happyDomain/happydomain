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

package session_test

import (
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/session"
	sessionUC "git.happydns.org/happyDomain/internal/usecase/session"
)

func Test_IsValidSessionID_RoundTrip(t *testing.T) {
	// A freshly-generated session ID must always be considered valid.
	for range 32 {
		id := sessionUC.NewSessionID()
		if !session.IsValidSessionID(id) {
			t.Fatalf("NewSessionID() produced %q which IsValidSessionID rejected", id)
		}
	}
}

func Test_IsValidSessionID_Rejects(t *testing.T) {
	valid := sessionUC.NewSessionID()

	cases := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"one char short", valid[:len(valid)-1]},
		{"one char long", valid + "A"},
		{"all lowercase", strings.ToLower(valid)},
		{"with base32 padding", strings.Repeat("A", sessionUC.SessionIDLen-1) + "="},
		{"digit 0 (not in base32 alphabet)", strings.Repeat("A", sessionUC.SessionIDLen-1) + "0"},
		{"digit 1 (not in base32 alphabet)", strings.Repeat("A", sessionUC.SessionIDLen-1) + "1"},
		{"digit 8 (not in base32 alphabet)", strings.Repeat("A", sessionUC.SessionIDLen-1) + "8"},
		{"digit 9 (not in base32 alphabet)", strings.Repeat("A", sessionUC.SessionIDLen-1) + "9"},
		{"embedded space", strings.Repeat("A", sessionUC.SessionIDLen-1) + " "},
		{"non-ASCII", strings.Repeat("A", sessionUC.SessionIDLen-1) + "é"},
		{"looks like a JWT", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0In0.sig"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if session.IsValidSessionID(tc.in) {
				t.Errorf("IsValidSessionID(%q) = true, want false", tc.in)
			}
		})
	}
}
