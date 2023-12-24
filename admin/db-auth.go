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
	"time"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/actions"
	"git.happydns.org/happyDomain/api"
	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/storage"
	"git.happydns.org/happyDomain/utils"
)

func declareUserAuthsRoutes(opts *config.Options, router *gin.RouterGroup) {
	router.GET("/auth", getAuthUsers)
	router.POST("/auth", newAuthUser)
	router.DELETE("/auth", deleteAuthUsers)

	apiUsersRoutes := router.Group("/auth/:uid")
	apiUsersRoutes.Use(authHandler)

	apiUsersRoutes.GET("", getAuthUser)
	apiUsersRoutes.PUT("", updateAuthUser)
	apiUsersRoutes.DELETE("", deleteAuthUser)

	apiUsersRoutes.POST("/recover_link", func(c *gin.Context) {
		recoverUserAcct(opts, c)
	})
	apiUsersRoutes.POST("/reset_password", resetUserPasswd)
	apiUsersRoutes.POST("/send_recover_email", func(c *gin.Context) {
		sendRecoverUserAcct(opts, c)
	})
	apiUsersRoutes.POST("/send_validation_email", func(c *gin.Context) {
		sendValidateUserEmail(opts, c)
	})
	apiUsersRoutes.POST("/validation_link", func(c *gin.Context) {
		emailValidationLink(opts, c)
	})
	apiUsersRoutes.POST("/validate_email", validateEmail)
}

func authHandler(c *gin.Context) {
	user, err := api.UserAuthHandlerBase(c)
	if err != nil {
		user, err = storage.MainStore.GetAuthUserByEmail(c.Param("uid"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "User not found"})
			return
		}
	}

	c.Set("authuser", user)

	c.Next()
}

func getAuthUsers(c *gin.Context) {
	users, err := storage.MainStore.GetAuthUsers()
	ApiResponse(c, users, err)
}

func newAuthUser(c *gin.Context) {
	uu := &happydns.UserAuth{}
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}
	uu.Id = []byte{}

	ApiResponse(c, uu, storage.MainStore.CreateAuthUser(uu))
}

func deleteAuthUsers(c *gin.Context) {
	ApiResponse(c, true, storage.MainStore.ClearAuthUsers())
}

func getAuthUser(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	c.JSON(http.StatusOK, user)
}

func updateAuthUser(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	uu := &happydns.UserAuth{}
	err := c.ShouldBindJSON(&uu)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}
	uu.Id = user.Id

	ApiResponse(c, uu, storage.MainStore.UpdateAuthUser(uu))
}

func deleteAuthUser(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	ApiResponse(c, true, storage.MainStore.DeleteAuthUser(user))
}

func emailValidationLink(opts *config.Options, c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	ApiResponse(c, opts.GetRegistrationURL(user), nil)
}

func recoverUserAcct(opts *config.Options, c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	ApiResponse(c, opts.GetAccountRecoveryURL(user), nil)
}

type resetPassword struct {
	Password string
}

func resetUserPasswd(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	urp := &resetPassword{}
	err := c.ShouldBindJSON(&urp)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if urp.Password == "" {
		urp.Password, err = utils.GeneratePassword()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
			return
		}
	} else if user.CheckAuth(urp.Password) {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"errmsg": "The reset password is identical to the current password"})
		return
	}

	err = user.DefinePassword(urp.Password)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	ApiResponse(c, urp, storage.MainStore.UpdateAuthUser(user))
}

func sendRecoverUserAcct(opts *config.Options, c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	ApiResponse(c, true, actions.SendRecoveryLink(opts, user))
}

func sendValidateUserEmail(opts *config.Options, c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	ApiResponse(c, true, actions.SendValidationLink(opts, user))
}

func validateEmail(c *gin.Context) {
	user := c.MustGet("authuser").(*happydns.UserAuth)

	now := time.Now()
	user.EmailVerification = &now
	ApiResponse(c, user, storage.MainStore.UpdateAuthUser(user))
}
