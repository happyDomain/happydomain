// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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
	"crypto/rand"
	"encoding/base64"
	"strings"
)

// GeneratePassword randomly generates a secure 12 chars long password.
func GeneratePassword() (password string, err error) {
	// This will make a 12 chars long password
	b := make([]byte, 9)

	if _, err = rand.Read(b); err != nil {
		return
	}

	password = base64.StdEncoding.EncodeToString(b)

	// Avoid hard to read characters
	for _, i := range [][2]string{
		{"v", "*"}, {"u", "("},
		{"l", "%"}, {"1", "?"},
		{"o", "@"}, {"O", "!"}, {"0", ">"},
		// This one is to avoid problem with openssl
		{"/", "^"},
	} {
		password = strings.Replace(password, i[0], i[1], -1)
	}

	return
}
