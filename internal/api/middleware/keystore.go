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
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
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

// SessionScope returns the scope the keystores unlocked during the session
// of c are kept under: its public identifier, the one sessions are stored
// under, so that tidying closes those of sessions gone.
func SessionScope(c *gin.Context) string {
	id := sessions.Default(c).ID()
	if id == "" {
		return ""
	}
	return happydns.SessionIDFromToken(id)
}

// UnlockScope carries the session of the request to the use cases opening
// secrets: the keystores unlocked during it open, see secret.WithUnlockScope.
func UnlockScope() gin.HandlerFunc {
	return func(c *gin.Context) {
		if scope := SessionScope(c); scope != "" {
			c.Request = c.Request.WithContext(secret.WithUnlockScope(c.Request.Context(), scope))
		}
		c.Next()
	}
}
