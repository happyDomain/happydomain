// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := c.Get("LoggedUser"); !ok {
			l := "/login"
			c.AbortWithStatusJSON(http.StatusUnauthorized, happydns.ErrorResponse{Message: "Please login to access this resource.", Link: &l})
			return
		}

		c.Next()
	}
}

func SessionLoginOK(c *gin.Context, user happydns.UserInfo) error {
	session := sessions.Default(c)

	session.Clear()
	session.Set("iduser", user.GetUserId())
	err := session.Save()
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("failed to save save user session: %s", err),
			UserMessage: "Invalid username or password.",
		}
	}

	log.Printf("%s: now logged as %q\n", c.ClientIP(), user.GetEmail())
	return nil
}
