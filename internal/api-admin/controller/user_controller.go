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
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type UserController struct {
	userService happydns.UserUsecase
	store       storage.UserStorage
}

func NewUserController(store storage.Storage, userService happydns.UserUsecase) *UserController {
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

func (uc *UserController) NewUser(c *gin.Context) {
	uu := &happydns.User{}
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	happydns.ApiResponse(c, uu, uc.store.CreateOrUpdateUser(uu))
}

func (uc *UserController) DeleteUsers(c *gin.Context) {
	happydns.ApiResponse(c, true, uc.store.ClearUsers())
}

func (uc *UserController) GetUser(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	c.JSON(http.StatusOK, user)
}

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

func (uc *UserController) DeleteUser(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	happydns.ApiResponse(c, true, uc.store.DeleteUser(user.Id))
}
