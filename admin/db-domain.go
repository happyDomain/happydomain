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

package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happydns/api"
	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/storage"
)

func declareDomainsRoutes(opts *config.Options, router *gin.RouterGroup) {
	router.GET("/domains", func(c *gin.Context) {
		if _, exists := c.Get("user"); exists {
			api.GetDomains(c)
		} else {
			getAllDomains(c)
		}
	})
	router.POST("/domains", newDomain)

	router.DELETE("/domains/:domain", deleteUserDomain)

	apiDomainsRoutes := router.Group("/domains/:domain")
	apiDomainsRoutes.Use(api.DomainHandler)

	apiDomainsRoutes.GET("", api.GetDomain)
	apiDomainsRoutes.PUT("", updateUserDomain)

	declareZonesRoutes(opts, apiDomainsRoutes)
}

func getAllDomains(c *gin.Context) {
	var domains happydns.Domains

	users, err := storage.MainStore.GetUsers()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": fmt.Sprintf("Unable to retrieve users list: %s", err.Error())})
		return
	}
	for _, user := range users {
		usersDomains, err := storage.MainStore.GetDomains(user)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": fmt.Sprintf("Unable to retrieve %s's domains: %s", user.Email, err.Error())})
			return
		}

		domains = append(domains, usersDomains...)
	}

	ApiResponse(c, domains, nil)
}

func newDomain(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	ud := &happydns.Domain{}
	err := c.ShouldBindJSON(&ud)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}
	ud.Id = 0
	ud.IdUser = user.Id

	ApiResponse(c, ud, storage.MainStore.CreateDomain(user, ud))
}

func updateUserDomain(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	ud := &happydns.Domain{}
	err := c.ShouldBindJSON(&ud)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}
	ud.Id = domain.Id

	if ud.IdUser != domain.IdUser {
		if err := storage.MainStore.UpdateDomainOwner(domain, &happydns.User{Id: ud.IdUser}); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
			return
		}
	}

	ApiResponse(c, ud, storage.MainStore.UpdateDomain(ud))
}

func deleteUserDomain(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	domainid, err := strconv.ParseInt(c.Param("domain"), 10, 64)
	if err != nil {
		domain, err := storage.MainStore.GetDomainByDN(user, c.Param("domain"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": err.Error()})
			return
		} else {
			domainid = domain.Id
		}
	}
	ApiResponse(c, true, storage.MainStore.DeleteDomain(&happydns.Domain{Id: domainid}))
}

func clearDomains(c *gin.Context) {
	ApiResponse(c, true, storage.MainStore.ClearDomains())
}
