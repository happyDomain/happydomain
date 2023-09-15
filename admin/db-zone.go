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
