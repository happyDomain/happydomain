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

package helpers_test

import (
	"testing"

	"git.happydns.org/happyDomain/internal/helpers"
)

func TestEmailIdentifier(t *testing.T) {
	for _, tt := range []struct {
		username string
		expected string
	}{
		// RFC 7929 sec. 4 example: hugh@example.com
		{"hugh", "c93f1e400f26708f98cb19d936620da35eec8f72e57f9eec01c1afd6"},
		{"user", "04f8996da763b7a969b1028ee3007569eaf3a635486ddab211d512c8"},
		{"", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b"},
	} {
		if got := helpers.EmailIdentifier(tt.username); got != tt.expected {
			t.Errorf("EmailIdentifier(%q) = %q, expected %q", tt.username, got, tt.expected)
		}
	}
}

func TestEmailIdentifierLength(t *testing.T) {
	// 28 bytes, hex-encoded: the length OPENPGPKEY and SMIMEA owner names use.
	if got := len(helpers.EmailIdentifier("someone")); got != 56 {
		t.Errorf("EmailIdentifier length = %d, expected 56", got)
	}
}
