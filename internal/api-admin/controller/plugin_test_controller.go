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

func (uc *TestPluginController) GetTestPluginStatus(c *gin.Context) {
	plugin := c.MustGet("plugin").(happydns.TestPlugin)

	c.JSON(http.StatusOK, happydns.PluginStatus{
		PluginVersionInfo: plugin.Version(),
		Opts:              plugin.AvailableOptions(),
	})
}

func (uc *TestPluginController) GetTestPluginOptions(c *gin.Context) {
	pname := c.Param("pname")

	opts, err := uc.testPluginService.GetTestPluginOptions(pname, nil, nil, nil)
	happydns.ApiResponse(c, opts, err)
}

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

func (uc *TestPluginController) GetTestPluginOption(c *gin.Context) {
	opt := c.MustGet("option")

	happydns.ApiResponse(c, opt, nil)
}

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
