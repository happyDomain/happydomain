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
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

type ProviderSettingsController struct {
	pSettingsServices happydns.ProviderSettingsUsecase
}

func NewProviderSettingsController(pSettingsServices happydns.ProviderSettingsUsecase) *ProviderSettingsController {
	return &ProviderSettingsController{
		pSettingsServices: pSettingsServices,
	}
}

// NextProviderSettingsState creates or updates a Provider with human fillable forms.
//
//	@Summary	Assistant to Provider creation.
//	@Schemes
//	@Description	This creates or updates a Provider with human fillable forms.
//	@Tags			provider_specs
//	@Accept			json
//	@Produce		json
//	@Param			providerType	path	string					true	"The provider's type"
//	@Param			body			body	happydns.ProviderSettingsState	true	"The current state of the Provider's settings, possibly empty (but not null)"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Provider	"The Provider has been created with the given settings"
//	@Success		202	{object}	happydns.ProviderSettingsResponse	"The settings need more rafinement"
//	@Failure		400	{object}	happydns.ErrorResponse				"Invalid input"
//	@Failure		401	{object}	happydns.ErrorResponse				"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse				"Provider not found"
//	@Router			/providers/_specs/{providerType}/settings [post]
func (psc *ProviderSettingsController) NextProviderSettingsState(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined"})
		return
	}

	specs := c.MustGet("providerspecs").(happydns.ProviderBody)
	pType := c.MustGet("providertype").(string)

	var uss happydns.ProviderSettingsState
	uss.ProviderBody = specs
	err := c.ShouldBindJSON(&uss)
	if err != nil {
		log.Printf("%s sends invalid ProviderSettingsState JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	provider, form, err := psc.pSettingsServices.NextProviderSettingsState(&uss, pType, user)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	if provider != nil {
		c.JSON(http.StatusOK, provider)
	} else {
		c.JSON(http.StatusAccepted, form)
	}
}
