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

package middleware

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

func AuthUserHandlerBase(auService happydns.AuthUserUsecase, c *gin.Context) (*happydns.UserAuth, error) {
	uid, err := base64.RawURLEncoding.DecodeString(c.Param("uid"))
	if err != nil {
		return nil, fmt.Errorf("Invalid user identifier given: %w", err)
	}

	user, err := auService.GetAuthUser(uid)
	if err != nil {
		return nil, fmt.Errorf("User not found")
	}

	return user, nil
}

func AuthUserHandler(auService happydns.AuthUserUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := AuthUserHandlerBase(auService, c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: err.Error()})
			return
		}

		c.Set("authuser", user)

		c.Next()
	}
}
