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
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/internal/session"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/storage"
)

const NO_AUTH_ACCOUNT = "_no_auth"

func declareAuthenticationRoutes(opts *config.Options, router *gin.RouterGroup) {
	router.POST("/auth", func(c *gin.Context) {
		checkAuth(opts, c)
	})
	router.POST("/auth/logout", func(c *gin.Context) {
		logout(opts, c)
	})

	apiAuthRoutes := router.Group("/auth")
	apiAuthRoutes.Use(authMiddleware(opts, true))

	apiAuthRoutes.GET("", func(c *gin.Context) {
		if _, exists := c.Get("LoggedUser"); exists {
			displayAuthToken(c)
		} else {
			displayNotAuthToken(opts, c)
		}
	})
}

type DisplayUser struct {
	// Id is the user identifier
	Id happydns.Identifier `json:"id" swaggertype:"string"`

	// Email is the user email.
	Email string `json:"email"`

	// CreatedAt stores the date of the account creation.
	CreatedAt time.Time `json:"created_at,omitempty"`

	// Settings holds the user configuration.
	Settings happydns.UserSettings `json:"settings,omitempty"`
}

func currentUser(u *happydns.User) *DisplayUser {
	return &DisplayUser{
		Id:        u.Id,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		Settings:  u.Settings,
	}
}

// displayAuthToken returns the user information.
//
//	@Summary	User info.
//	@Schemes
//	@Description	Retrieve information about the currently logged user.
//	@Tags			user_auth
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	DisplayUser
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Router			/auth [get]
func displayAuthToken(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)

	c.JSON(http.StatusOK, currentUser(user))
}

func displayNotAuthToken(opts *config.Options, c *gin.Context) *UserClaims {
	if !opts.NoAuth {
		requireLogin(opts, c, "Authorization required")
		return nil
	}

	claims, err := completeAuth(opts, c, UserProfile{
		UserId:        []byte{0},
		Email:         NO_AUTH_ACCOUNT,
		EmailVerified: true,
	})
	if err != nil {
		log.Printf("%s %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Something went wrong during your authentication. Please retry in a few minutes"})
		return nil
	}

	realUser, err := retrieveUserFromClaims(claims)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"errmsg": "Login success"})
	} else {
		c.JSON(http.StatusOK, currentUser(realUser))
	}

	return claims
}

// logout closes the user session.
//
//	@Summary	Close session.
//	@Schemes
//	@Description	Erase the HTTP-only cookie. This leads to user logout in its browser.
//	@Tags			user_auth
//	@Accept			json
//	@Produce		json
//	@Success		204	{null}		null			"Loged out"
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Router			/auth/logout [post]
func logout(opts *config.Options, c *gin.Context) {
	c.SetCookie(
		session.COOKIE_NAME,
		"",
		-1,
		opts.BaseURL+"/",
		"",
		opts.DevProxy == "" && opts.ExternalURL.URL.Scheme != "http",
		true,
	)
	c.Status(http.StatusNoContent)
}

type LoginForm struct {
	// Email of the user.
	Email string

	// Password of the user.
	Password string
}

// checkAuth validate user authentication and delivers a session token.
//
//	@Summary	Authenticate user.
//	@Schemes
//	@Description	Validate user authentication and delivers a session token.
//	@Tags			user_auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginForm		true	"Login information"
//	@Success		200		{object}	DisplayUser		"Login success"
//	@Failure		401		{object}	happydns.Error	"Authentication failure"
//	@Failure		500		{object}	happydns.Error
//	@Router			/auth [post]
func checkAuth(opts *config.Options, c *gin.Context) {
	var lf LoginForm
	if err := c.ShouldBindJSON(&lf); err != nil {
		log.Printf("%s sends invalid LoginForm JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	user, err := storage.MainStore.GetAuthUserByEmail(lf.Email)
	if err != nil {
		log.Printf("%s user's email (%s) not found: %s", c.ClientIP(), lf.Email, err.Error())
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "Invalid username or password."})
		return
	}

	if !user.CheckAuth(lf.Password) {
		log.Printf("%s tries to login as %q, but sent an invalid password", c.ClientIP(), lf.Email)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "Invalid username or password."})
		return
	}

	if user.EmailVerification == nil {
		log.Printf("%s tries to login as %q, but has not verified email", c.ClientIP(), lf.Email)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "Please validate your e-mail address before your first login.", "href": "/email-validation"})
		return
	}

	claims, err := completeAuth(opts, c, UserProfile{
		UserId:        user.Id,
		Email:         user.Email,
		EmailVerified: user.EmailVerification != nil,
		CreatedAt:     user.CreatedAt,
		Newsletter:    user.AllowCommercials,
	})
	if err != nil {
		log.Printf("%s %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Something went wrong during your authentication. Please retry in a few minutes"})
		return
	}

	log.Printf("%s now logged as %q\n", c.ClientIP(), user.Email)

	realUser, err := retrieveUserFromClaims(claims)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"errmsg": "Login success"})
	} else {
		c.JSON(http.StatusOK, currentUser(realUser))
	}
}

func completeAuth(opts *config.Options, c *gin.Context, userprofile UserProfile) (*UserClaims, error) {
	session := sessions.Default(c)

	session.Clear()
	session.Set("iduser", userprofile.UserId)
	err := session.Save()
	if err != nil {
		return nil, err
	}

	return &UserClaims{
		userprofile,
		jwt.RegisteredClaims{},
	}, nil
}
