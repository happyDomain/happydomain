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

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/internal/usecase/authuser"
	"git.happydns.org/happyDomain/model"
)

type AuthUserController struct {
	auService happydns.AuthUserUsecase
	store     authuser.AuthUserStorage
}

func NewAuthUserController(auService happydns.AuthUserUsecase, store authuser.AuthUserStorage) *AuthUserController {
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

// GetAuthUsers retrieves a list of all registered users.
//
//	@Summary		List all users
//	@Schemes
//	@Description	Retrieve a list of all registered users in the system.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}		happydns.UserAuth
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/auth [get]
func (ac *AuthUserController) GetAuthUsers(c *gin.Context) {
	iter, err := ac.store.ListAllAuthUsers()
	if err != nil {
		happydns.ApiResponse(c, nil, err)
		return
	}
	defer iter.Close()

	var users []*happydns.UserAuth
	for iter.Next() {
		users = append(users, iter.Item())
	}

	happydns.ApiResponse(c, users, err)
}

// NewAuthUser creates a new user account.
//
//	@Summary		Create new user
//	@Schemes
//	@Description	Create a new user account in the system.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		happydns.UserAuth	true	"User data"
//	@Success		200		{object}	happydns.UserAuth
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/auth [post]
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

// DeleteAuthUsers deletes all user accounts.
//
//	@Summary		Delete all users
//	@Schemes
//	@Description	Delete all user accounts from the system.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Success		200	{boolean}	true
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/auth [delete]
func (ac *AuthUserController) DeleteAuthUsers(c *gin.Context) {
	happydns.ApiResponse(c, true, ac.store.ClearAuthUsers())
}

// GetAuthUser retrieves a specific user by identifier.
//
//	@Summary		Get user details
//	@Schemes
//	@Description	Retrieve details for a specific user account.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid	path		string	true	"User identifier (email or ID)"
//	@Success		200	{object}	happydns.UserAuth
//	@Failure		404	{object}	happydns.ErrorResponse	"User not found"
//	@Router			/auth/{uid} [get]
func (ac *AuthUserController) GetAuthUser(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	c.JSON(http.StatusOK, user)
}

// UpdateAuthUser updates an existing user account.
//
//	@Summary		Update user
//	@Schemes
//	@Description	Update an existing user account's information.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string				true	"User identifier (email or ID)"
//	@Param			body	body		happydns.UserAuth	true	"Updated user data"
//	@Success		200		{object}	happydns.UserAuth
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		404		{object}	happydns.ErrorResponse	"User not found"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/auth/{uid} [put]
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

// DeleteAuthUser deletes a specific user account.
//
//	@Summary		Delete user
//	@Schemes
//	@Description	Delete a specific user account from the system.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid	path		string	true	"User identifier (email or ID)"
//	@Success		200	{boolean}	true
//	@Failure		404	{object}	happydns.ErrorResponse	"User not found"
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/auth/{uid} [delete]
func (ac *AuthUserController) DeleteAuthUser(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	happydns.ApiResponse(c, true, ac.store.DeleteAuthUser(user))
}

// EmailValidationLink generates an email validation link for a user.
//
//	@Summary		Generate email validation link
//	@Schemes
//	@Description	Generate a validation link for verifying user's email address.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid	path		string	true	"User identifier (email or ID)"
//	@Success		200	{string}	string	"Validation link"
//	@Failure		404	{object}	happydns.ErrorResponse	"User not found"
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/auth/{uid}/validation_link [post]
func (ac *AuthUserController) EmailValidationLink(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	happydns.ApiResponse(c, ac.auService.GenerateValidationLink(user), nil)
}

// RecoverUserAcct generates an account recovery link for a user.
//
//	@Summary		Generate account recovery link
//	@Schemes
//	@Description	Generate a recovery link for user account access.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid	path		string	true	"User identifier (email or ID)"
//	@Success		200	{string}	string	"Recovery link"
//	@Failure		404	{object}	happydns.ErrorResponse	"User not found"
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/auth/{uid}/recover_link [post]
func (ac *AuthUserController) RecoverUserAcct(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	link, err := ac.auService.GenerateRecoveryLink(user)
	happydns.ApiResponse(c, link, err)
}

type resetPassword struct {
	Password string
}

// ResetUserPasswd resets a user's password.
//
//	@Summary		Reset user password
//	@Schemes
//	@Description	Reset a user's password. If no password is provided, a random one will be generated.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string			true	"User identifier (email or ID)"
//	@Param			body	body		resetPassword	true	"New password (optional)"
//	@Success		200		{object}	resetPassword	"New password"
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		404		{object}	happydns.ErrorResponse	"User not found"
//	@Failure		406		{object}	happydns.ErrorResponse	"Password identical to current"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/auth/{uid}/reset_password [post]
func (ac *AuthUserController) ResetUserPasswd(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	urp := &resetPassword{}
	err := c.ShouldBindJSON(&urp)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if urp.Password == "" {
		urp.Password, err = helpers.GeneratePassword()
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

// SendRecoverUserAcct sends an account recovery email to the user.
//
//	@Summary		Send account recovery email
//	@Schemes
//	@Description	Send an account recovery link to the user's email address.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid	path		string	true	"User identifier (email or ID)"
//	@Success		200	{boolean}	true
//	@Failure		404	{object}	happydns.ErrorResponse	"User not found"
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/auth/{uid}/send_recover_email [post]
func (ac *AuthUserController) SendRecoverUserAcct(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	happydns.ApiResponse(c, true, ac.auService.SendRecoveryLink(user))
}

// SendValidateUserEmail sends an email validation link to the user.
//
//	@Summary		Send email validation link
//	@Schemes
//	@Description	Send an email validation link to the user's email address.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid	path		string	true	"User identifier (email or ID)"
//	@Success		200	{boolean}	true
//	@Failure		404	{object}	happydns.ErrorResponse	"User not found"
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/auth/{uid}/send_validation_email [post]
func (ac *AuthUserController) SendValidateUserEmail(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	happydns.ApiResponse(c, true, ac.auService.SendValidationLink(user))
}

// ValidateEmail marks a user's email as verified.
//
//	@Summary		Validate user email
//	@Schemes
//	@Description	Mark a user's email address as verified.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid	path		string	true	"User identifier (email or ID)"
//	@Success		200	{object}	happydns.UserAuth
//	@Failure		404	{object}	happydns.ErrorResponse	"User not found"
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/auth/{uid}/validate_email [post]
func (ac *AuthUserController) ValidateEmail(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	now := time.Now()
	user.EmailVerification = &now
	happydns.ApiResponse(c, user, ac.store.UpdateAuthUser(user))
}
