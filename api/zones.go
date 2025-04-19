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

package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/StackExchange/dnscontrol/v4/pkg/diff2"
	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
	"go.uber.org/multierr"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/storage"
	"git.happydns.org/happyDomain/utils"
)

func declareZonesRoutes(cfg *config.Options, router *gin.RouterGroup) {
	router.POST("/retrieve_zone", retrieveZone)
	router.POST("/diff_zones/:zoneid1/:zoneid2", diffZones)

	apiZonesRoutes := router.Group("/zone/:zoneid")
	apiZonesRoutes.Use(ZoneHandler)

	apiZonesRoutes.POST("/import", importZone)
	apiZonesRoutes.POST("/view", viewZone)
	apiZonesRoutes.POST("/apply_changes", applyZone)

	apiZonesRoutes.GET("", GetZone)
	apiZonesRoutes.PATCH("", UpdateZoneService)

	apiZonesSubdomainRoutes := apiZonesRoutes.Group("/:subdomain")
	apiZonesSubdomainRoutes.Use(subdomainHandler)
	apiZonesSubdomainRoutes.GET("", getZoneSubdomain)
	apiZonesSubdomainRoutes.POST("/services", addZoneService)

	declareServiceSettingsRoutes(cfg, apiZonesSubdomainRoutes)

	apiZonesSubdomainServiceIdRoutes := apiZonesSubdomainRoutes.Group("/services/:serviceid")
	apiZonesSubdomainServiceIdRoutes.Use(serviceIdHandler)
	apiZonesSubdomainServiceIdRoutes.GET("", getZoneService)
	apiZonesSubdomainServiceIdRoutes.DELETE("", deleteZoneService)
	apiZonesSubdomainServiceIdRoutes.GET("/records", getServiceRecords)
}

func loadZoneFromId(domain *happydns.Domain, id string) (*happydns.Zone, int, error) {
	zoneid, err := happydns.NewIdentifierFromString(id)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("Invalid zoneid: %q", id)
	}

	// Check that the zoneid exists in the domain history
	if !domain.HasZone(zoneid) {
		return nil, http.StatusNotFound, fmt.Errorf("Zone not found: %q", id)
	}

	zone, err := storage.MainStore.GetZone(zoneid)
	if err != nil {
		log.Printf("An error occurs when trying to retrieve user zone (id=%s): %s", id, err.Error())
		return nil, http.StatusNotFound, fmt.Errorf("Zone not found: %q", id)
	}

	return zone, http.StatusOK, nil
}

func ZoneHandler(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	zone, statuscode, err := loadZoneFromId(domain, c.Param("zoneid"))
	if err != nil {
		c.AbortWithStatusJSON(statuscode, gin.H{"errmsg": err.Error()})
		return
	}

	c.Set("zone", zone)

	c.Next()
}

func GetZone(c *gin.Context) {
	zone := c.MustGet("zone").(*happydns.Zone)

	c.JSON(http.StatusOK, zone)
}

func subdomainHandler(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	subdomain := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(c.Param("subdomain"), "."+domain.DomainName), "@"), domain.DomainName)

	c.Set("subdomain", subdomain)

	c.Next()
}

type zoneServices struct {
	Services []*happydns.ServiceCombined `json:"services"`
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
//	@Success		200			{object}	zoneServices
//	@Failure		401			{object}	happydns.Error	"Authentication failure"
//	@Failure		404			{object}	happydns.Error	"Domain or Zone not found"
//	@Router			/domains/{domainId}/zone/{zoneId}/{subdomain} [get]
func getZoneSubdomain(c *gin.Context) {
	zone := c.MustGet("zone").(*happydns.Zone)
	subdomain := c.MustGet("subdomain").(string)

	c.JSON(http.StatusOK, zoneServices{
		Services: zone.Services[subdomain],
	})
}

// addZoneService adds a Service to the given subdomain of the Zone.
//
//	@Summary	Add a Service.
//	@Schemes
//	@Description	Add a Service to the given subdomain of the Zone.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string	true	"Domain identifier"
//	@Param			zoneId		path		string	true	"Zone identifier"
//	@Param			subdomain	path		string	true	"Part of the subdomain considered for the service (@ for the root of the zone ; subdomain is relative to the root, do not include it)"
//	@Success		200			{object}	happydns.Zone
//	@Failure		400			{object}	happydns.Error	"Invalid input"
//	@Failure		401			{object}	happydns.Error	"Authentication failure"
//	@Failure		404			{object}	happydns.Error	"Domain or Zone not found"
//	@Router			/domains/{domainId}/zone/{zoneId}/{subdomain}/services [post]
func addZoneService(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)
	subdomain := c.MustGet("subdomain").(string)

	usc := &happydns.ServiceCombined{}
	err := c.ShouldBindJSON(&usc)
	if err != nil {
		log.Printf("%s sends invalid service JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if usc.Service == nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Unable to parse the given service."})
		return
	}

	err = zone.AppendService(subdomain, domain.DomainName, usc)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to add service: %s", err.Error())})
		return
	}

	err = storage.MainStore.UpdateZone(zone)
	if err != nil {
		log.Printf("%s: Unable to UpdateZone in updateZoneService: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your zone. Please retry later."})
		return
	}

	c.JSON(http.StatusOK, zone)
}

