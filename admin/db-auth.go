// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package admin

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happydomain/actions"
	"git.happydns.org/happydomain/api"
	"git.happydns.org/happydomain/config"
	"git.happydns.org/happydomain/model"
	"git.happydns.org/happydomain/storage"
	"git.happydns.org/happydomain/utils"
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
