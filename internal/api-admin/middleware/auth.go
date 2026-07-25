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

// Package middleware holds gin middlewares specific to the admin API.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/adminauth"
	happydns "git.happydns.org/happyDomain/model"
)

// AdminAuth enforces admin-session authentication on the admin API. It is
// opt-in: when no AdminPasswordHash is configured the middleware is a
// passthrough, preserving the historical behavior for the default local unix
// socket. When a password is configured, every request must carry a valid
// admin bearer token (obtained via the admin-login endpoint).
func AdminAuth(cfg *happydns.Options) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.AdminPasswordHash == "" {
			c.Next()
			return
		}

		flds := strings.Fields(c.GetHeader("Authorization"))
		if len(flds) != 2 || strings.ToLower(flds[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, happydns.ErrorResponse{Message: "Admin authentication required."})
			return
		}

		if err := adminauth.VerifyAdminToken(cfg.JWTSecretKey, cfg.AdminPasswordHash, flds[1]); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, happydns.ErrorResponse{Message: "Invalid or expired admin session."})
			return
		}

		c.Next()
	}
}
