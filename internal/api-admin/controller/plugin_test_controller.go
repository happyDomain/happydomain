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

	apicontroller "git.happydns.org/happyDomain/internal/api/controller"
	"git.happydns.org/happyDomain/model"
)

// TestPluginController handles admin-level plugin operations.
// All methods in this controller work with admin-scoped options (nil user/domain/service IDs).
type TestPluginController struct {
	*apicontroller.BaseTestPluginController
}

func NewTestPluginController(testPluginService happydns.TestPluginUsecase) *TestPluginController {
	return &TestPluginController{
		BaseTestPluginController: apicontroller.NewBaseTestPluginController(testPluginService),
	}
}

// TestPluginHandler is a middleware that retrieves a test plugin by name and sets it in the context.
func (uc *TestPluginController) TestPluginHandler(c *gin.Context) {
	pname := c.Param("pname")

	plugin, err := uc.BaseTestPluginController.GetTestPluginService().GetTestPlugin(pname)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: "Plugin not found"})
		return
	}

	c.Set("plugin", plugin)

	c.Next()
}

// TestPluginOptionHandler is a middleware that retrieves a specific admin-level plugin option and sets it in the context.
func (uc *TestPluginController) TestPluginOptionHandler(c *gin.Context) {
	pname := c.Param("pname")
	optname := c.Param("optname")

	// Get admin-level options (nil user/domain/service IDs)
	opts, err := uc.BaseTestPluginController.GetTestPluginService().GetTestPluginOptions(pname, nil, nil, nil)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	c.Set("option", (*opts)[optname])

	c.Next()
}

// ListTestPlugins retrieves all available test plugins.
//
//	@Summary		List test plugins (admin)
//	@Schemes
//	@Description	Retrieves a list of all available test plugins with their version information.
//	@Tags			plugins
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]happydns.PluginVersionInfo	"Map of plugin name to version info"
//	@Failure		500	{object}	happydns.ErrorResponse					"Internal server error"
//	@Router			/plugins/tests [get]
func (uc *TestPluginController) ListTestPlugins(c *gin.Context) {
	uc.BaseTestPluginController.ListTestPlugins(c)
}

// GetTestPluginStatus retrieves the status and available options for a test plugin.
//
//	@Summary		Get test plugin status (admin)
//	@Schemes
//	@Description	Retrieves the status and available configuration options for a specific test plugin.
//	@Tags			plugins
//	@Accept			json
//	@Produce		json
//	@Param			pname	path		string					true	"Plugin name"
//	@Success		200		{object}	happydns.PluginStatus	"Plugin status with available options"
//	@Failure		404		{object}	happydns.ErrorResponse	"Plugin not found"
//	@Router			/plugins/tests/{pname} [get]
func (uc *TestPluginController) GetTestPluginStatus(c *gin.Context) {
	uc.BaseTestPluginController.GetTestPluginStatus(c)
}

// GetTestPluginOptions retrieves all admin-level options for a test plugin.
//
//	@Summary		Get test plugin options (admin)
//	@Schemes
//	@Description	Retrieves all admin-level configuration options for a specific test plugin.
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

	// Get admin-level options (nil user/domain/service IDs)
	uc.GetTestPluginOptionsWithScope(c, pname, nil, nil, nil)
}

// AddTestPluginOptions adds or overwrites specific admin-level options for a test plugin.
//
//	@Summary		Add test plugin options (admin)
//	@Schemes
//	@Description	Adds or overwrites specific admin-level configuration options for a test plugin without affecting other options.
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

	// Add admin-level options (nil user/domain/service IDs)
	uc.AddTestPluginOptionsWithScope(c, pname, nil, nil, nil)
}

// ChangeTestPluginOptions replaces all admin-level options for a test plugin.
//
//	@Summary		Replace test plugin options (admin)
//	@Schemes
//	@Description	Replaces all admin-level configuration options for a test plugin with the provided options.
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

	// Replace admin-level options (nil user/domain/service IDs)
	uc.ChangeTestPluginOptionsWithScope(c, pname, nil, nil, nil)
}

// GetTestPluginOption retrieves a specific admin-level option value for a test plugin.
//
//	@Summary		Get test plugin option (admin)
//	@Schemes
//	@Description	Retrieves the value of a specific admin-level configuration option for a test plugin.
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
	uc.GetTestPluginOptionValue(c)
}

// SetTestPluginOption sets or updates a specific admin-level option value for a test plugin.
//
//	@Summary		Set test plugin option (admin)
//	@Schemes
//	@Description	Sets or updates the value of a specific admin-level configuration option for a test plugin.
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

	// Set admin-level option (nil user/domain/service IDs)
	uc.SetTestPluginOptionWithScope(c, pname, optname, nil, nil, nil)
}
