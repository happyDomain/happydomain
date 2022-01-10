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

	"github.com/gin-gonic/gin"

	"git.happydns.org/happydomain/api"
	"git.happydns.org/happydomain/config"
	"git.happydns.org/happydomain/model"
	"git.happydns.org/happydomain/storage"
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
	uu.Id = []byte{}

	ApiResponse(c, uu, storage.MainStore.CreateUser(uu))
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
