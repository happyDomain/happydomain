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
	"fmt"
)

// ParsePolicy returns the policy named s, the plaintext one when s is empty.
// Any other value is an error: a mistyped policy must stop the server before
// anything is written to the database, not fall back to some default.
func ParsePolicy(s string) (Policy, error) {
	switch p := Policy(s); p {
	case "":
		return PolicyPlaintext, nil
	case PolicyPlaintext, PolicyInstance:
		return p, nil
	default:
		return "", fmt.Errorf("unknown secret policy %q: expected %q or %q", s, PolicyPlaintext, PolicyInstance)
	}
}
