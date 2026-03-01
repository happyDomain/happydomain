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
	"reflect"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

type ServiceSpecsController struct {
	sSpecsServices happydns.ServiceSpecsUsecase
}

func NewServiceSpecsController(sSpecsServices happydns.ServiceSpecsUsecase) *ServiceSpecsController {
	return &ServiceSpecsController{
		sSpecsServices: sSpecsServices,
	}
}

// ListServiceSpecs returns the static list of usable services in this happyDomain release.
//
//	@Summary	List all services with which you can connect.
//	@Schemes
//	@Description	This returns the static list of usable services in this happyDomain release.
//	@Tags			service_specs
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]happydns.ServiceInfos{}	"The list"
//	@Router			/service_specs [get]
func (ssc *ServiceSpecsController) ListServiceSpecs(c *gin.Context) {
	c.JSON(http.StatusOK, ssc.sSpecsServices.ListServices())
}

// GetServiceSpecIcon returns the icon as image/png.
//
//	@Summary	Get the PNG icon.
//	@Schemes
//	@Description	Return the icon as a image/png file for the given service type.
//	@Tags			service_specs
//	@Accept			json
//	@Produce		png
//	@Param			serviceType	path		string	true	"The service's type"
//	@Success		200			{file}		png
//	@Failure		404			{object}	happydns.ErrorResponse	"Service type does not exist"
//	@Router			/service_specs/{serviceType}/icon.png [get]
func (ssc *ServiceSpecsController) GetServiceSpecIcon(c *gin.Context) {
	ssid := string(c.Param("ssid"))

	cnt, err := ssc.sSpecsServices.GetServiceIcon(ssid)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.Data(http.StatusOK, "image/png", cnt)
}

// getServiceSpec returns a description of the expected fields.
//
//	@Summary	Get the service expected fields.
//	@Schemes
//	@Description	Return a description of the expected fields.
//	@Tags			service_specs
//	@Accept			json
//	@Produce		json
//	@Param			serviceType	path		string	true	"The service's type"
//	@Success		200			{object}	happydns.ServiceSpecs
//	@Failure		404			{object}	happydns.ErrorResponse	"Service type does not exist"
//	@Router			/service_specs/{serviceType} [get]
func (ssc *ServiceSpecsController) GetServiceSpec(c *gin.Context) {
	svctype := c.MustGet("servicetype").(reflect.Type)

	specs, err := ssc.sSpecsServices.GetServiceSpecs(svctype)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, specs)
}

// InitializeServiceSpec returns an initialized service instance with default values.
//
//	@Summary	Initialize a new service instance.
//	@Schemes
//	@Description	Return an initialized service instance with default or custom values.
//	@Tags			service_specs
//	@Accept			json
//	@Produce		json
//	@Param			serviceType	path		string	true	"The service's type"
//	@Success		200			{object}	any
//	@Failure		404			{object}	happydns.ErrorResponse	"Service type does not exist"
//	@Failure		500			{object}	happydns.ErrorResponse	"Internal error"
//	@Router			/service_specs/{serviceType}/init [post]
func (ssc *ServiceSpecsController) InitializeServiceSpec(c *gin.Context) {
	svctype := c.MustGet("servicetype").(reflect.Type)

	initialized, err := ssc.sSpecsServices.InitializeService(svctype)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, initialized)
}
