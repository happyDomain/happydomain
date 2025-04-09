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

package middleware

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

func UserHandlerBase(userService happydns.UserUsecase, c *gin.Context) (*happydns.User, error) {
	uid, err := base64.RawURLEncoding.DecodeString(c.Param("uid"))
	if err != nil {
		return nil, fmt.Errorf("Invalid user identifier given: %w", err)
	}

	user, err := userService.GetUser(uid)
	if err != nil {
		return nil, fmt.Errorf("User not found")
	}

	return user, nil
}

func UserHandler(userService happydns.UserUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := UserHandlerBase(userService, c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: err.Error()})
			return
		}

		c.Set("user", user)

		c.Next()
	}
}

func MyUser(c *gin.Context) (user *happydns.User) {
	if u, exists := c.Get("LoggedUser"); exists {
		user = u.(*happydns.User)
	} else if u, exists := c.Get("user"); exists {
		user = u.(*happydns.User)
	}
	return
}

func SameUserHandler(c *gin.Context) {
	myuser := c.MustGet("LoggedUser").(*happydns.User)
	user := c.MustGet("user").(*happydns.User)

	if !bytes.Equal(user.Id, myuser.Id) {
		log.Printf("%s: tries to do action as %s (logged %s)", c.ClientIP(), myuser.Id, user.Id)
		c.AbortWithStatusJSON(http.StatusForbidden, happydns.ErrorResponse{Message: "Not authorized"})
		return
	}

	c.Next()
}
