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

	"git.happydns.org/happyDomain/internal/api/controller"
	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

type ZoneController struct {
	domainService         happydns.DomainUsecase
	zoneService           happydns.ZoneUsecase
	zoneCorrectionService happydns.ZoneCorrectionApplierUsecase
	store                 zone.ZoneStorage
}

func NewZoneController(domainService happydns.DomainUsecase, zoneService happydns.ZoneUsecase, zoneCorrectionService happydns.ZoneCorrectionApplierUsecase, store zone.ZoneStorage) *ZoneController {
	return &ZoneController{
		domainService,
		zoneService,
		zoneCorrectionService,
		store,
	}
}

// addZone creates a new zone in the system.
//
//	@Summary		Create a new zone.
//	@Schemes
//	@Description	Create a new zone with the provided configuration.
//	@Tags			admin-zones
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					false	"User ID or email"
//	@Param			pid		path		string					false	"Provider identifier"
//	@Param			domain	path		string					true	"Domain identifier"
//	@Param			body	body		happydns.Zone			true	"Zone configuration"
//	@Success		200		{object}	happydns.Zone
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Router			/domains/{domain}/zones [post]
//	@Router			/users/{uid}/domains/{domain}/zones [post]
//	@Router			/users/{uid}/providers/{pid}/domains/{domain}/zones [post]
func (zc *ZoneController) AddZone(c *gin.Context) {
	uz := &happydns.Zone{}
	err := c.ShouldBindJSON(&uz)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("something is wrong in received data: %w", err))
		return
	}
	uz.Id = nil

	happydns.ApiResponse(c, uz, zc.store.CreateZone(uz))
}

// deleteZone deletes a zone from the system.
//
//	@Summary		Delete a zone.
//	@Schemes
//	@Description	Delete a zone by its identifier.
//	@Tags			admin-zones
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					false	"User ID or email"
//	@Param			pid		path		string					false	"Provider identifier"
//	@Param			domain	path		string					true	"Domain identifier"
//	@Param			zoneid	path		string					true	"Zone identifier"
//	@Success		200		{object}	bool
//	@Failure		404		{object}	happydns.ErrorResponse	"Zone not found"
//	@Router			/domains/{domain}/zones/{zoneid} [delete]
//	@Router			/users/{uid}/domains/{domain}/zones/{zoneid} [delete]
//	@Router			/users/{uid}/providers/{pid}/domains/{domain}/zones/{zoneid} [delete]
func (zc *ZoneController) DeleteZone(c *gin.Context) {
	zoneid, err := happydns.NewIdentifierFromString(c.Param("zoneid"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	happydns.ApiResponse(c, true, zc.store.DeleteZone(zoneid))
}

// getZone retrieves a zone's information.
//
//	@Summary		Retrieve a zone.
//	@Schemes
//	@Description	Retrieve information about a zone by its identifier.
//	@Tags			admin-zones
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					false	"User ID or email"
//	@Param			pid		path		string					false	"Provider identifier"
//	@Param			domain	path		string					true	"Domain identifier"
//	@Param			zoneid	path		string					true	"Zone identifier"
//	@Success		200		{object}	happydns.Zone
//	@Failure		404		{object}	happydns.ErrorResponse	"Zone not found"
//	@Router			/domains/{domain}/zones/{zoneid} [get]
//	@Router			/users/{uid}/domains/{domain}/zones/{zoneid} [get]
//	@Router			/users/{uid}/providers/{pid}/domains/{domain}/zones/{zoneid} [get]
func (zc *ZoneController) GetZone(c *gin.Context) {
	apizc := controller.NewZoneController(zc.zoneService, zc.domainService, zc.zoneCorrectionService)
	apizc.GetZone(c)
}

// listZones lists all zones for a domain.
//
//	@Summary		List all zones.
//	@Schemes
//	@Description	Retrieve the list of all zones (zone history) for a domain.
//	@Tags			admin-zones
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					false	"User ID or email"
//	@Param			pid		path		string					false	"Provider identifier"
//	@Param			domain	path		string					true	"Domain identifier"
//	@Success		200	{array}	happydns.Identifier	"List of zone identifiers from zone history"
//	@Router			/domains/{domain}/zones [get]
//	@Router			/users/{uid}/domains/{domain}/zones [get]
//	@Router			/users/{uid}/providers/{pid}/domains/{domain}/zones [get]
func (zc *ZoneController) ListZones(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	c.JSON(http.StatusOK, domain.ZoneHistory)
}

// updateZone updates an existing zone.
//
//	@Summary		Update a zone.
//	@Schemes
//	@Description	Update an existing zone with new configuration.
//	@Tags			admin-zones
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					false	"User ID or email"
//	@Param			pid		path		string					false	"Provider identifier"
//	@Param			domain	path		string					true	"Domain identifier"
//	@Param			zoneid	path		string					true	"Zone identifier"
//	@Param			body	body		happydns.Zone			true	"Updated zone configuration"
//	@Success		200		{object}	happydns.Zone
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		404		{object}	happydns.ErrorResponse	"Zone not found"
//	@Router			/domains/{domain}/zones/{zoneid} [put]
//	@Router			/users/{uid}/domains/{domain}/zones/{zoneid} [put]
//	@Router			/users/{uid}/providers/{pid}/domains/{domain}/zones/{zoneid} [put]
func (zc *ZoneController) UpdateZone(c *gin.Context) {
	zone := c.MustGet("zone").(*happydns.Zone)

	uz := &happydns.Zone{}
	err := c.ShouldBindJSON(&uz)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, fmt.Errorf("something is wrong in received data: %w", err))
		return
	}
	uz.Id = zone.Id

	happydns.ApiResponse(c, uz, zc.store.UpdateZone(uz))
}
