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

package helpers

import (
	"crypto/sha256"
	"fmt"
)

// EmailIdentifier computes the owner name prefix used by OPENPGPKEY (RFC 7929)
// and SMIMEA (RFC 8162) records: the SHA-256 of the local-part, truncated to
// its 28 leftmost bytes, in hexadecimal.
func EmailIdentifier(username string) string {
	hash := sha256.Sum256([]byte(username))
	return fmt.Sprintf("%x", hash[:28])
}
