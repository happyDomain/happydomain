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
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/forms"
	"git.happydns.org/happyDomain/providers"
)

func declareProviderSpecsRoutes(router *gin.RouterGroup) {
	router.GET("/providers/_specs", listProviders)

	router.GET("/providers/_specs/:psid/icon.png", getProviderSpecIcon)

	apiProviderSpecsRoutes := router.Group("/providers/_specs/:psid")
	apiProviderSpecsRoutes.Use(ProviderSpecsHandler)

	apiProviderSpecsRoutes.GET("", getProviderSpec)
}

// listProviders returns the static list of usable providers in this happyDomain release.
//
//	@Summary	List all providers with which you can connect.
//	@Schemes
//	@Description	This returns the static list of usable providers in this happyDomain release.
//	@Tags			provider_specs
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]providers.ProviderInfos{}	"The list"
//	@Router			/providers/_specs [get]
func listProviders(c *gin.Context) {
	srcs := providers.GetProviders()

	ret := map[string]providers.ProviderInfos{}
	for k, src := range *srcs {
		ret[k] = src.Infos
	}

	c.JSON(http.StatusOK, ret)
}

// getProviderSpecIcon returns the icon as image/png.
//
//	@Summary	Get the PNG icon.
//	@Schemes
//	@Description	Return the icon as a image/png file for the given provider type.
//	@Tags			provider_specs
//	@Accept			json
//	@Produce		png
//	@Param			providerType	path		string	true	"The provider's type"
//	@Success		200				{file}		png
//	@Failure		404				{object}	happydns.Error	"Provider type does not exist"
//	@Router			/providers/_specs/{providerType}/icon.png [get]
func getProviderSpecIcon(c *gin.Context) {
	psid := string(c.Param("psid"))

	cnt, ok := providers.Icons[strings.TrimSuffix(psid, ".png")]
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Icon not found."})
		return
	}

	c.Data(http.StatusOK, "image/png", cnt)
}

func ProviderSpecsHandler(c *gin.Context) {
	psid := string(c.Param("psid"))

	src, err := providers.FindProvider(psid)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": fmt.Sprintf("Unable to find provider: %s", err.Error())})
		return
	}

	c.Set("providertype", src)

	c.Next()
}

type viewProviderSpec struct {
	// Fields describes the settings needed to configure the provider.
	Fields []*forms.Field `json:"fields,omitempty"`

	// Capabilities exposes what the provider can do.
	Capabilities []string `json:"capabilities,omitempty"`
}

// getProviderSpec returns a description of the expected settings and the provider capabilities.
//
//	@Summary	Get the provider capabilities and expected settings.
//	@Schemes
//	@Description	Return a description of the expected settings and the provider capabilities.
//	@Tags			provider_specs
//	@Accept			json
//	@Produce		json
//	@Param			providerType	path		string	true	"The provider's type"
//	@Success		200				{object}	viewProviderSpec
//	@Failure		404				{object}	happydns.Error	"Provider type does not exist"
//	@Router			/providers/_specs/{providerType} [get]
func getProviderSpec(c *gin.Context) {
	src := c.MustGet("providertype").(providers.Provider)

	c.JSON(http.StatusOK, viewProviderSpec{
		Fields:       forms.GenStructFields(src),
		Capabilities: providers.GetProviderCapabilities(src),
	})
}
