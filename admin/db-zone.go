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

package admin

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/api"
	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/storage"
)

func declareZonesRoutes(opts *config.Options, router *gin.RouterGroup) {
	router.GET("/zones", getUserDomainZones)
	router.PUT("/zones", updateUserDomainZones)
	router.POST("/zones", newUserDomainZone)

	router.DELETE("/zones/:zoneid", deleteZone)

	apiZonesRoutes := router.Group("/zones/:zoneid")
	apiZonesRoutes.Use(api.ZoneHandler)

	apiZonesRoutes.GET("", api.GetZone)
	apiZonesRoutes.PUT("", updateZone)
	apiZonesRoutes.PATCH("", patchZoneService)

	apiZonesRoutes.GET("/*serviceid", getZoneService)
	apiZonesRoutes.PUT("/*serviceid", updateZoneService)
}

func getUserDomainZones(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	c.JSON(http.StatusOK, domain.ZoneHistory)
}

func updateUserDomainZones(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	err := c.ShouldBindJSON(&domain.ZoneHistory)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	ApiResponse(c, domain, storage.MainStore.UpdateDomain(domain))
}

func newUserDomainZone(c *gin.Context) {
	uz := &happydns.Zone{}
	err := c.ShouldBindJSON(&uz)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}
	uz.Id = nil

	ApiResponse(c, uz, storage.MainStore.CreateZone(uz))
}

func updateZone(c *gin.Context) {
	zone := c.MustGet("zone").(*happydns.Zone)

	uz := &happydns.Zone{}
	err := c.ShouldBindJSON(&uz)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}
	uz.Id = zone.Id

	ApiResponse(c, uz, storage.MainStore.UpdateZone(uz))
}

func getZoneService(c *gin.Context) {
	zone := c.MustGet("zone").(*happydns.Zone)

	serviceid, err := base64.StdEncoding.DecodeString(c.Param("serviceid")[1:])
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": err.Error()})
		return
	}

	_, svc := zone.FindService(serviceid)

	c.JSON(http.StatusOK, svc)
}

func updateZoneService(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)

	serviceid, err := base64.StdEncoding.DecodeString(c.Param("serviceid")[1:])
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": err.Error()})
		return
	}

	usc := &happydns.ServiceCombined{}
	err = c.ShouldBindJSON(&usc)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	err = zone.EraseService(usc.Domain, domain.DomainName, serviceid, usc)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	ApiResponse(c, zone.Services, storage.MainStore.UpdateZone(zone))
}

func patchZoneService(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)

	usc := &happydns.ServiceCombined{}
	err := c.ShouldBindJSON(&usc)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	err = zone.EraseService(usc.Domain, domain.DomainName, usc.Id, usc)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	ApiResponse(c, zone.Services, storage.MainStore.UpdateZone(zone))
}

func deleteZone(c *gin.Context) {
	zoneid, err := happydns.NewIdentifierFromString(c.Param("zoneid"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": err.Error()})
	} else {
		ApiResponse(c, true, storage.MainStore.DeleteZone(&happydns.Zone{ZoneMeta: happydns.ZoneMeta{Id: zoneid}}))
	}
}
