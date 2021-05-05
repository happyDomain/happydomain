// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
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

	"git.happydns.org/happydns/forms"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/sources"
)

func declareSourceSpecsRoutes(router *gin.RouterGroup) {
	router.GET("/source_specs", getSourceSpecs)

	router.GET("/source_specs/:ssid/icon.png", getSourceSpecIcon)

	apiSourceSpecsRoutes := router.Group("/source_specs/:ssid")
	apiSourceSpecsRoutes.Use(SourceSpecsHandler)

	apiSourceSpecsRoutes.GET("", getSourceSpec)
}

func getSourceSpecs(c *gin.Context) {
	srcs := sources.GetSources()

	ret := map[string]sources.SourceInfos{}
	for k, src := range *srcs {
		src.Infos.Capabilities = sources.GetSourceCapabilities(src.Creator())
		ret[k] = src.Infos
	}

	c.JSON(http.StatusOK, ret)
}

func getSourceSpecIcon(c *gin.Context) {
	ssid := string(c.Param("ssid"))

	if cnt, ok := sources.Icons[strings.TrimSuffix(ssid, ".png")]; ok {
		c.Data(http.StatusOK, "image/png", cnt)
	} else {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Icon not found."})
	}
}

func SourceSpecsHandler(c *gin.Context) {
	ssid := string(c.Param("ssid"))

	src, err := sources.FindSource(ssid)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": fmt.Sprintf("Unable to find source: %w", err)})
		return
	}

	c.Set("sourcetype", src)

	c.Next()
}

type viewSourceSpec struct {
	Fields       []*forms.Field `json:"fields,omitempty"`
	Capabilities []string       `json:"capabilities,omitempty"`
}

func getSourceSpec(c *gin.Context) {
	src := c.MustGet("sourcetype").(happydns.Source)

	c.JSON(http.StatusOK, viewSourceSpec{
		Fields:       forms.GenStructFields(src),
		Capabilities: sources.GetSourceCapabilities(src),
	})
}
