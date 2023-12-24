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

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"git.happydns.org/happyDomain/actions"
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

const COOKIE_NAME = "happydomain_session"

var signingMethod = jwt.SigningMethodHS512

func updateUserFromClaims(user *happydns.User, claims *UserClaims) {
	user.Email = claims.Profile.Email
	user.LastSeen = time.Now()
}

func retrieveUserFromClaims(claims *UserClaims) (user *happydns.User, err error) {
	user, err = storage.MainStore.GetUser(claims.Profile.UserId)
	if err != nil {
		// The user doesn't exists yet: create it!
		user = &happydns.User{
			Id:        claims.Profile.UserId,
			Email:     claims.Profile.Email,
			CreatedAt: time.Now(),
			LastSeen:  time.Now(),
			Settings:  *happydns.DefaultUserSettings(),
		}

		err = storage.MainStore.UpdateUser(user)
		if err != nil {
			err = fmt.Errorf("has a correct JWT, but an error occured when trying to create the user: %w", err)
			return
		}

		if claims.Profile.Newsletter {
			err = actions.SubscribeToNewsletter(user)
			if err != nil {
				err = fmt.Errorf("something goes wrong during newsletter subscription: %w", err)
				return
			}
		}
	} else if time.Since(user.LastSeen) > time.Hour*12 {
		// Update user's data when connected more than 12 hours
		updateUserFromClaims(user, claims)

		err = storage.MainStore.UpdateUser(user)
		if err != nil {
			err = fmt.Errorf("has a correct JWT, user has been found, but an error occured when trying to update the user's information: %w", err)
			return
		}
	}

	return
}

func retrieveSessionFromClaims(claims *UserClaims, user *happydns.User, session_id []byte) (session *happydns.Session, err error) {
	session, err = storage.MainStore.GetSession(session_id)
	if err != nil {
		// The session doesn't exists yet: create it!
		session = &happydns.Session{
			Id:       session_id,
			IdUser:   claims.Profile.UserId,
			IssuedAt: time.Now(),
		}

		err = storage.MainStore.UpdateSession(session)
		if err != nil {
			err = fmt.Errorf("has a correct JWT, but an error occured when trying to create the session: %w", err)
			return
		}

		// Update user's data
		updateUserFromClaims(user, claims)

		err = storage.MainStore.UpdateUser(user)
		if err != nil {
			err = fmt.Errorf("has a correct JWT, session has been created, but an error occured when trying to update the user's information: %w", err)
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
		var token string

		// Retrieve the token from cookie or header
		if cookie, err := c.Cookie(COOKIE_NAME); err == nil {
			token = cookie
		} else if flds := strings.Fields(c.GetHeader("Authorization")); len(flds) == 2 && flds[0] == "Bearer" {
			token = flds[1]
		}

		// Stop here if there is no cookie
		if len(token) == 0 {
			if optional {
				c.Next()
			} else {
				requireLogin(opts, c, "No authorization token found in cookie nor in Authorization header.")
			}
			return
		}

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
			c.SetCookie(COOKIE_NAME, "", -1, opts.BaseURL+"/", "", opts.DevProxy == "", true)
			requireLogin(opts, c, "Something went wrong with your session. Please reconnect.")
			return
		}

		// Check that required fields are filled
		if claims == nil || len(claims.Profile.UserId) == 0 {
			log.Printf("%s: no UserId found in JWT claims", c.ClientIP())
			c.SetCookie(COOKIE_NAME, "", -1, opts.BaseURL+"/", "", opts.DevProxy == "", true)
			requireLogin(opts, c, "Something went wrong with your session. Please reconnect.")
			return
		}

		if claims.Profile.Email == "" {
			log.Printf("%s: no Email found in JWT claims", c.ClientIP())
			c.SetCookie(COOKIE_NAME, "", -1, opts.BaseURL+"/", "", opts.DevProxy == "", true)
			requireLogin(opts, c, "Something went wrong with your session. Please reconnect.")
			return
		}

		// Retrieve corresponding user
		user, err := retrieveUserFromClaims(claims)
		if err != nil {
			log.Printf("%s %s", c.ClientIP(), err.Error())
			c.SetCookie(COOKIE_NAME, "", -1, opts.BaseURL+"/", "", opts.DevProxy == "", true)
			requireLogin(opts, c, "Something went wrong with your session. Please reconnect.")
			return
		}

		c.Set("LoggedUser", user)

		// Retrieve the session
		session_id := append([]byte(claims.Profile.UserId), []byte(claims.ID)...)
		session, err := retrieveSessionFromClaims(claims, user, session_id)
		if err != nil {
			log.Printf("%s %s", c.ClientIP(), err.Error())
			c.SetCookie(COOKIE_NAME, "", -1, opts.BaseURL+"/", "", opts.DevProxy == "", true)
			requireLogin(opts, c, "Your session has expired. Please reconnect.")
			return
		}

		c.Set("MySession", session)

		// We are now ready to continue
		c.Next()

		// On return, check if the session has changed
		if session.HasChanged() {
			storage.MainStore.UpdateSession(session)
		}
	}
}