func serviceIdHandler(c *gin.Context) {
	serviceid, err := happydns.NewIdentifierFromString(c.Param("serviceid"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Bad service identifier: %s", err.Error())})
		return
	}

	c.Set("serviceid", serviceid)

	c.Next()
}

// getServiceService retrieves the designated Service.
//
//	@Summary	Get the Service.
//	@Schemes
//	@Description	Retrieve the designated Service.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string	true	"Domain identifier"
//	@Param			zoneId		path		string	true	"Zone identifier"
//	@Param			subdomain	path		string	true	"Part of the subdomain considered for the service (@ for the root of the zone ; subdomain is relative to the root, do not include it)"
//	@Param			serviceId	path		string	true	"Service identifier"
//	@Success		200			{object}	happydns.ServiceCombined
//	@Failure		401			{object}	happydns.Error	"Authentication failure"
//	@Failure		404			{object}	happydns.Error	"Domain or Zone not found"
//	@Router			/domains/{domainId}/zone/{zoneId}/{subdomain}/services/{serviceId} [get]
func getZoneService(c *gin.Context) {
	zone := c.MustGet("zone").(*happydns.Zone)
	serviceid := c.MustGet("serviceid").(happydns.Identifier)
	subdomain := c.MustGet("subdomain").(string)

	c.JSON(http.StatusOK, zone.FindSubdomainService(subdomain, serviceid))
}

// retrieveZone retrieves the current zone deployed on the NS Provider.
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
//	@Failure		401			{object}	happydns.Error		"Authentication failure"
//	@Failure		404			{object}	happydns.Error		"Domain not found"
//	@Router			/domains/{domainId}/retrieve_zone [post]
func retrieveZone(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	domain := c.MustGet("domain").(*happydns.Domain)

	p, err := storage.MainStore.GetProvider(user, domain.IdProvider)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": fmt.Sprintf("Unable to find your provider: %s", err.Error())})
		return
	}

	provider, err := p.ParseProvider()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": fmt.Sprintf("Unable to retrieve provider's data: %s", err.Error())})
		return
	}

	zone, err := provider.ImportZone(domain)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to synchronize your zone: %s", err.Error())})
		return
	}

	services, defaultTTL, err := svcs.AnalyzeZone(domain.DomainName, zone)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to perform the analysis of your zone: %s", err.Error())})
		return
	}

	myZone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{
			IdAuthor:     domain.IdUser,
			DefaultTTL:   defaultTTL,
			LastModified: time.Now(),
		},
		Services: services,
	}

	// Create history zone
	err = storage.MainStore.CreateZone(myZone)
	if err != nil {
		log.Printf("%s: unable to CreateZone in importZone: %s\n", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are unable to create your zone."})
		return
	}
	domain.ZoneHistory = append(
		[]happydns.Identifier{myZone.Id}, domain.ZoneHistory...)

	// Create wip zone
	err = storage.MainStore.CreateZone(myZone)
	if err != nil {
		log.Printf("%s: unable to CreateZone2 in importZone: %s\n", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are unable to create your zone."})
		return
	}
	domain.ZoneHistory = append(
		[]happydns.Identifier{myZone.Id}, domain.ZoneHistory...)

	err = storage.MainStore.UpdateDomain(domain)
	if err != nil {
		log.Printf("%s: unable to UpdateDomain in importZone: %s\n", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are unable to create your zone."})
		return
	}

	storage.MainStore.CreateDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_INFO, fmt.Sprintf("Zone imported from provider API: %s", myZone.Id.String())))

	c.JSON(http.StatusOK, &myZone.ZoneMeta)
}

