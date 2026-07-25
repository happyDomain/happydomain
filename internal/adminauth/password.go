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

// Package adminauth provides the password verification and session-token
// primitives that gate access to the happyDomain admin interface.
package adminauth

import (
	"crypto/subtle"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// IsHashed reports whether the configured admin secret is a bcrypt hash rather
// than a cleartext password. bcrypt hashes start with the "$2" prefix, but a
// cleartext password is allowed to start with "$2" as well, so the value is only
// treated as a hash when bcrypt itself can parse it. Testing the prefix alone
// would route such a password to bcrypt, where every comparison fails, locking
// the operator out with no way to tell why.
func IsHashed(configured string) bool {
	_, err := bcrypt.Cost([]byte(configured))
	return err == nil
}

// IsMalformedHash reports whether the configured admin secret carries the bcrypt
// "$2" prefix but cannot be parsed as a hash. Such a value is used as a cleartext
// password, which is most likely not what the operator meant: it usually is a
// truncated or mangled hash. Callers should report it at startup.
func IsMalformedHash(configured string) bool {
	return strings.HasPrefix(configured, "$2") && !IsHashed(configured)
}

// VerifyAdminPassword compares provided against the configured admin secret.
// The configured value is either a bcrypt hash (verified with bcrypt, which is
// constant-time and enforces the 72-byte input limit) or a cleartext password
// (compared in constant time). It returns false when no secret is configured.
func VerifyAdminPassword(configured, provided string) bool {
	if configured == "" {
		return false
	}

	if IsHashed(configured) {
		return bcrypt.CompareHashAndPassword([]byte(configured), []byte(provided)) == nil
	}

	return subtle.ConstantTimeCompare([]byte(configured), []byte(provided)) == 1
}
