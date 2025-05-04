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
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/api/middleware"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
)

type AuthUserController struct {
	auService happydns.AuthUserUsecase
	store     storage.AuthUserStorage
}

func NewAuthUserController(auService happydns.AuthUserUsecase, store storage.AuthUserStorage) *AuthUserController {
	return &AuthUserController{
		auService,
		store,
	}
}

func (ac *AuthUserController) AuthUserHandler(c *gin.Context) {
	user, err := middleware.AuthUserHandlerBase(ac.auService, c)
	if err != nil {
		user, err = ac.store.GetAuthUserByEmail(c.Param("uid"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: "User not found"})
			return
		}
	}

	c.Set("authuser", user)

	c.Next()
}

func (ac *AuthUserController) GetAuthUsers(c *gin.Context) {
	users, err := ac.store.ListAllAuthUsers()
	happydns.ApiResponse(c, users, err)
}

func (ac *AuthUserController) NewAuthUser(c *gin.Context) {
	uu := &happydns.UserAuth{}
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}
	uu.Id = []byte{}

	happydns.ApiResponse(c, uu, ac.store.CreateAuthUser(uu))
}

func (ac *AuthUserController) DeleteAuthUsers(c *gin.Context) {
	happydns.ApiResponse(c, true, ac.store.ClearAuthUsers())
}

func (ac *AuthUserController) GetAuthUser(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	c.JSON(http.StatusOK, user)
}

func (ac *AuthUserController) UpdateAuthUser(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	uu := &happydns.UserAuth{}
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}
	uu.Id = user.Id

	happydns.ApiResponse(c, uu, ac.store.UpdateAuthUser(uu))
}

func (ac *AuthUserController) DeleteAuthUser(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	happydns.ApiResponse(c, true, ac.store.DeleteAuthUser(user))
}

func (ac *AuthUserController) EmailValidationLink(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	happydns.ApiResponse(c, ac.auService.GetValidationLink(user), nil)
}

func (ac *AuthUserController) RecoverUserAcct(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	happydns.ApiResponse(c, ac.auService.GetRecoveryLink(user), nil)
}

type resetPassword struct {
	Password string
}

func (ac *AuthUserController) ResetUserPasswd(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	urp := &resetPassword{}
	err := c.ShouldBindJSON(&urp)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if urp.Password == "" {
		urp.Password, err = utils.GeneratePassword()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
			return
		}
	} else if user.CheckPassword(urp.Password) {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, happydns.ErrorResponse{Message: "The reset password is identical to the current password"})
		return
	}

	err = user.DefinePassword(urp.Password)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	happydns.ApiResponse(c, urp, ac.store.UpdateAuthUser(user))
}

func (ac *AuthUserController) SendRecoverUserAcct(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	happydns.ApiResponse(c, true, ac.auService.SendRecoveryLink(user))
}

func (ac *AuthUserController) SendValidateUserEmail(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	happydns.ApiResponse(c, true, ac.auService.SendValidationLink(user))
}

func (ac *AuthUserController) ValidateEmail(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	now := time.Now()
	user.EmailVerification = &now
	happydns.ApiResponse(c, user, ac.store.UpdateAuthUser(user))
}
