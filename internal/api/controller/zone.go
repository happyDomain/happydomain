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
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

type ZoneController struct {
	domainService happydns.DomainUsecase
	zoneService   happydns.ZoneUsecase
}

func NewZoneController(zoneService happydns.ZoneUsecase, domainService happydns.DomainUsecase) *ZoneController {
	return &ZoneController{
		domainService: domainService,
		zoneService:   zoneService,
	}
}

func (zc *ZoneController) GetZone(c *gin.Context) {
	zone := c.MustGet("zone").(*happydns.Zone)

	c.JSON(http.StatusOK, zone)
}

// getZoneSubdomain returns the services associated with a given subdomain.
//
//	@Summary	List services
//	@Schemes
//	@Description	Returns the services associated with the given subdomain.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string	true	"Domain identifier"
//	@Param			zoneId		path		string	true	"Zone identifier"
//	@Param			subdomain	path		string	true	"Part of the subdomain considered for the service (@ for the root of the zone ; subdomain is relative to the root, do not include it)"
//	@Success		200			{object}	happydns.ZoneServices
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404			{object}	happydns.ErrorResponse	"Domain or Zone not found"
//	@Router			/domains/{domainId}/zone/{zoneId}/{subdomain} [get]
func (zc *ZoneController) GetZoneSubdomain(c *gin.Context) {
	zone := c.MustGet("zone").(*happydns.Zone)
	subdomain := c.MustGet("subdomain").(happydns.Subdomain)

	c.JSON(http.StatusOK, happydns.ZoneServices{
		Services: zone.Services[subdomain],
	})
}

// DiffZones computes the difference between the two zone identifiers given.
//
//	@Summary	Compute differences between zones.
//	@Schemes
//	@Description	Compute the difference between the two zone identifiers given.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string			true	"Domain identifier"
//	@Param			zoneId		path		string			true	"Zone identifier to use as the new one."
//	@Param			oldZoneId		path		string			true	"Zone identifier to use as the old one. Currently only @ are expected, to use the currently deployed zone."
//	@Success		200			{object}	[]string		"Differences, reported as text, one diff per item"
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404			{object}	happydns.ErrorResponse	"Domain not found"
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Failure		501			{object}	happydns.ErrorResponse	"Diff between to zone identifier, currently not supported"
//	@Router			/domains/{domainId}/zone/{zoneId}/diff/{oldZoneId} [post]
func (zc *ZoneController) DiffZones(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	domain := c.MustGet("domain").(*happydns.Domain)
	newzone := c.MustGet("zone").(*happydns.Zone)

	var corrections []*happydns.Correction
	if c.Param("oldzoneid") == "@" {
		var err error
		corrections, err = zc.zoneService.GetZoneCorrections(user, domain, newzone)
		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}
	} else {
		oldzoneid, err := middleware.ParseZoneId(c, "oldzoneid")
		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}

		corrections, err = zc.zoneService.DiffZones(domain, newzone, oldzoneid)
		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}
	}

	c.JSON(http.StatusOK, corrections)
}

// applyZone performs the requested changes with the provider.
//
//	@Summary	Performs requested changes to the real zone.
//	@Schemes
//	@Description	Perform the requested changes with the provider.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string				true	"Domain identifier"
//	@Param			zoneId		path		string				true	"Zone identifier"
//	@Param			body		body		[]string			true	"Differences (from /diff_zones) to apply"
//	@Success		200			{object}	happydns.ZoneMeta	"The new Zone metadata containing the current zone"
//	@Failure		400			{object}	happydns.ErrorResponse		"Invalid input"
//	@Failure		401			{object}	happydns.ErrorResponse		"Authentication failure"
//	@Failure		404			{object}	happydns.ErrorResponse		"Domain or Zone not found"
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Router			/domains/{domainId}/zone/{zoneId}/apply_changes [post]
func (zc *ZoneController) ApplyZoneCorrections(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)

	var form happydns.ApplyZoneForm
	err := c.ShouldBindJSON(&form)
	if err != nil {
		log.Printf("%s sends invalid string array JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	newZone, err := zc.domainService.ApplyZoneCorrection(user, domain, zone, &form)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, newZone.ZoneMeta)
}

// ExportZone creates a flatten export of the zone.
//
//	@Summary	Get flatten zone file.
//	@Schemes
//	@Description	Create a flatten export of the zone that can be read as a BIND-like file.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string			true	"Domain identifier"
//	@Param			zoneId		path		string			true	"Zone identifier"
//	@Success		200			{object}	string			"The exported zone file (with initial and leading JSON quote)"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404			{object}	happydns.ErrorResponse	"Domain or Zone not found"
//	@Router			/domains/{domainId}/zone/{zoneId}/view [post]
func (zc *ZoneController) ExportZone(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)

	ret, err := zc.zoneService.FlattenZoneFile(domain, zone)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, ret)
}
