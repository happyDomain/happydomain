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

	"git.happydns.org/happyDomain/api"
	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/storage"
)

func declareProvidersRoutes(opts *config.Options, router *gin.RouterGroup) {
	router.GET("/providers", getProviders)
	router.POST("/providers", newUserProvider)

	apiProvidersMetaRoutes := router.Group("/providers/:pid")
	apiProvidersMetaRoutes.Use(api.ProviderMetaHandler)

	apiProvidersMetaRoutes.PUT("", api.UpdateProvider)
	apiProvidersMetaRoutes.DELETE("", deleteUserProvider)

	apiProvidersRoutes := router.Group("/providers/:pid")
	apiProvidersRoutes.Use(FindUserProviderHandler)
	apiProvidersRoutes.Use(api.ProviderHandler)

	apiProvidersRoutes.GET("", api.GetProvider)

	declareDomainsRoutes(opts, apiProvidersRoutes)
}

func FindUserProviderHandler(c *gin.Context) {
	if _, ok := c.Get("user"); ok {
		c.Next()
		return
	}

	srcmeta, err := listAllProviders()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to retrieve providers information. Please try again later."})
		return
	}

	for _, src := range srcmeta {
		if src.Id.String() == string(c.Param("pid")) {
			user, err := storage.MainStore.GetUser(src.OwnerId)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to retrieve required users information. Please try again later."})
				return
			}

			c.Set("user", user)

			c.Next()
		}
	}
}

func listAllProviders() ([]happydns.ProviderMeta, error) {
	var providers []happydns.ProviderMeta

	users, err := storage.MainStore.GetUsers()
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		usersProviders, err := storage.MainStore.GetProviderMetas(user)
		if err != nil {
			return nil, err
		}

		providers = append(providers, usersProviders...)
	}

	return providers, err
}

func getProviders(c *gin.Context) {
	user, exists := c.Get("user")
	if exists {
		srcmeta, err := storage.MainStore.GetProviderMetas(user.(*happydns.User))
		ApiResponse(c, srcmeta, err)
	} else {
		providers, err := listAllProviders()
		if err != nil {
			ApiResponse(c, nil, fmt.Errorf("Unable to retrieve providers: %w", err))
		} else {
			ApiResponse(c, providers, nil)
		}
	}
}

func newUserProvider(c *gin.Context) {
	user, exists := c.Get("user")

	if !exists {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "No user specified."})
		return
	}

	us, _, err := api.DecodeProvider(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}
	us.Id = nil

	src, err := storage.MainStore.CreateProvider(user.(*happydns.User), us, "")
	ApiResponse(c, src, err)
}

func deleteUserProvider(c *gin.Context) {
	srcMeta := c.MustGet("providermeta").(*happydns.ProviderMeta)

	ApiResponse(c, true, storage.MainStore.DeleteProvider(srcMeta))
}

func clearProviders(c *gin.Context) {
	ApiResponse(c, true, storage.MainStore.ClearProviders())
}
