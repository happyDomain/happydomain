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

func (lc *LoginController) GetLoggedUser(c *gin.Context) {
	c.JSON(http.StatusOK, c.MustGet("LoggedUser"))
}

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

	session := sessions.Default(c)

	session.Clear()
	session.Set("iduser", user.Id)
	err = session.Save()
	if err != nil {
		log.Printf("%s: unable to save user session: %s", c.ClientIP(), err.Error())
		c.JSON(http.StatusUnauthorized, happydns.ErrorResponse{Message: "Invalid username or password."})
		return
	}

	log.Printf("%s: now logged as %q\n", c.ClientIP(), user.Email)

	c.JSON(http.StatusOK, user)
}

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