// importZone takes a bind style file
func importZone(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)

	fd, _, err := c.Request.FormFile("zone")
	if err != nil {
		log.Printf("Error when retrieving zone file from %s: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Unable to read your zone file: something is wrong in your request"})
		return
	}
	defer fd.Close()

	zp := dns.NewZoneParser(fd, domain.DomainName, "")

	var rrs []dns.RR
	for rr, ok := zp.Next(); ok; rr, ok = zp.Next() {
		rrs = append(rrs, rr)
	}

	rcs, err := utils.RRstoRCs(rrs, strings.TrimSuffix(domain.DomainName, "."))
	if err != nil {
		log.Printf("Error when converting RRs to RCs of %s: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to read your zone file: %s", err.Error())})
		return
	}

	zone.Services, _, err = svcs.AnalyzeZone(domain.DomainName, rcs)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	zone.LastModified = time.Now()

	err = storage.MainStore.UpdateZone(zone)
	if err != nil {
		log.Printf("%s: Unable to UpdateZone in updateZoneService: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your zone. Please retry later."})
		return
	}

	storage.MainStore.CreateDomainLog(domain, happydns.NewDomainLog(c.MustGet("LoggedUser").(*happydns.User), happydns.LOG_INFO, fmt.Sprintf("Zone imported from Bind-style file: %s", zone.Id.String())))

	c.JSON(http.StatusOK, zone)
}

// diffZones computes the difference between the two zone identifiers given.
//
//	@Summary	Compute differences between zones.
//	@Schemes
//	@Description	Compute the difference between the two zone identifiers given.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string			true	"Domain identifier"
//	@Param			zoneId1		path		string			true	"Zone identifier to use as the old one. Currently only @ are expected, to use the currently deployed zone."
//	@Param			zoneId2		path		string			true	"Zone identifier to use as the new one"
//	@Success		200			{object}	[]string		"Differences, reported as text, one diff per item"
//	@Failure		400			{object}	happydns.Error	"Invalid input"
//	@Failure		401			{object}	happydns.Error	"Authentication failure"
//	@Failure		404			{object}	happydns.Error	"Domain not found"
//	@Failure		500			{object}	happydns.Error
//	@Failure		501			{object}	happydns.Error	"Diff between to zone identifier, currently not supported"
//	@Router			/domains/{domainId}/diff_zones/{zoneId1}/{zoneId2} [post]
func diffZones(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	domain := c.MustGet("domain").(*happydns.Domain)

	zone2, statuscode, err := loadZoneFromId(domain, c.Param("zoneid2"))
	if err != nil {
		c.AbortWithStatusJSON(statuscode, gin.H{"errmsg": err.Error()})
		return
	}

	records2, err := zone2.GenerateRecords(domain.DomainName)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	dc2 := &models.DomainConfig{
		Name:    strings.TrimSuffix(domain.DomainName, "."),
		Records: records2,
	}

	if c.Param("zoneid1") == "@" {
		p, err := storage.MainStore.GetProvider(user, domain.IdProvider)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": fmt.Errorf("Unable to find the given provider: %q for %q", domain.IdProvider, domain.DomainName)})
			return
		}

		provider, err := p.ParseProvider()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": fmt.Errorf("Unable to retrieve provider's data: %s", err.Error())})
			return
		}

		corrections, err := provider.GetDomainCorrections(domain, dc2)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
			return
		}

		var rrCorected []string
		for _, c := range corrections {
			rrCorected = append(rrCorected, c.Msg)
		}

		c.JSON(http.StatusOK, rrCorected)
	} else {
		zone1, statuscode, err := loadZoneFromId(domain, c.Param("zoneid1"))
		if err != nil {
			c.AbortWithStatusJSON(statuscode, gin.H{"errmsg": err.Error()})
			return
		}

		records1, err := zone1.GenerateRecords(domain.DomainName)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
			return
		}

		dc1 := &models.DomainConfig{
			Name:    strings.TrimSuffix(domain.DomainName, "."),
			Records: records1,
		}

		corrections, err := diff2.ByRecord(records2, dc1, nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
			return
		}

		var rrCorected []string
		for _, c := range corrections {
			rrCorected = append(rrCorected, c.Msgs...)
		}

		c.JSON(http.StatusOK, rrCorected)
	}
}

