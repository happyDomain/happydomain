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

package middleware

import (
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/secret"
)

// KeystoreChoice carries the keystore a request chooses, through its
// keystore query parameter, to the use cases sealing the secrets of the
// object it writes: see secret.WithKeystore.
func KeystoreChoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		if ref := c.Query("keystore"); ref != "" {
			c.Request = c.Request.WithContext(secret.WithKeystore(c.Request.Context(), ref))
		}
		c.Next()
	}
}
