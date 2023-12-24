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