type applyZoneForm struct {
	WantedCorrections []string `json:"wantedCorrections"`
	CommitMsg         string   `json:"commitMessage"`
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
//	@Failure		400			{object}	happydns.Error		"Invalid input"
//	@Failure		401			{object}	happydns.Error		"Authentication failure"
//	@Failure		404			{object}	happydns.Error		"Domain or Zone not found"
//	@Failure		500			{object}	happydns.Error
//	@Router			/domains/{domainId}/zone/{zoneId}/apply_changes [post]
func applyZone(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)

	p, err := storage.MainStore.GetProvider(user, domain.IdProvider)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, fmt.Errorf("Unable to find the given provider: %q for %q", domain.IdProvider, domain.DomainName))
		return
	}

	provider, err := p.ParseProvider()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, fmt.Errorf("Unable to retrieve the provider data: %s", err.Error()))
		return
	}

	records, err := zone.GenerateRecords(domain.DomainName)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, err.Error())
		return
	}

	dc := &models.DomainConfig{
		Name:    strings.TrimSuffix(domain.DomainName, "."),
		Records: records,
	}

	var form applyZoneForm
	err = c.ShouldBindJSON(&form)
	if err != nil {
		log.Printf("%s sends invalid string array JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	nbcorrections := len(form.WantedCorrections)
	corrections, err := provider.GetDomainCorrections(domain, dc)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": fmt.Sprintf("Unable to compute domain corrections: %s", err.Error())})
		return
	}

	var errs error
corrections:
	for i, cr := range corrections {
		for ic, wc := range form.WantedCorrections {
			if wc == cr.Msg {
				log.Printf("%s: apply correction: %s", domain.DomainName, cr.Msg)
				err := cr.F()

				if err != nil {
					log.Printf("%s: unable to apply correction: %s", domain.DomainName, err.Error())
					storage.MainStore.CreateDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("Failed record update (%s): %s", cr.Msg, err.Error())))
					errs = multierr.Append(errs, fmt.Errorf("%s: %w", cr.Msg, err))

					// Stop the zone update if we didn't change it yet
					if i == 0 {
						break corrections
					}
				} else {
					form.WantedCorrections = append(form.WantedCorrections[:ic], form.WantedCorrections[ic+1:]...)
				}
				break
			}
		}
	}

	if len(multierr.Errors(errs)) > 0 {
		storage.MainStore.CreateDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("Failed zone publishing (%s): %d corrections were not applied due to errors.", zone.Id.String(), nbcorrections)))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to update the zone: %s", err.Error())})
		return
	} else if len(form.WantedCorrections) > 0 {
		storage.MainStore.CreateDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("Failed zone publishing (%s): %d corrections were not applied.", zone.Id.String(), nbcorrections)))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to perform the following changes: %s", form.WantedCorrections)})
		return
	}

	storage.MainStore.CreateDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ACK, fmt.Sprintf("Zone published (%s), %d corrections applied with success", zone.Id.String(), nbcorrections)))

	// Create a new zone in history for futher updates
	newZone := zone.DerivateNew()
	err = storage.MainStore.CreateZone(newZone)
	if err != nil {
		log.Printf("%s was unable to CreateZone: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are unable to create the zone now."})
		return
	}

	domain.ZoneHistory = append(
		[]happydns.Identifier{newZone.Id}, domain.ZoneHistory...)

	err = storage.MainStore.UpdateDomain(domain)
	if err != nil {
		log.Printf("%s was unable to UpdateDomain: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are unable to create the zone now."})
		return
	}

	// Commit changes in previous zone
	now := time.Now()
	zone.ZoneMeta.IdAuthor = user.Id
	zone.CommitMsg = &form.CommitMsg
	zone.ZoneMeta.CommitDate = &now
	zone.ZoneMeta.Published = &now

	zone.LastModified = time.Now()

	err = storage.MainStore.UpdateZone(zone)
	if err != nil {
		log.Printf("%s was unable to UpdateZone: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are unable to create the zone now."})
		return
	}

	c.JSON(http.StatusOK, newZone.ZoneMeta)
}

// viewZone creates a flatten export of the zone.
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
//	@Failure		401			{object}	happydns.Error	"Authentication failure"
//	@Failure		404			{object}	happydns.Error	"Domain or Zone not found"
//	@Router			/domains/{domainId}/zone/{zoneId}/view [post]
func viewZone(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)

	var ret string

	rrs, err := zone.GenerateRecords(domain.DomainName)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": fmt.Sprintf("An error occurs during zone records generation: %s.", err.Error())})
		return
	}

	for _, rc := range rrs {
		if _, ok := dns.StringToType[rc.Type]; ok {
			ret += rc.ToRR().String() + "\n"
		} else {
			ret += fmt.Sprintf("%s %d IN %s %s\n", rc.NameFQDN, rc.TTL, rc.Type, rc.String())
		}
	}

	c.JSON(http.StatusOK, ret)
}

