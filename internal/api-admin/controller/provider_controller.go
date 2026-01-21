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
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/controller"
	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
)

type ProviderController struct {
	providerService happydns.ProviderUsecase
	store           provider.ProviderStorage
}

func NewProviderController(providerService happydns.ProviderUsecase, store provider.ProviderStorage) *ProviderController {
	return &ProviderController{
		providerService,
		store,
	}
}

// ListProviders retrieves all providers or user-specific providers.
//
//	@Summary		List all providers (admin)
//	@Schemes
//	@Description	List all DNS providers in the system, or providers for a specific user if user context is provided.
//	@Tags			admin-providers
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					false	"User ID or email"
//	@Success		200	{array}		happydns.ProviderMeta
//	@Failure		500	{object}	happydns.ErrorResponse	"Unable to list providers"
//	@Router			/providers [get]
//	@Router			/users/{uid}/providers [get]
func (pc *ProviderController) ListProviders(c *gin.Context) {
	user := middleware.MyUser(c)
	if user != nil {
		srcmeta, err := pc.store.ListProviders(user)
		happydns.ApiResponse(c, srcmeta.Metas(), err)
		return
	}

	iter, err := pc.store.ListAllProviders()
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, fmt.Errorf("unable to list providers: %w", err))
		return
	}
	defer iter.Close()

	var res []*happydns.ProviderMeta
	for iter.Next() {
		provider := iter.Item()
		res = append(res, &provider.ProviderMeta)
	}

	happydns.ApiResponse(c, res, nil)
}

// AddProvider appends a new provider.
//
//	@Summary		Add a new provider (admin)
//	@Schemes
//	@Description	Append a new DNS provider to the system.
//	@Tags			admin-providers
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string						false	"User ID or email"
//	@Param			body	body		happydns.ProviderMinimal	true	"Provider to add"
//	@Success		200		{object}	happydns.Provider
//	@Failure		400		{object}	happydns.ErrorResponse	"Error in received data"
//	@Failure		500		{object}	happydns.ErrorResponse	"Unable to create provider"
//	@Router			/providers [post]
//	@Router			/users/{uid}/providers [post]
func (pc *ProviderController) AddProvider(c *gin.Context) {
	apidc := controller.NewProviderController(pc.providerService)
	apidc.AddProvider(c)
	return
}

// DeleteProvider removes a provider from the database.
//
//	@Summary		Delete a provider (admin)
//	@Schemes
//	@Description	Delete a DNS provider from the system database.
//	@Tags			admin-providers
//	@Accept			json
//	@Produce		json
//	@Param			uid	path		string	false	"User ID or email"
//	@Param			pid	path		string	true	"Provider identifier"
//	@Success		200	{boolean}	true
//	@Failure		404	{object}	happydns.ErrorResponse	"Provider not found"
//	@Failure		500	{object}	happydns.ErrorResponse	"Database deletion error"
//	@Router			/providers/{pid} [delete]
//	@Router			/users/{uid}/providers/{pid} [delete]
func (pc *ProviderController) DeleteProvider(c *gin.Context) {
	srcMeta := c.MustGet("providermeta").(*happydns.ProviderMeta)

	happydns.ApiResponse(c, true, pc.store.DeleteProvider(srcMeta.Id))
}

// GetProvider retrieves information about a given provider.
//
//	@Summary		Retrieve provider information (admin)
//	@Schemes
//	@Description	Retrieve information in the database about a given DNS provider.
//	@Tags			admin-providers
//	@Accept			json
//	@Produce		json
//	@Param			uid	path		string	false	"User ID or email"
//	@Param			pid	path		string	true	"Provider identifier"
//	@Success		200	{object}	happydns.Provider
//	@Failure		404	{object}	happydns.ErrorResponse	"Provider not found"
//	@Router			/providers/{pid} [get]
//	@Router			/users/{uid}/providers/{pid} [get]
func (pc *ProviderController) GetProvider(c *gin.Context) {
	apidc := controller.NewProviderController(pc.providerService)
	apidc.GetProvider(c)
	return
}

// UpdateProvider updates the information about a given provider.
//
//	@Summary		Update provider information (admin)
//	@Schemes
//	@Description	Updates the information about a given DNS provider in the system.
//	@Tags			admin-providers
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string				false	"User ID or email"
//	@Param			pid		path		string				true	"Provider identifier"
//	@Param			body	body		happydns.Provider	true	"The new object overriding the current provider"
//	@Success		200		{object}	happydns.Provider
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		404		{object}	happydns.ErrorResponse	"Provider not found"
//	@Failure		500		{object}	happydns.ErrorResponse	"Database writing error"
//	@Router			/providers/{pid} [put]
//	@Router			/users/{uid}/providers/{pid} [put]
func (pc *ProviderController) UpdateProvider(c *gin.Context) {
	apidc := controller.NewProviderController(pc.providerService)
	apidc.UpdateProvider(c)
	return
}

// ClearProviders removes all providers from the database.
//
//	@Summary		Clear all providers (admin)
//	@Schemes
//	@Description	Delete all DNS providers from the system, or all providers for a specific user if user context is provided.
//	@Tags			admin-providers
//	@Accept			json
//	@Produce		json
//	@Param			uid	path		string	false	"User ID or email"
//	@Success		200	{boolean}	true
//	@Failure		500	{object}	happydns.ErrorResponse	"Database deletion error"
//	@Router			/providers [delete]
//	@Router			/users/{uid}/providers [delete]
func (pc *ProviderController) ClearProviders(c *gin.Context) {
	user := middleware.MyUser(c)
	if user != nil {
		providers, err := pc.providerService.ListUserProviders(user)
		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}

		for _, p := range providers {
			e := pc.store.DeleteProvider(p.Id)
			if e != nil {
				err = errors.Join(err, e)
			}
		}

		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusOK)
		return
	}

	happydns.ApiResponse(c, true, pc.store.ClearProviders())
}
