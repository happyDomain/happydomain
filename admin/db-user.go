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

package admin

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/api"
	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/storage"
)

func declareUsersRoutes(opts *config.Options, router *gin.RouterGroup) {
	router.GET("/users", getUsers)
	router.POST("/users", newUser)
	router.DELETE("/users", deleteUsers)

	apiUsersRoutes := router.Group("/users/:uid")
	apiUsersRoutes.Use(userHandler)

	apiUsersRoutes.GET("", getUser)
	apiUsersRoutes.PUT("", updateUser)
	apiUsersRoutes.DELETE("", deleteUser)

	declareDomainsRoutes(opts, apiUsersRoutes)
	declareProvidersRoutes(opts, apiUsersRoutes)

	router.POST("/tidy", tidyDB)
}

func userHandler(c *gin.Context) {
	user, err := api.UserHandlerBase(c)
	if err != nil {
		user, err = storage.MainStore.GetUserByEmail(c.Param("uid"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "User not found"})
			return
		}
	}

	c.Set("user", user)

	c.Next()
}

func getUsers(c *gin.Context) {
	users, err := storage.MainStore.GetUsers()
	ApiResponse(c, users, err)
}

func newUser(c *gin.Context) {
	uu := &happydns.User{}
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if uu.Id.IsEmpty() {
		ApiResponse(c, uu, storage.MainStore.CreateUser(uu))
	} else {
		ApiResponse(c, uu, storage.MainStore.UpdateUser(uu))
	}
}

func deleteUsers(c *gin.Context) {
	ApiResponse(c, true, storage.MainStore.ClearUsers())
}

func getUser(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	c.JSON(http.StatusOK, user)
}

func updateUser(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	uu := &happydns.User{}
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}
	uu.Id = user.Id

	ApiResponse(c, uu, storage.MainStore.UpdateUser(uu))
}

func deleteUser(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	ApiResponse(c, true, storage.MainStore.DeleteUser(user))
}

func tidyDB(c *gin.Context) {
	ApiResponse(c, true, storage.MainStore.Tidy())
}