// UpdateZoneService adds or updates a service inside the given Zone.
//
//	@Summary	Add or update a Service.
//	@Schemes
//	@Description	Add or update a Service inside the given Zone.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string						true	"Domain identifier"
//	@Param			zoneId		path		string						true	"Zone identifier"
//	@Param			body		body		happydns.ServiceCombined	true	"Service to update"
//	@Success		200			{object}	happydns.Zone
//	@Failure		401			{object}	happydns.Error	"Authentication failure"
//	@Failure		404			{object}	happydns.Error	"Domain or Zone not found"
//	@Router			/domains/{domainId}/zone/{zoneId} [patch]
func UpdateZoneService(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)

	usc := &happydns.ServiceCombined{}
	err := c.ShouldBindJSON(&usc)
	if err != nil {
		log.Printf("%s sends invalid domain JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	err = zone.EraseService(usc.Domain, domain.DomainName, usc.Id, usc)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to delete service: %s", err.Error())})
		return
	}

	zone.LastModified = time.Now()

	err = storage.MainStore.UpdateZone(zone)
	if err != nil {
		log.Printf("%s: Unable to UpdateZone in updateZoneService: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your zone. Please retry later."})
		return
	}

	c.JSON(http.StatusOK, zone)
}

// deleteZoneService drops the given Service.
//
//	@Summary	Drop the given Service.
//	@Schemes
//	@Description	Drop the given Service.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string	true	"Domain identifier"
//	@Param			zoneId		path		string	true	"Zone identifier"
//	@Param			subdomain	path		string	true	"Part of the subdomain considered for the service (@ for the root of the zone ; subdomain is relative to the root, do not include it)"
//	@Param			serviceId	path		string	true	"Service identifier"
//	@Success		200			{object}	happydns.Zone
//	@Failure		400			{object}	happydns.Error	"Invalid input"
//	@Failure		401			{object}	happydns.Error	"Authentication failure"
//	@Failure		404			{object}	happydns.Error	"Domain or Zone not found"
//	@Router			/domains/{domainId}/zone/{zoneId}/{subdomain}/services/{serviceId} [delete]
func deleteZoneService(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)
	serviceid := c.MustGet("serviceid").(happydns.Identifier)
	subdomain := c.MustGet("subdomain").(string)

	err := zone.EraseService(subdomain, domain.DomainName, serviceid, nil)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to delete service: %s", err.Error())})
		return
	}

	zone.LastModified = time.Now()

	err = storage.MainStore.UpdateZone(zone)
	if err != nil {
		log.Printf("%s: Unable to UpdateZone in deleteZoneService: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your zone. Please retry later."})
		return
	}

	c.JSON(http.StatusOK, zone)
}

type serviceRecord struct {
	Type   string               `json:"type"`
	String string               `json:"str"`
	Fields *models.RecordConfig `json:"fields,omitempty"`
	RR     dns.RR               `json:"rr,omitempty"`
}

// getServiceRecords retrieves the records that will be generated by a Service.
//
//	@Summary	Get the records for a Service.
//	@Schemes
//	@Description	Retrieve the records that will be generated by a Service.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domainId	path		string	true	"Domain identifier"
//	@Param			zoneId		path		string	true	"Zone identifier"
//	@Param			subdomain	path		string	true	"Part of the subdomain considered for the service (@ for the root of the zone ; subdomain is relative to the root, do not include it)"
//	@Param			serviceId	path		string	true	"Service identifier"
//	@Success		200			{object}	happydns.Zone
//	@Failure		401			{object}	happydns.Error	"Authentication failure"
//	@Failure		404			{object}	happydns.Error	"Domain or Zone not found"
//	@Router			/domains/{domainId}/zone/{zoneId}/{subdomain}/services/{serviceId}/records [get]
func getServiceRecords(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)
	serviceid := c.MustGet("serviceid").(happydns.Identifier)
	subdomain := c.MustGet("subdomain").(string)

	svc := zone.FindSubdomainService(subdomain, serviceid)
	if svc == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Service not found."})
		return
	}

	ttl := svc.Ttl
	if ttl == 0 {
		ttl = zone.DefaultTTL
	}

	rrs, err := svc.GetRecords(subdomain, ttl, domain.DomainName)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": err.Error()})
		return
	}

	var ret []serviceRecord
	for _, rr := range rrs {
		ret = append(ret, serviceRecord{
			Type:   dns.Type(rr.Header().Rrtype).String(),
			String: rr.String(),
			RR:     rr,
		})
	}

	c.JSON(http.StatusOK, ret)
}
