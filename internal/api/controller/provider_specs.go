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
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

type ProviderSpecsController struct {
	pSpecsServices happydns.ProviderSpecsUsecase
}

func NewProviderSpecsController(pSpecsServices happydns.ProviderSpecsUsecase) *ProviderSpecsController {
	return &ProviderSpecsController{
		pSpecsServices: pSpecsServices,
	}
}

// ListProviders returns the static list of usable providers in this happyDomain release.
//
//	@Summary	List all providers with which you can connect.
//	@Schemes
//	@Description	This returns the static list of usable providers in this happyDomain release.
//	@Tags			provider_specs
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]happydns.ProviderInfos{}	"The list"
//	@Router			/providers/_specs [get]
func (psc *ProviderSpecsController) ListProviders(c *gin.Context) {
	c.JSON(http.StatusOK, psc.pSpecsServices.ListProviders())
}

// GetProviderSpecIcon returns the provider's icon.
//
//	@Summary	Get the provider's icon.
//	@Schemes
//	@Description	Return the icon for the given provider type. It is served as image/png by default, or with the content type fetched from the provider's website when available (e.g. svg, webp, ico).
//	@Tags			provider_specs
//	@Accept			json
//	@Produce		octet-stream
//	@Param			providerType	path		string	true	"The provider's type"
//	@Success		200				{file}		binary
//	@Failure		404				{object}	happydns.ErrorResponse	"Provider type does not exist"
//	@Router			/providers/_specs/{providerType}/icon [get]
func (psc *ProviderSpecsController) GetProviderSpecIcon(c *gin.Context) {
	psid := string(c.Param("psid"))

	cnt, contentType, err := psc.pSpecsServices.GetProviderIcon(psid)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	// The icon may have been fetched from the provider's website, so it gets
	// the same treatment as any other third-party bytes served from our origin.
	middleware.ServeSandboxedImage(c, contentType, cnt)
}

// GetProviderSpec returns a description of the expected settings and the provider capabilities.
//
//	@Summary	Get the provider capabilities and expected settings.
//	@Schemes
//	@Description	Return a description of the expected settings and the provider capabilities.
//	@Tags			provider_specs
//	@Accept			json
//	@Produce		json
//	@Param			providerType	path		string	true	"The provider's type"
//	@Success		200				{object}	happydns.ProviderSpecs
//	@Failure		404				{object}	happydns.ErrorResponse	"Provider type does not exist"
//	@Router			/providers/_specs/{providerType} [get]
func (psc *ProviderSpecsController) GetProviderSpec(c *gin.Context) {
	psid := string(c.Param("psid"))

	specs, err := psc.pSpecsServices.GetProviderSpecs(psid)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, specs)
}
