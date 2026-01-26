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
	"strings"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

type ZoneController struct {
	domainService         happydns.DomainUsecase
	zoneCorrectionService happydns.ZoneCorrectionApplierUsecase
	zoneService           happydns.ZoneUsecase
}

func NewZoneController(zoneService happydns.ZoneUsecase, domainService happydns.DomainUsecase, zoneCorrectionService happydns.ZoneCorrectionApplierUsecase) *ZoneController {
	return &ZoneController{
		domainService:         domainService,
		zoneCorrectionService: zoneCorrectionService,
		zoneService:           zoneService,
	}
}

// GetZone retrieves a zone's information.
//
//	@Summary	Retrieve a zone.
//	@Schemes
//	@Description	Retrieve information about a zone.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string	true	"Domain identifier"
//	@Param			zoneId		path		string	true	"Zone identifier"
//	@Success		200			{object}	happydns.Zone
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404			{object}	happydns.ErrorResponse	"Domain or Zone not found"
//	@Router			/domains/{domainId}/zone/{zoneId} [get]
func (zc *ZoneController) GetZone(c *gin.Context) {
	zone := c.MustGet("zone").(*happydns.Zone)

	c.JSON(http.StatusOK, zone)
}

// GetZoneSubdomain returns the services associated with a given subdomain.
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
//	@Success		200			{object}	[]happydns.Correction	"Differences, reported as text, one diff per item"
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
		corrections, _, err = zc.zoneCorrectionService.List(user, domain, newzone)
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

// ApplyZoneCorrections performs the requested changes with the provider.
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

	newZone, err := zc.zoneCorrectionService.Apply(user, domain, zone, &form)
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

// AddRecords adds a given record in the zone.
//
//	@Summary	Add a given record in the zone.
//	@Schemes
//	@Description	Add a given record in the zone.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string			true	"Domain identifier"
//	@Param			zoneId		path		string			true	"Zone identifier"
//	@Param			body		body		[]string		true	"Records to add as text, one record per line array"
//	@Success		200			{object}	happydns.Zone		"The updated zone"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404			{object}	happydns.ErrorResponse	"Domain or Zone not found"
//	@Router			/domains/{domainId}/zone/{zoneId}/records [post]
func (zc *ZoneController) AddRecords(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)

	var records []string
	err := c.ShouldBindJSON(&records)
	if err != nil {
		log.Printf("%s sends invalid JSON record: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	for _, record := range records {
		rr, err := helpers.ParseRecord(record, domain.DomainName)
		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}

		// Make record relative
		rr = helpers.RRRelative(rr, domain.DomainName)

		if strings.HasSuffix(rr.Header().Name, ".") {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Record %q is not part of the current domain: %s", rr.Header().String(), domain.DomainName)})
			return
		}

		err = zc.zoneService.AddRecord(zone, domain.DomainName, rr)
		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}
	}

	err = zc.zoneService.UpdateZone(zone.Id, func(z *happydns.Zone) {
		z.Services = zone.Services
	})
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, zone)
}

// DeleteRecords deletes a given record in the zone.
//
//	@Summary	Delete a given record in the zone.
//	@Schemes
//	@Description	Delete a given record in the zone.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string			true	"Domain identifier"
//	@Param			zoneId		path		string			true	"Zone identifier"
//	@Param			body		body		[]string		true	"Records to delete as text, one record per line array"
//	@Success		200			{object}	happydns.Zone		"The updated zone"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404			{object}	happydns.ErrorResponse	"Domain or Zone not found"
//	@Router			/domains/{domainId}/zone/{zoneId}/records/delete [post]
func (zc *ZoneController) DeleteRecords(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)

	var records []string
	err := c.ShouldBindJSON(&records)
	if err != nil {
		log.Printf("%s sends invalid JSON record: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	for _, record := range records {
		rr, err := helpers.ParseRecord(record, domain.DomainName)
		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}

		// Make record relative
		rr = helpers.RRRelative(rr, domain.DomainName)

		err = zc.zoneService.DeleteRecord(zone, domain.DomainName, rr)
		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}
	}

	err = zc.zoneService.UpdateZone(zone.Id, func(z *happydns.Zone) {
		z.Services = zone.Services
	})
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, zone)
}

// UpdateRecord updates a given record in the zone.
//
//	@Summary	Update a given record in the zone.
//	@Schemes
//	@Description	Update a given record in the zone.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string			true	"Domain identifier"
//	@Param			zoneId		path		string			true	"Zone identifier"
//	@Param			body		body		happydns.UpdateRecordForm	true	"Record to update as text"
//	@Success		200			{object}	happydns.Zone		"The updated zone"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404			{object}	happydns.ErrorResponse	"Domain or Zone not found"
//	@Router			/domains/{domainId}/zone/{zoneId}/records [patch]
func (zc *ZoneController) UpdateRecord(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)

	var form happydns.UpdateRecordForm
	err := c.ShouldBindJSON(&form)
	if err != nil {
		log.Printf("%s sends invalid JSON record: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	oldRecord, err := helpers.ParseRecord(form.OldRR, domain.DomainName)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("Unable to parse the record to update: %w", err))
		return
	}

	newRecord, err := helpers.ParseRecord(form.NewRR, domain.DomainName)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("Unable to parse the new record: %w", err))
		return
	}

	// Make record relative
	oldRecord = helpers.RRRelative(oldRecord, domain.DomainName)
	newRecord = helpers.RRRelative(newRecord, domain.DomainName)

	err = zc.zoneService.DeleteRecord(zone, domain.DomainName, oldRecord)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	err = zc.zoneService.AddRecord(zone, domain.DomainName, newRecord)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	err = zc.zoneService.UpdateZone(zone.Id, func(z *happydns.Zone) {
		z.Services = zone.Services
	})
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, zone)
}
