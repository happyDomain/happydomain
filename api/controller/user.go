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
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

type UserController struct {
	lc          *LoginController
	userService happydns.UserUsecase
}

func NewUserController(userService happydns.UserUsecase, lc *LoginController) *UserController {
	return &UserController{
		lc:          lc,
		userService: userService,
	}
}

// getUser shows a user in the database.
//
//	@Summary	Show user.
//	@Schemes
//	@Description	Show a user from the database, information is limited to id and email if this is not the current user.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	happydns.User		"The created user"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users/{userId} [get]
func (uc *UserController) GetUser(c *gin.Context) {
	myuser := c.MustGet("LoggedUser").(*happydns.User)
	user := c.MustGet("user").(*happydns.User)

	if bytes.Equal(user.Id, myuser.Id) {
		c.JSON(http.StatusOK, user)
	} else {
		c.JSON(http.StatusOK, &happydns.User{
			Id:    user.Id,
			Email: user.Email,
		})
	}
}

// getUserAvatar returns a unique avatar for the user.
//
//	@Summary	Show user's avatar.
//	@Schemes
//	@Description	Returns a unique avatar for the user.
//	@Tags			users
//	@Accept			json
//	@Produce		png
//	@Param			size	query	int	false	"Image output desired size"
//	@Success		200		{file}		png
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/users/{userId}/avatar.png [get]
func (uc *UserController) GetUserAvatar(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	sizequery := c.DefaultQuery("size", "300")
	size, err := strconv.ParseInt(sizequery, 10, 32)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Invalid size asked: %s", err.Error())})
		return
	} else if size > 2048 {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: "Size too large."})
		return
	}

	c.Writer.Header().Set("Content-Type", "image/png")
	c.Writer.WriteHeader(http.StatusOK)

	err = uc.userService.GenerateUserAvatar(user, int(size), c.Writer)
	if err != nil {
		log.Printf("Unable to generate user avatar (uid=%s,user=%s): %s", user.Id.String(), user.Email, err.Error())
	}
}

// getUserSettings gets the settings of the given user.
//
//	@Summary	Retrieve user's settings.
//	@Schemes
//	@Description	Retrieve the user's settings.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			userId	path	string	true	"User identifier"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.UserSettings	"User settings"
//	@Failure		401	{object}	happydns.ErrorResponse			"Authentication failure"
//	@Failure		403	{object}	happydns.ErrorResponse			"Not your account"
//	@Router			/users/{userId}/settings [get]
func (uc *UserController) GetUserSettings(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	c.JSON(http.StatusOK, user.Settings)
}

// changeUserSettings updates the settings of the given user.
//
//	@Summary	Update user's settings.
//	@Schemes
//	@Description	Update the user's settings.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			userId	path	string					true	"User identifier"
//	@Param			body	body	happydns.UserSettings	true	"User settings"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.UserSettings	"User settings"
//	@Failure		400	{object}	happydns.ErrorResponse			"Invalid input"
//	@Failure		401	{object}	happydns.ErrorResponse			"Authentication failure"
//	@Failure		403	{object}	happydns.ErrorResponse			"Not your account"
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/users/{userId}/settings [post]
func (uc *UserController) ChangeUserSettings(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	var us happydns.UserSettings
	if err := c.ShouldBindJSON(&us); err != nil {
		log.Printf("%s sends invalid UserSettings JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if err := uc.userService.ChangeUserSettings(user, us); err != nil {
		log.Printf("%s: unable to UpdateUser in changeUserSettings: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to update your profile. Please try again later."})
		return
	}

	c.JSON(http.StatusOK, user.Settings)
}

func (uc *UserController) DeleteMyUser(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	if err := uc.userService.DeleteUser(user.Id); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, happydns.ErrorResponse{Message: "The given current password is invalid."})
		return
	}

	log.Printf("%s: deletes user: %s", c.ClientIP(), user.Email)

	uc.lc.Logout(c)
}
