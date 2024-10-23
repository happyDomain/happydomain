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

package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/storage"
)

type UserProfile struct {
	UserId        []byte    `json:"userid"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	Newsletter    bool      `json:"wantReceiveUpdate,omitempty"`
}

type UserClaims struct {
	Profile UserProfile `json:"profile"`
	jwt.RegisteredClaims
}

var signingMethod = jwt.SigningMethodHS512

func updateUserFromClaims(user *happydns.User, claims *UserClaims) {
	user.Email = claims.Profile.Email
	user.LastSeen = time.Now()
}

func retrieveUserFromClaims(claims *UserClaims) (user *happydns.User, err error) {
	user, err = storage.MainStore.GetUser(claims.Profile.UserId)
	if err != nil {
		// The user doesn't exists yet: create it!
		user, err = createUserFromProfile(claims.Profile)
		if err != nil {
			err = fmt.Errorf("has a correct JWT, but an error occured when trying to create the user: %w", err)
			return
		}
	} else if time.Since(user.LastSeen) > time.Hour*12 {
		// Update user's data when connected more than 12 hours
		updateUserFromClaims(user, claims)

		err = storage.MainStore.CreateOrUpdateUser(user)
		if err != nil {
			err = fmt.Errorf("has a correct JWT, user has been found, but an error occured when trying to update the user's information: %w", err)
			return
		}
	}

	return
}

func requireLogin(opts *config.Options, c *gin.Context, msg string) {
	if opts.ExternalAuth.URL != nil {
		customurl := *opts.ExternalAuth.URL
		q := customurl.Query()
		q.Set("errmsg", msg)
		customurl.RawQuery = q.Encode()
		c.Header("Location", customurl.String())
	}

	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": msg})
}

func authMiddleware(opts *config.Options, optional bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Load user from session
		session := sessions.Default(c)

		var userid happydns.Identifier
		if iu, ok := session.Get("iduser").([]byte); ok {
			userid = happydns.Identifier(iu)
		}

		var method string
		if _, ok := c.Request.Header["Authorization"]; ok && len(c.Request.Header["Authorization"]) > 0 {
			if flds := strings.Fields(c.Request.Header["Authorization"][0]); len(flds) == 2 {
				method = strings.ToLower(flds[0])
			}
		} else {
			method = "cookie"
		}

		// Authentication through JWT
		var token string
		if c.GetHeader("X-User-Token") != "" {
			token = c.GetHeader("X-User-Token")
		} else if cookie, err := c.Cookie("happydomain-account"); err == nil {
			token = cookie
		}
		if len(opts.JWTSecretKey) > 0 && len(token) > 0 {
			// Validate the token and retrieve claims
			claims := &UserClaims{}
			_, err := jwt.ParseWithClaims(token, claims,
				func(token *jwt.Token) (interface{}, error) {
					return []byte(opts.JWTSecretKey), nil
				}, jwt.WithValidMethods([]string{signingMethod.Name}))
			if err != nil {
				if opts.NoAuth {
					claims = displayNotAuthToken(opts, c)
				}

				log.Printf("%s provide a bad JWT claims: %s", c.ClientIP(), err.Error())
				requireLogin(opts, c, "Something went wrong with your session. Please reconnect.")
				return
			}

			// Check that required fields are filled
			if claims == nil || len(claims.Profile.UserId) == 0 {
				log.Printf("%s: no UserId found in JWT claims", c.ClientIP())
				requireLogin(opts, c, "Something went wrong with your session. Please reconnect.")
				return
			}

			if claims.Profile.Email == "" {
				log.Printf("%s: no Email found in JWT claims", c.ClientIP())
				requireLogin(opts, c, "Something went wrong with your session. Please reconnect.")
				return
			}

			// Retrieve corresponding user
			user, err := retrieveUserFromClaims(claims)
			userid = user.Id

			if userid != nil {
				method = "jwt"
				if userid == nil || userid.IsEmpty() || !userid.Equals(user.Id) {
					CompleteAuth(opts, c, claims.Profile)
					session.Clear()
					session.Set("iduser", user.Id)
					err = session.Save()
					if err != nil {
						log.Printf("%s: unable to recreate session: %s", c.ClientIP(), err.Error())
						requireLogin(opts, c, "Something went wrong with your session. Please contact your administrator.")
						return
					}
					userid = user.Id
				}
			}
		}

		// Stop here if there is no cookie
		if userid == nil || method == "" {
			if optional {
				c.Next()
			} else {
				requireLogin(opts, c, "No authorization token found in cookie nor in Authorization header.")
			}
			return
		}

		// Retrieve corresponding user
		user, err := storage.MainStore.GetUser(userid)
		if err != nil {
			requireLogin(opts, c, "Unable to retrieve your user. Please reauthenticate.")
			return
		}

		c.Set("AuthMethod", method)
		c.Set("LoggedUser", user)

		// We are now ready to continue
		c.Next()
	}
}
