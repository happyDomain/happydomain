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

// DeleteZoneService removes a service from the given zone.
//
//	@Summary		Delete a service from zone
//	@Schemes
//	@Description	Remove the specified service from the zone.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Param			uid			path		string	false	"User ID or email"
//	@Param			pid			path		string	false	"Provider identifier"
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			zoneid		path		string	true	"Zone identifier"
//	@Param			serviceid	path		string	true	"Service identifier"
//	@Success		200			{object}	happydns.Zone
//	@Failure		404			{object}	happydns.ErrorResponse	"Service or zone not found"
//	@Failure		500			{object}	happydns.ErrorResponse	"Unable to remove service"
//	@Router			/domains/{domain}/zones/{zoneid}/services/{serviceid} [delete]
//	@Router			/users/{uid}/domains/{domain}/zones/{zoneid}/services/{serviceid} [delete]
//	@Router			/users/{uid}/providers/{pid}/domains/{domain}/zones/{zoneid}/services/{serviceid} [delete]
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

// GetZoneService retrieves a specific service from the zone.
//
//	@Summary		Get service information
//	@Schemes
//	@Description	Retrieve the specified service from the zone.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Param			uid			path		string	false	"User ID or email"
//	@Param			pid			path		string	false	"Provider identifier"
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			zoneid		path		string	true	"Zone identifier"
//	@Param			serviceid	path		string	true	"Service identifier"
//	@Success		200			{object}	happydns.Service
//	@Failure		404			{object}	happydns.ErrorResponse	"Service or zone not found"
//	@Router			/domains/{domain}/zones/{zoneid}/services/{serviceid} [get]
//	@Router			/users/{uid}/domains/{domain}/zones/{zoneid}/services/{serviceid} [get]
//	@Router			/users/{uid}/providers/{pid}/domains/{domain}/zones/{zoneid}/services/{serviceid} [get]
func (sc *ServiceController) GetZoneService(c *gin.Context) {
	zone := c.MustGet("zone").(*happydns.Zone)
	serviceid := c.MustGet("serviceid").(happydns.Identifier)

	_, svc := zone.FindService(serviceid)

	c.JSON(http.StatusOK, svc)
}

// UpdateZoneService updates an existing service in the zone.
//
//	@Summary		Update a service
//	@Schemes
//	@Description	Update the configuration of an existing service in the zone.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Param			uid			path		string					false	"User ID or email"
//	@Param			pid			path		string					false	"Provider identifier"
//	@Param			domain		path		string					true	"Domain identifier"
//	@Param			zoneid		path		string					true	"Zone identifier"
//	@Param			serviceid	path		string					true	"Service identifier"
//	@Param			body		body		happydns.Service	true	"Updated service object"
//	@Success		200			{object}	happydns.Zone
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid input or service ID mismatch"
//	@Failure		404			{object}	happydns.ErrorResponse	"Service or zone not found"
//	@Failure		500			{object}	happydns.ErrorResponse	"Unable to update service"
//	@Router			/domains/{domain}/zones/{zoneid}/services/{serviceid} [put]
//	@Router			/users/{uid}/domains/{domain}/zones/{zoneid}/services/{serviceid} [put]
//	@Router			/users/{uid}/providers/{pid}/domains/{domain}/zones/{zoneid}/services/{serviceid} [put]
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
