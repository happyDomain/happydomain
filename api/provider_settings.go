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

package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/forms"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/providers"
	"git.happydns.org/happyDomain/storage"
)

func declareProviderSettingsRoutes(cfg *config.Options, router *gin.RouterGroup) {
	router.POST("/providers/_specs/:ssid/settings", func(c *gin.Context) {
		getProviderSettingsState(cfg, c)
	})
	//router.POST("/domains/:domain/zone/:zoneid/:subdomain/provider_settings/:psid", getProviderSettingsState)
}

type ProviderSettingsState struct {
	FormState
	happydns.Provider `json:"Provider" swaggertype:"object"`
}

type ProviderSettingsResponse struct {
	Provider *happydns.Provider     `json:"Provider,omitempty" swaggertype:"object"`
	Values   map[string]interface{} `json:"values,omitempty"`
	Form     *forms.CustomForm      `json:"form,omitempty"`
}

// getProviderSettingsState creates or updates a Provider with human fillable forms.
//
//	@Summary	Assistant to Provider creation.
//	@Schemes
//	@Description	This creates or updates a Provider with human fillable forms.
//	@Tags			provider_specs
//	@Accept			json
//	@Produce		json
//	@Param			providerType	path	string					true	"The provider's type"
//	@Param			body			body	ProviderSettingsState	true	"The current state of the Provider's settings, possibly empty (but not null)"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	ProviderSettingsResponse	"The settings need more rafinement"
//	@Success		200	{object}	happydns.ProviderCombined	"The Provider has been created with the given settings"
//	@Failure		400	{object}	happydns.Error				"Invalid input"
//	@Failure		401	{object}	happydns.Error				"Authentication failure"
//	@Failure		404	{object}	happydns.Error				"Provider not found"
//	@Router			/providers/_specs/{providerType}/settings [post]
func getProviderSettingsState(cfg *config.Options, c *gin.Context) {
	user := myUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined"})
		return
	}

	ssid := string(c.Param("ssid"))

	src, err := providers.FindProvider(ssid)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": fmt.Sprintf("Unable to find your provider: %s", err.Error())})
		return
	}

	var uss ProviderSettingsState
	uss.Provider = src
	err = c.ShouldBindJSON(&uss)
	if err != nil {
		log.Printf("%s sends invalid ProviderSettingsState JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	form, p, err := formDoState(cfg, c, &uss.FormState, src, forms.GenDefaultSettingsForm)

	if err != nil {
		if err != forms.DoneForm {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
			return
		} else if cfg.DisableProviders {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Cannot change provider settings as DisableProviders parameter is set."})
			return
		} else if _, err = src.NewDNSServiceProvider(); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
			return
		} else if uss.Id == nil {
			// Create a new Provider
			s, err := storage.MainStore.CreateProvider(user, src, uss.Name)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
				return
			}

			c.JSON(http.StatusOK, s)
			return
		} else {
			// Update an existing Provider
			s, err := storage.MainStore.GetProvider(user, *uss.Id)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
				return
			}
			s.Comment = uss.Name
			s.Provider = uss.Provider

			err = storage.MainStore.UpdateProvider(s)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
				return
			}

			c.JSON(http.StatusOK, s)
			return
		}
	}

	c.JSON(http.StatusOK, ProviderSettingsResponse{
		Form:   form,
		Values: p,
	})
}
