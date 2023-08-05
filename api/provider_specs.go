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

package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happydomain/forms"
	"git.happydns.org/happydomain/model"
	"git.happydns.org/happydomain/providers"
)

func declareProviderSpecsRoutes(router *gin.RouterGroup) {
	router.GET("/providers/_specs", listProviders)

	router.GET("/providers/_specs/:psid/icon.png", getProviderSpecIcon)

	apiProviderSpecsRoutes := router.Group("/providers/_specs/:psid")
	apiProviderSpecsRoutes.Use(ProviderSpecsHandler)

	apiProviderSpecsRoutes.GET("", getProviderSpec)
}

// listProviders returns the static list of usable providers in this happyDomain release.
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
	src := c.MustGet("providertype").(happydns.Provider)

	c.JSON(http.StatusOK, viewProviderSpec{
		Fields:       forms.GenStructFields(src),
		Capabilities: providers.GetProviderCapabilities(src),
	})
}
