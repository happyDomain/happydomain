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

// TestPluginOptionHandler is a middleware that retrieves a specific plugin option and sets it in the context.
func (uc *TestPluginController) TestPluginOptionHandler(c *gin.Context) {
	pname := c.Param("pname")
	optname := c.Param("optname")

	opts, err := uc.testPluginService.GetTestPluginOptions(pname, nil, nil, nil)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	c.Set("option", (*opts)[optname])

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

	c.JSON(http.StatusOK, happydns.PluginStatus{
		PluginVersionInfo: plugin.Version(),
		Opts:              plugin.AvailableOptions(),
	})
}

// GetTestPluginOptions retrieves all options for a test plugin.
//
//	@Summary		Get test plugin options
//	@Schemes
//	@Description	Retrieves all configuration options for a specific test plugin.
//	@Tags			plugins
//	@Accept			json
//	@Produce		json
//	@Param			pname	path		string	true	"Plugin name"
//	@Success		200		{object}	happydns.PluginOptions	"Plugin options as key-value pairs"
//	@Failure		404		{object}	happydns.ErrorResponse	"Plugin not found"
//	@Failure		500		{object}	happydns.ErrorResponse	"Internal server error"
//	@Router			/plugins/tests/{pname}/options [get]
func (uc *TestPluginController) GetTestPluginOptions(c *gin.Context) {
	pname := c.Param("pname")

	opts, err := uc.testPluginService.GetTestPluginOptions(pname, nil, nil, nil)
	happydns.ApiResponse(c, opts, err)
}

// AddTestPluginOptions adds or overwrites specific options for a test plugin.
//
//	@Summary		Add test plugin options
//	@Schemes
//	@Description	Adds or overwrites specific configuration options for a test plugin without affecting other options.
//	@Tags			plugins
//	@Accept			json
//	@Produce		json
//	@Param			pname	path		string								true	"Plugin name"
//	@Param			body	body		happydns.SetPluginOptionsRequest	true	"Options to add or overwrite"
//	@Success		200		{object}	bool								"Success status"
//	@Failure		400		{object}	happydns.ErrorResponse				"Invalid request body"
//	@Failure		404		{object}	happydns.ErrorResponse				"Plugin not found"
//	@Failure		500		{object}	happydns.ErrorResponse				"Internal server error"
//	@Router			/plugins/tests/{pname}/options [post]
func (uc *TestPluginController) AddTestPluginOptions(c *gin.Context) {
	pname := c.Param("pname")

	var req happydns.SetPluginOptionsRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	err = uc.testPluginService.OverwriteSomeTestPluginOptions(pname, nil, nil, nil, req.Options)
	happydns.ApiResponse(c, true, err)
}

// ChangeTestPluginOptions replaces all options for a test plugin.
//
//	@Summary		Replace test plugin options
//	@Schemes
//	@Description	Replaces all configuration options for a test plugin with the provided options.
//	@Tags			plugins
//	@Accept			json
//	@Produce		json
//	@Param			pname	path		string								true	"Plugin name"
//	@Param			body	body		happydns.SetPluginOptionsRequest	true	"New complete set of options"
//	@Success		200		{object}	bool								"Success status"
//	@Failure		400		{object}	happydns.ErrorResponse				"Invalid request body"
//	@Failure		404		{object}	happydns.ErrorResponse				"Plugin not found"
//	@Failure		500		{object}	happydns.ErrorResponse				"Internal server error"
//	@Router			/plugins/tests/{pname}/options [put]
func (uc *TestPluginController) ChangeTestPluginOptions(c *gin.Context) {
	pname := c.Param("pname")

	var req happydns.SetPluginOptionsRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	err = uc.testPluginService.SetTestPluginOptions(pname, nil, nil, nil, req.Options)
	happydns.ApiResponse(c, true, err)
}

// GetTestPluginOption retrieves a specific option value for a test plugin.
//
//	@Summary		Get test plugin option
//	@Schemes
//	@Description	Retrieves the value of a specific configuration option for a test plugin.
//	@Tags			plugins
//	@Accept			json
//	@Produce		json
//	@Param			pname		path		string	true	"Plugin name"
//	@Param			optname		path		string	true	"Option name"
//	@Success		200			{object}	object	"Option value (type varies)"
//	@Failure		404			{object}	happydns.ErrorResponse	"Plugin not found"
//	@Failure		500			{object}	happydns.ErrorResponse	"Internal server error"
//	@Router			/plugins/tests/{pname}/options/{optname} [get]
func (uc *TestPluginController) GetTestPluginOption(c *gin.Context) {
	opt := c.MustGet("option")

	happydns.ApiResponse(c, opt, nil)
}

// SetTestPluginOption sets or updates a specific option value for a test plugin.
//
//	@Summary		Set test plugin option
//	@Schemes
//	@Description	Sets or updates the value of a specific configuration option for a test plugin.
//	@Tags			plugins
//	@Accept			json
//	@Produce		json
//	@Param			pname		path		string	true	"Plugin name"
//	@Param			optname		path		string	true	"Option name"
//	@Param			body		body		object	true	"Option value (type varies by option)"
//	@Success		200			{object}	bool	"Success status"
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid request body"
//	@Failure		404			{object}	happydns.ErrorResponse	"Plugin not found"
//	@Failure		500			{object}	happydns.ErrorResponse	"Internal server error"
//	@Router			/plugins/tests/{pname}/options/{optname} [put]
func (uc *TestPluginController) SetTestPluginOption(c *gin.Context) {
	pname := c.Param("pname")
	optname := c.Param("optname")

	var req interface{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	po := happydns.PluginOptions{}
	po[optname] = req

	err = uc.testPluginService.OverwriteSomeTestPluginOptions(pname, nil, nil, nil, po)
	happydns.ApiResponse(c, true, err)
}
