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
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

type DomainController struct {
	domainService      happydns.DomainUsecase
	remoteZoneImporter happydns.RemoteZoneImporterUsecase
	zoneImporter       happydns.ZoneImporterUsecase
}

func NewDomainController(domainService happydns.DomainUsecase, remoteZoneImporter happydns.RemoteZoneImporterUsecase, zoneImporter happydns.ZoneImporterUsecase) *DomainController {
	return &DomainController{
		domainService:      domainService,
		remoteZoneImporter: remoteZoneImporter,
		zoneImporter:       zoneImporter,
	}
}

// GetDomains retrieves all domains belonging to the user.
//
//	@Summary	Retrieve user's domains
//	@Schemes
//	@Description	Retrieve all domains belonging to the user.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{array}		happydns.Domain
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Unable to retrieve user's domains"
//	@Router			/domains [get]
func (dc *DomainController) GetDomains(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined"})
		return
	}

	domains, err := dc.domainService.ListUserDomains(user)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, domains)
}

// addDomain appends a new domain to those managed.
//
//	@Summary	Manage a new domain
//	@Schemes
//	@Description	Append a new domain to those managed.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			body	body	happydns.DomainCreationInput	true	"Domain object that you want to manage through happyDomain."
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Domain
//	@Failure		400	{object}	happydns.ErrorResponse	"Error in received data"
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		500	{object}	happydns.ErrorResponse	"Unable to retrieve current user's domains"
//	@Router			/domains [post]
func (dc *DomainController) AddDomain(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("User not defined."))
		return
	}

	var uz happydns.Domain
	err := c.ShouldBindJSON(&uz)
	if err != nil {
		log.Printf("%s sends invalid Domain JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	err = dc.domainService.CreateDomain(user, &uz)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, uz)
}

// GetDomain retrieves information about a given Domain owned by the user.
//
//	@Summary	Retrieve Domain local information.
//	@Schemes
//	@Description	Retrieve information in the database about a given Domain owned by the user.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			domainId	path	string	true	"Domain identifier"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.DomainWithZoneMetadata
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Domain not found"
//	@Router			/domains/{domainId} [get]
func (dc *DomainController) GetDomain(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	domainExtended, err := dc.domainService.ExtendsDomainWithZoneMeta(domain)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, domainExtended)
}

// UpdateDomain updates the information about a given Domain owned by the user.
//
//	@Summary	Update Domain local information.
//	@Schemes
//	@Description	Updates the information about a given Domain owned by the user.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			domainId	path	string			true	"Domain identifier"
//	@Param			body		body	happydns.Domain	true	"The new object overriding the current domain"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Domain
//	@Failure		400	{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		400	{object}	happydns.ErrorResponse	"Identifier changed"
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Domain not found"
//	@Failure		500	{object}	happydns.ErrorResponse	"Database writing error"
//	@Router			/domains/{domainId} [put]
func (dc *DomainController) UpdateDomain(c *gin.Context) {
	old := c.MustGet("domain").(*happydns.Domain)
	user := middleware.MyUser(c)
	if user == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("User not defined."))
		return
	}

	var domain happydns.Domain
	err := c.ShouldBindJSON(&domain)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	err = dc.domainService.UpdateDomain(old.Id, user, func(new *happydns.Domain) {
		new.Group = domain.Group
	})
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, old)
}

// delDomain removes a domain from the database.
//
//	@Summary	Stop managing a Domain.
//	@Schemes
//	@Description	Delete all the information in the database about the given Domain. This only stops happyDomain from managing the Domain, it doesn't do anything on the Provider.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			domainId	path	string	true	"Domain identifier"
//	@Security		securitydefinitions.basic
//	@Success		204	"Domain deleted"
//	@Failure		400	{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Domain not found"
//	@Failure		500	{object}	happydns.ErrorResponse	"Database writing error"
//	@Router			/domains/{domainId} [delete]
func (dc *DomainController) DelDomain(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	if err := dc.domainService.DeleteDomain(domain.Id); err != nil {
		log.Printf("%s was unable to DeleteDomain: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": fmt.Sprintf("Unable to delete your domain: %s", err.Error())})
		return
	}

	c.Status(http.StatusNoContent)
}

// RetrieveZone retrieves the current zone deployed on the NS Provider.
//
//	@Summary	Retrieve the zone on the Provider.
//	@Schemes
//	@Description	Retrieve the current zone deployed on the NS Provider.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string				true	"Domain identifier"
//	@Success		200			{object}	happydns.ZoneMeta	"The new zone metadata"
//	@Failure		401			{object}	happydns.ErrorResponse		"Authentication failure"
//	@Failure		404			{object}	happydns.ErrorResponse		"Domain not found"
//	@Router			/domains/{domainId}/retrieve_zone [post]
func (dc *DomainController) RetrieveZone(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("User not defined."))
		return
	}
	domain := c.MustGet("domain").(*happydns.Domain)

	zone, err := dc.remoteZoneImporter.Import(user, domain)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, zone.Meta())
}

// ImportZone takes a bind style file
func (dc *DomainController) ImportZone(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("User not defined."))
		return
	}
	domain := c.MustGet("domain").(*happydns.Domain)

	fd, _, err := c.Request.FormFile("zone")
	if err != nil {
		log.Printf("Error when retrieving zone file from %s: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Unable to read your zone file: something is wrong in your request"})
		return
	}
	defer fd.Close()

	zp := dns.NewZoneParser(fd, domain.Domain, "")

	var rrs []happydns.Record
	for rr, ok := zp.Next(); ok; rr, ok = zp.Next() {
		rrs = append(rrs, rr)
	}

	zone, err := dc.zoneImporter.Import(user, domain, rrs)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, zone)
}
