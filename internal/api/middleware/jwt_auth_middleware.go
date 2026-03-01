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
	"log"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"git.happydns.org/happyDomain/model"
)

// UserClaims is an object that permit user creation after authentication.
type UserClaims struct {
	Profile happydns.UserProfile `json:"profile"`
	jwt.RegisteredClaims
}

func JwtAuthMiddleware(authService happydns.AuthenticationUsecase, signingMethod string, secretKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := c.Request.Header["Authorization"]
		if !ok || len(c.Request.Header["Authorization"]) == 0 {
			c.Next()
			return
		}

		flds := strings.Fields(c.Request.Header["Authorization"][0])
		if len(flds) != 2 || strings.ToLower(flds[0]) != "bearer" {
			c.Next()
			return
		}

		token := flds[1]

		if len(token) == 0 {
			log.Printf("%s Skip %s authorization due to malformed token", c.ClientIP(), flds[0])
			c.Next()
			return
		}

		// Validate the token and retrieve claims
		claims := &UserClaims{}
		_, err := jwt.ParseWithClaims(token, claims,
			func(token *jwt.Token) (any, error) {
				return secretKey, nil
			}, jwt.WithValidMethods([]string{signingMethod}))
		if err != nil {
			log.Printf("%s provide a bad JWT claims: %s", c.ClientIP(), err.Error())
			return
		}

		// Check that required fields are filled
		if claims == nil || len(claims.Profile.UserId) == 0 {
			log.Printf("%s: no UserId found in JWT claims", c.ClientIP())
			return
		}

		if claims.Profile.GetEmail() == "" {
			log.Printf("%s: no Email found in JWT claims", c.ClientIP())
			return
		}

		// Retrieve corresponding user
		user, err := authService.CompleteAuthentication(&claims.Profile)
		if err != nil {
			log.Printf("%s: unable to complete JWT authentication: %s", c.ClientIP(), err.Error())
			return
		}

		session := sessions.Default(c)

		session.Clear()
		session.Set("iduser", user.Id)
		err = session.Save()
		if err != nil {
			log.Printf("%s: unable to save user session: %s", c.ClientIP(), err.Error())
			c.JSON(http.StatusUnauthorized, happydns.ErrorResponse{Message: "Invalid username or password."})
			return
		}

		c.Set("AuthMethod", "jwt")
		c.Set("LoggedUser", user)

		// We are now ready to continue
		c.Next()
	}
}
