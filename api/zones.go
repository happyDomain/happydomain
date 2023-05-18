// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/StackExchange/dnscontrol/v3/models"
	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"

	"git.happydns.org/happydomain/config"
	"git.happydns.org/happydomain/model"
	"git.happydns.org/happydomain/services"
	"git.happydns.org/happydomain/storage"
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

func getZoneSubdomain(c *gin.Context) {
	zone := c.MustGet("zone").(*happydns.Zone)
	subdomain := c.MustGet("subdomain").(string)

	c.JSON(http.StatusOK, gin.H{"services": zone.Services[subdomain]})
}

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

func getZoneService(c *gin.Context) {
	zone := c.MustGet("zone").(*happydns.Zone)
	serviceid := c.MustGet("serviceid").([]byte)
	subdomain := c.MustGet("subdomain").(string)

	c.JSON(http.StatusOK, zone.FindSubdomainService(subdomain, serviceid))
}

func retrieveZone(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	domain := c.MustGet("domain").(*happydns.Domain)

	provider, err := storage.MainStore.GetProvider(user, domain.IdProvider)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": fmt.Sprintf("Unable to find your provider: %s", err.Error())})
		return
	}

	zone, err := provider.ImportZone(domain)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	services, defaultTTL, err := svcs.AnalyzeZone(domain.DomainName, zone)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
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

	zone.Services, _, err = svcs.AnalyzeZone(domain.DomainName, rrs)
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

	c.JSON(http.StatusOK, zone)
}

func diffZones(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	domain := c.MustGet("domain").(*happydns.Domain)

	if c.Param("zoneid1") != "@" {
		c.AbortWithStatusJSON(http.StatusNotImplemented, gin.H{"errmsg": "Diff between two zone is not implemented."})
		return
	}

	provider, err := storage.MainStore.GetProvider(user, domain.IdProvider)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, fmt.Errorf("Unable to find the given provider: %q for %q", domain.IdProvider, domain.DomainName))
		return
	}

	zone, statuscode, err := loadZoneFromId(domain, c.Param("zoneid2"))
	if err != nil {
		c.AbortWithStatusJSON(statuscode, gin.H{"errmsg": err.Error()})
		return
	}

	records, err := models.RRstoRCs(zone.GenerateRRs(domain.DomainName), strings.TrimSuffix(domain.DomainName, "."))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	dc := &models.DomainConfig{
		Name:    strings.TrimSuffix(domain.DomainName, "."),
		Records: records,
	}

	corrections, err := provider.GetDomainCorrections(domain, dc)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	var rrCorected []string
	for _, c := range corrections {
		rrCorected = append(rrCorected, c.Msg)
	}

	c.JSON(http.StatusOK, rrCorected)
}

func applyZone(c *gin.Context) {
	user := c.MustGet("LoggedUser").(*happydns.User)
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)

	provider, err := storage.MainStore.GetProvider(user, domain.IdProvider)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, fmt.Errorf("Unable to find the given provider: %q for %q", domain.IdProvider, domain.DomainName))
		return
	}

	records, err := models.RRstoRCs(zone.GenerateRRs(domain.DomainName), strings.TrimSuffix(domain.DomainName, "."))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	dc := &models.DomainConfig{
		Name:    strings.TrimSuffix(domain.DomainName, "."),
		Records: records,
	}

	var wantedCorrections []string
	err = c.ShouldBindJSON(&wantedCorrections)
	if err != nil {
		log.Printf("%s sends invalid string array JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	corrections, err := provider.GetDomainCorrections(domain, dc)
	for _, cr := range corrections {
		for ic, wc := range wantedCorrections {
			if wc == cr.Msg {
				log.Printf("%s: apply correction: %s", domain.DomainName, cr.Msg)
				err := cr.F()

				if err != nil {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to update the zone: %s", err.Error())})
					return
				}

				wantedCorrections = append(wantedCorrections[:ic], wantedCorrections[ic+1:]...)

				break
			}
		}
	}

	if len(wantedCorrections) > 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to perform the following changes: %s", wantedCorrections)})
		return
	}

	// Create a new zone in history for futher updates
	newZone := zone.DerivateNew()
	//newZone.IdAuthor = //TODO get current user id
	err = storage.MainStore.CreateZone(newZone)
	if err != nil {
		log.Printf("%s was unable to CreateZone", c.ClientIP(), err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are unable to create the zone now."})
		return
	}

	domain.ZoneHistory = append(
		[]happydns.Identifier{newZone.Id}, domain.ZoneHistory...)

	err = storage.MainStore.UpdateDomain(domain)
	if err != nil {
		log.Printf("%s was unable to UpdateDomain", c.ClientIP(), err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are unable to create the zone now."})
		return
	}

	// Commit changes in previous zone
	now := time.Now()
	// zone.ZoneMeta.IdAuthor = // TODO get current user id
	zone.ZoneMeta.Published = &now

	zone.LastModified = time.Now()

	err = storage.MainStore.UpdateZone(zone)
	if err != nil {
		log.Printf("%s was unable to UpdateZone", c.ClientIP(), err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are unable to create the zone now."})
		return
	}

	c.JSON(http.StatusOK, newZone.ZoneMeta)
}

func viewZone(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	zone := c.MustGet("zone").(*happydns.Zone)

	var ret string

	for _, rr := range zone.GenerateRRs(domain.DomainName) {
		ret += rr.String() + "\n"
	}

	c.JSON(http.StatusOK, ret)
}

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
	String string  `json:"str"`
	Fields *dns.RR `json:"fields,omitempty"`
}

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

	var ret []serviceRecord
	for _, rr := range svc.GenRRs(subdomain, 3600, domain.DomainName) {
		ret = append(ret, serviceRecord{
			String: rr.String(),
			Fields: &rr,
		})
	}

	c.JSON(http.StatusOK, ret)
}
