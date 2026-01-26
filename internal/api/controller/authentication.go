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

package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

type LoginController struct {
	authService happydns.AuthenticationUsecase
}

func NewLoginController(authService happydns.AuthenticationUsecase) *LoginController {
	return &LoginController{
		authService: authService,
	}
}

// GetLoggedUser retrieves the currently logged-in user.
//
//	@Summary	Get the current user.
//	@Schemes
//	@Description	Retrieve information about the currently logged-in user.
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.User
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Router			/auth/user [get]
func (lc *LoginController) GetLoggedUser(c *gin.Context) {
	c.JSON(http.StatusOK, c.MustGet("LoggedUser"))
}

// Login authenticates a user with username and password.
//
//	@Summary	Log in a user.
//	@Schemes
//	@Description	Authenticate a user with email and password, creating a new session.
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Param			body	body		happydns.LoginRequest	true	"Login credentials"
//	@Success		200		{object}	happydns.User
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401		{object}	happydns.ErrorResponse	"Invalid username or password"
//	@Router			/auth/login [post]
func (lc *LoginController) Login(c *gin.Context) {
	var request happydns.LoginRequest

	err := c.ShouldBindJSON(&request)
	if err != nil {
		log.Printf("%s sends invalid LoginForm JSON: %s", c.ClientIP(), err.Error())
		c.JSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	user, err := lc.authService.AuthenticateUserWithPassword(request)
	if err != nil {
		log.Printf("%s: %s", c.ClientIP(), err.Error())
		c.JSON(http.StatusUnauthorized, happydns.ErrorResponse{Message: "Invalid username or password."})
		return
	}

	middleware.SessionLoginOK(c, user)

	c.JSON(http.StatusOK, user)
}

// Logout clears the current user's session.
//
//	@Summary	Log out the current user.
//	@Schemes
//	@Description	Clear the current user's session and log them out.
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		204	"Session cleared"
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/auth/logout [post]
func (lc *LoginController) Logout(c *gin.Context) {
	session := sessions.Default(c)

	session.Clear()
	err := session.Save()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
