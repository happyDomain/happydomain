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

// BaseTestPluginController contains shared functionality for test plugin controllers.
// It provides common methods that can be used by both admin and user-scoped controllers.
type BaseTestPluginController struct {
	testPluginService happydns.TestPluginUsecase
}

func NewBaseTestPluginController(testPluginService happydns.TestPluginUsecase) *BaseTestPluginController {
	return &BaseTestPluginController{
		testPluginService,
	}
}

// GetTestPluginService returns the test plugin service for use by derived controllers.
func (bc *BaseTestPluginController) GetTestPluginService() happydns.TestPluginUsecase {
	return bc.testPluginService
}

// ListTestPlugins retrieves all available test plugins.
func (bc *BaseTestPluginController) ListTestPlugins(c *gin.Context) {
	plugins, err := bc.testPluginService.ListTestPlugins()
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
func (bc *BaseTestPluginController) GetTestPluginStatus(c *gin.Context) {
	plugin := c.MustGet("plugin").(happydns.TestPlugin)

	c.JSON(http.StatusOK, happydns.PluginStatus{
		PluginVersionInfo: plugin.Version(),
		Opts:              plugin.AvailableOptions(),
	})
}

// GetTestPluginOptionsWithScope retrieves all options for a test plugin with the given scope.
func (bc *BaseTestPluginController) GetTestPluginOptionsWithScope(c *gin.Context, pname string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) {
	opts, err := bc.testPluginService.GetTestPluginOptions(pname, userId, domainId, serviceId)
	happydns.ApiResponse(c, opts, err)
}

// AddTestPluginOptionsWithScope adds or overwrites specific options for a test plugin with the given scope.
func (bc *BaseTestPluginController) AddTestPluginOptionsWithScope(c *gin.Context, pname string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) {
	var req happydns.SetPluginOptionsRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	err = bc.testPluginService.OverwriteSomeTestPluginOptions(pname, userId, domainId, serviceId, req.Options)
	happydns.ApiResponse(c, true, err)
}

// ChangeTestPluginOptionsWithScope replaces all options for a test plugin with the given scope.
func (bc *BaseTestPluginController) ChangeTestPluginOptionsWithScope(c *gin.Context, pname string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) {
	var req happydns.SetPluginOptionsRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	err = bc.testPluginService.SetTestPluginOptions(pname, userId, domainId, serviceId, req.Options)
	happydns.ApiResponse(c, true, err)
}

// GetTestPluginOptionValue retrieves a specific option value from the context.
func (bc *BaseTestPluginController) GetTestPluginOptionValue(c *gin.Context) {
	opt := c.MustGet("option")

	happydns.ApiResponse(c, opt, nil)
}

// SetTestPluginOptionWithScope sets or updates a specific option value for a test plugin with the given scope.
func (bc *BaseTestPluginController) SetTestPluginOptionWithScope(c *gin.Context, pname string, optname string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) {
	var req interface{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	po := happydns.PluginOptions{}
	po[optname] = req

	err = bc.testPluginService.OverwriteSomeTestPluginOptions(pname, userId, domainId, serviceId, po)
	happydns.ApiResponse(c, true, err)
}
