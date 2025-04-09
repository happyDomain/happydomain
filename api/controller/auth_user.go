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

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

type AuthUserController struct {
	auService happydns.AuthUserUsecase
	lc        *LoginController
}

func NewAuthUserController(auService happydns.AuthUserUsecase, lc *LoginController) *AuthUserController {
	return &AuthUserController{
		auService: auService,
		lc:        lc,
	}
}

// changePassword changes the password of the given account.
//
//	@Summary	Change password
//	@Schemes
//	@Description	Change the password of the given account.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			userId	path		string			true	"User identifier"
//	@Param			body	body		happydns.ChangePasswordForm	true	"Password confirmation"
//	@Success		204		{null}		null
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401		{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403		{object}	happydns.ErrorResponse	"Bad current password"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users/{userId}/new_password [post]
func (ac *AuthUserController) ChangePassword(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	var lf happydns.ChangePasswordForm
	if err := c.ShouldBindJSON(&lf); err != nil {
		log.Printf("%s sends invalid passwordForm JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	err := ac.auService.CheckPassword(user, lf)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	err = ac.auService.ChangePassword(user, lf.Password)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	log.Printf("%s changes password for user %s", c.ClientIP(), user.Email)

	ac.lc.Logout(c)
}

// DeleteAuthUser delete the account related to the given user.
//
//	@Summary	Drop account
//	@Schemes
//	@Description	Delete the account related to the given user.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			userId	path		string			true	"User identifier"
//	@Param			body	body		happydns.ChangePasswordForm	true	"Password confirmation"
//	@Success		204		{null}		null
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401		{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403		{object}	happydns.ErrorResponse	"Bad current password"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users/{userId}/delete [post]
func (ac *AuthUserController) DeleteAuthUser(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	var lf happydns.ChangePasswordForm
	if err := c.ShouldBindJSON(&lf); err != nil {
		log.Printf("%s sends invalid passwordForm JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if err := ac.auService.DeleteAuthUser(user, lf.Current); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, happydns.ErrorResponse{Message: "The given current password is invalid."})
		return
	}

	log.Printf("%s: deletes user: %s", c.ClientIP(), user.Email)

	ac.lc.Logout(c)
}
