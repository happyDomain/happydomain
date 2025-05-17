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
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
)

type ServiceController struct {
	serviceService happydns.ServiceUsecase
	zoneServiceUC  happydns.ZoneServiceUsecase
}

func NewServiceController(serviceService happydns.ServiceUsecase, zoneServiceUC happydns.ZoneServiceUsecase) *ServiceController {
	return &ServiceController{
		serviceService,
		zoneServiceUC,
	}
}

func (sc *ServiceController) DeleteZoneService(c *gin.Context) {
	user := middleware.MyUser(c)
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)
	serviceid := c.MustGet("serviceid").(happydns.Identifier)

	subdomain, svc := zone.FindService(serviceid)
	if svc == nil {
		middleware.ErrorResponse(c, http.StatusNotFound, fmt.Errorf("Service not found"))
		return
	}

	zone, err := sc.zoneServiceUC.RemoveServiceFromZone(user, domain, zone, subdomain, serviceid)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, zone)
}

func (sc *ServiceController) GetZoneService(c *gin.Context) {
	zone := c.MustGet("zone").(*happydns.Zone)
	serviceid := c.MustGet("serviceid").(happydns.Identifier)

	_, svc := zone.FindService(serviceid)

	c.JSON(http.StatusOK, svc)
}

func (sc *ServiceController) UpdateZoneService(c *gin.Context) {
	user := middleware.MyUser(c)
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)
	serviceid := c.MustGet("serviceid").(happydns.Identifier)

	var usc happydns.ServiceMessage
	err := c.ShouldBindJSON(&usc)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	newservice, err := serviceUC.ParseService(&usc)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	if !serviceid.Equals(usc.Id) {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("Uploaded service identifier doesn't match selected service identifier in route."))
		return
	}

	zone, err = sc.zoneServiceUC.UpdateZoneService(user, domain, zone, happydns.Subdomain(usc.Domain), usc.Id, newservice)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, zone)
}
