// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
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
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/services"
	"git.happydns.org/happydns/storage"
)

func declareServiceRoutes(router *gin.RouterGroup) {
	router.GET("/services", listServices)
	//router.POST("/services", newService)

	//router.POST("/domains/:domain/analyze", analyzeDomain)
}

func listServices(c *gin.Context) {
	ret := map[string]svcs.ServiceInfos{}

	for k, svc := range *svcs.GetServices() {
		ret[k] = svc.Infos
	}

	c.JSON(http.StatusOK, ret)
}

func analyzeDomain(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	user := myUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined"})
		return
	}

	provider, err := storage.MainStore.GetProvider(user, domain.IdProvider)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to get the related provider: %s", err.Error())})
		return
	}

	zone, err := provider.ImportZone(domain)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to import zone: %s", err.Error())})
		return
	}

	services, defaultTTL, err := svcs.AnalyzeZone(domain.DomainName, zone)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": fmt.Sprintf("An error occurs during analysis: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"services":   services,
		"defaultTTL": defaultTTL,
	})
}
