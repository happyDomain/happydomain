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

	"git.happydns.org/happyDomain/api/controller"
	"git.happydns.org/happyDomain/api/middleware"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type ZoneController struct {
	domainService happydns.DomainUsecase
	zoneService   happydns.ZoneUsecase
	store         storage.Storage
}

func NewZoneController(domainService happydns.DomainUsecase, zoneService happydns.ZoneUsecase, store storage.Storage) *ZoneController {
	return &ZoneController{
		domainService,
		zoneService,
		store,
	}
}

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

func (zc *ZoneController) DeleteZone(c *gin.Context) {
	zoneid, err := happydns.NewIdentifierFromString(c.Param("zoneid"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	happydns.ApiResponse(c, true, zc.store.DeleteZone(zoneid))
}

func (zc *ZoneController) GetZone(c *gin.Context) {
	apizc := controller.NewZoneController(zc.zoneService, zc.domainService)
	apizc.GetZone(c)
}

func (zc *ZoneController) ListZones(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	c.JSON(http.StatusOK, domain.ZoneHistory)
}

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

func (zc *ZoneController) UpdateZones(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	err := c.ShouldBindJSON(&domain.ZoneHistory)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, fmt.Errorf("something is wrong in received data: %w", err))
		return
	}

	happydns.ApiResponse(c, domain, zc.store.UpdateDomain(domain))
}
