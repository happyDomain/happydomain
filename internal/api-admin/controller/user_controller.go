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

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/usecase/user"
	"git.happydns.org/happyDomain/model"
)

type UserController struct {
	userService happydns.UserUsecase
	store       user.UserStorage
}

func NewUserController(store user.UserStorage, userService happydns.UserUsecase) *UserController {
	return &UserController{
		userService,
		store,
	}
}

func (uc *UserController) UserHandler(c *gin.Context) {
	user, err := middleware.UserHandlerBase(uc.userService, c)
	if err != nil {
		user, err = uc.store.GetUserByEmail(c.Param("uid"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: "User not found"})
			return
		}
	}

	c.Set("user", user)

	c.Next()
}

// getUsers retrieves all users from the database.
//
//	@Summary		List all users.
//	@Schemes
//	@Description	Retrieve a list of all users in the system.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Success		200		{array}		happydns.User			"List of users"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users [get]
func (uc *UserController) GetUsers(c *gin.Context) {
	iter, err := uc.store.ListAllUsers()
	if err != nil {
		happydns.ApiResponse(c, nil, err)
		return
	}

	var users []*happydns.User
	for iter.Next() {
		users = append(users, iter.Item())
	}

	happydns.ApiResponse(c, users, err)
}

// newUser creates a new user in the database.
//
//	@Summary		Create a new user.
//	@Schemes
//	@Description	Create a new user account with the provided information.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		happydns.User			true	"User information"
//	@Success		200		{object}	happydns.User			"The created user"
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users [post]
func (uc *UserController) NewUser(c *gin.Context) {
	uu := &happydns.User{}
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	happydns.ApiResponse(c, uu, uc.store.CreateOrUpdateUser(uu))
}

// deleteUsers deletes all users from the database.
//
//	@Summary		Delete all users.
//	@Schemes
//	@Description	Remove all user accounts from the system.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Success		200		{boolean}	bool					"Success status"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users [delete]
func (uc *UserController) DeleteUsers(c *gin.Context) {
	happydns.ApiResponse(c, true, uc.store.ClearUsers())
}

// getUser retrieves a specific user from the database.
//
//	@Summary		Show user.
//	@Schemes
//	@Description	Retrieve a user's complete information by their ID or email.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					true	"User ID or email"
//	@Success		200		{object}	happydns.User			"The user"
//	@Failure		404		{object}	happydns.ErrorResponse	"User not found"
//	@Router			/users/{uid} [get]
func (uc *UserController) GetUser(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	c.JSON(http.StatusOK, user)
}

// updateUser updates an existing user's information.
//
//	@Summary		Update user.
//	@Schemes
//	@Description	Update a user's information. The user ID is preserved from the URL parameter.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					true	"User ID or email"
//	@Param			body	body		happydns.User			true	"Updated user information"
//	@Success		200		{object}	happydns.User			"The updated user"
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		404		{object}	happydns.ErrorResponse	"User not found"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users/{uid} [put]
func (uc *UserController) UpdateUser(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	uu := &happydns.User{}
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}
	uu.Id = user.Id

	happydns.ApiResponse(c, uu, uc.store.CreateOrUpdateUser(uu))
}

// deleteUser removes a specific user from the database.
//
//	@Summary		Delete user.
//	@Schemes
//	@Description	Delete a user account and all associated data.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					true	"User ID or email"
//	@Success		200		{boolean}	bool					"Success status"
//	@Failure		404		{object}	happydns.ErrorResponse	"User not found"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users/{uid} [delete]
func (uc *UserController) DeleteUser(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	happydns.ApiResponse(c, true, uc.store.DeleteUser(user.Id))
}
