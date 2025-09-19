// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

type TestPluginController struct {
	testPluginService happydns.TestPluginUsecase
}

func NewTestPluginController(testPluginService happydns.TestPluginUsecase) *TestPluginController {
	return &TestPluginController{
		testPluginService,
	}
}

// TestPluginHandler is a middleware that retrieves a test plugin by name and sets it in the context.
func (uc *TestPluginController) TestPluginHandler(c *gin.Context) {
	pname := c.Param("pname")

	plugin, err := uc.testPluginService.GetTestPlugin(pname)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: "Plugin not found"})
		return
	}

	c.Set("plugin", plugin)

	c.Next()
}

// ListTestPlugins retrieves all available test plugins.
//
//	@Summary		List all test plugins
//	@Schemes
//	@Description	Returns a list of all available test plugins with their version information.
//	@Tags			plugins
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]happydns.PluginVersionInfo	"Map of plugin names to version info"
//	@Failure		500	{object}	happydns.ErrorResponse					"Internal server error"
//	@Router			/plugins/tests [get]
func (uc *TestPluginController) ListTestPlugins(c *gin.Context) {
	plugins, err := uc.testPluginService.ListTestPlugins()
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	ret := map[string]happydns.PluginVersionInfo{}

	for _, p := range plugins {
		pnames := p.PluginEnvName()
		ret[pnames[0]] = p.Version()
	}

	happydns.ApiResponse(c, ret, nil)
}

// GetTestPluginStatus retrieves the status and available options for a test plugin.
//
//	@Summary		Get test plugin status
//	@Schemes
//	@Description	Retrieves the status information and available options for a specific test plugin.
//	@Tags			plugins
//	@Accept			json
//	@Produce		json
//	@Param			pname	path		string	true	"Plugin name"
//	@Success		200		{object}	happydns.PluginStatus	"Plugin status with version info and available options"
//	@Failure		404		{object}	happydns.ErrorResponse	"Plugin not found"
//	@Router			/plugins/tests/{pname} [get]
func (uc *TestPluginController) GetTestPluginStatus(c *gin.Context) {
	plugin := c.MustGet("plugin").(happydns.TestPlugin)

	c.JSON(http.StatusOK, plugin.Version())
}
