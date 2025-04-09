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
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/api/middleware"
	"git.happydns.org/happyDomain/model"
)

type ProviderController struct {
	providerService happydns.ProviderUsecase
}

func NewProviderController(providerService happydns.ProviderUsecase) *ProviderController {
	return &ProviderController{
		providerService: providerService,
	}
}

// ListProviders retrieves all providers belonging to the user.
//
//	@Summary	Retrieve user's providers
//	@Schemes
//	@Description	Retrieve all DNS providers belonging to the user.
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{array}		happydns.ProviderMeta
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Unable to retrieve user's domains"
//	@Router			/providers [get]
func (pc *ProviderController) ListProviders(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined"})
		return
	}

	providers, err := pc.providerService.ListUserProviders(user)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, providers)
}

// GetProvider retrieves information about a given Provider owned by the user.
//
//	@Summary	Retrieve Provider information.
//	@Schemes
//	@Description	Retrieve information in the database about a given Provider owned by the user.
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Param			providerId	path	string	true	"Provider identifier"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Provider
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Provider not found"
//	@Router			/providers/{providerId} [get]
func (pc *ProviderController) GetProvider(c *gin.Context) {
	provider := c.MustGet("provider").(*happydns.Provider)

	c.JSON(http.StatusOK, provider)
}

// addProvider appends a new provider.
//
//	@Summary	Add a new provider
//	@Schemes
//	@Description	Append a new provider for the user.
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Param			body	body	happydns.ProviderMinimal	true	"Provider to add"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Provider
//	@Failure		400	{object}	happydns.ErrorResponse	"Error in received data"
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		500	{object}	happydns.ErrorResponse	"Unable to retrieve current user's providers"
//	@Router			/providers [post]
func (pc *ProviderController) AddProvider(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("No user specified."))
		return
	}

	var usrc happydns.ProviderMessage
	err := c.ShouldBindJSON(&usrc)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to decode given provider: %s", err.Error())})
		return
	}

	provider, err := pc.providerService.CreateProvider(user, &usrc)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, provider)
}

// UpdateProvider updates the information about a given Provider owned by the user.
//
//	@Summary	Update Provider information.
//	@Schemes
//	@Description	Updates the information about a given Provider owned by the user.
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Param			providerId	path	string				true	"Provider identifier"
//	@Param			body		body	happydns.Provider	true	"The new object overriding the current provider"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Provider
//	@Failure		400	{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		400	{object}	happydns.ErrorResponse	"Identifier changed"
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Provider not found"
//	@Failure		500	{object}	happydns.ErrorResponse	"Database writing error"
//	@Router			/providers/{providerId} [put]
func (pc *ProviderController) UpdateProvider(c *gin.Context) {
	old := c.MustGet("provider").(*happydns.Provider)
	user := middleware.MyUser(c)

	var provider happydns.ProviderMessage
	err := c.ShouldBindJSON(&provider)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	err = pc.providerService.UpdateProviderFromMessage(old.Id, user, &provider)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, old)
}

// DeleteProvider removes a provider from the database.
//
//	@Summary	Delete a Provider.
//	@Schemes
//	@Description	Delete a Provider from the database. It is required that no Domain are still managed by this Provider before calling this route.
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Param			providerId	path	string	true	"Provider identifier"
//	@Security		securitydefinitions.basic
//	@Success		204	"Provider deleted"
//	@Failure		400	{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Provider not found"
//	@Failure		500	{object}	happydns.ErrorResponse	"Database writing error"
//	@Router			/providers/{providerId} [delete]
func (pc *ProviderController) DeleteProvider(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("User not defined."))
		return
	}

	providermeta := c.MustGet("providermeta").(*happydns.ProviderMeta)

	err := pc.providerService.DeleteProvider(user, providermeta.Id)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// getDomainsHostedByProvider lists domains available to management from the given Provider.
//
//	@Summary	Lists manageable domains from the Provider.
//	@Schemes
//	@Description	List domains available from the given Provider.
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Param			providerId	path	string	true	"Provider identifier"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Provider
//	@Failure		400	{object}	happydns.ErrorResponse	"Unable to instantiate the provider"
//	@Failure		400	{object}	happydns.ErrorResponse	"The provider doesn't support domain listing"
//	@Failure		400	{object}	happydns.ErrorResponse	"Provider error"
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Provider not found"
//	@Router			/providers/{providerId}/domains [get]
func (pc *ProviderController) GetDomainsHostedByProvider(c *gin.Context) {
	provider := c.MustGet("provider").(*happydns.Provider)

	p, err := provider.InstantiateProvider()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to instantiate the provider: %s", err.Error())})
		return
	}

	zl, ok := p.(happydns.ZoneListerActuator)
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Provider doesn't support domain listing."})
		return
	}

	domains, err := zl.ListZones()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, domains)
}
