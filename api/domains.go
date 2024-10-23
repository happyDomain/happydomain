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
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/storage"
)

func declareDomainsRoutes(cfg *config.Options, router *gin.RouterGroup) {
	router.GET("/domains", GetDomains)
	router.POST("/domains", addDomain)

	apiDomainsRoutes := router.Group("/domains/:domain")
	apiDomainsRoutes.Use(DomainHandler)

	apiDomainsRoutes.GET("", GetDomain)
	apiDomainsRoutes.PUT("", UpdateDomain)
	apiDomainsRoutes.DELETE("", delDomain)

	apiDomainsRoutes.GET("/logs", GetDomainLogs)

	declareZonesRoutes(cfg, apiDomainsRoutes)
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
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Failure		404	{object}	happydns.Error	"Unable to retrieve user's domains"
//	@Router			/domains [get]
func GetDomains(c *gin.Context) {
	user := myUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined"})
		return
	}

	if domains, err := storage.MainStore.GetDomains(user); err != nil {
		log.Printf("%s: An error occurs when trying to GetDomains: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": err})
	} else if len(domains) > 0 {
		c.JSON(http.StatusOK, domains)
	} else {
		c.JSON(http.StatusOK, []happydns.Domain{})
	}
}

// addDomain appends a new domain to those managed.
//
//	@Summary	Manage a new domain
//	@Schemes
//	@Description	Append a new domain to those managed.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			body	body	happydns.DomainMinimal	true	"Domain object that you want to manage through happyDomain."
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.Domain
//	@Failure		400	{object}	happydns.Error	"Error in received data"
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Failure		500	{object}	happydns.Error	"Unable to retrieve current user's domains"
//	@Router			/domains [post]
func addDomain(c *gin.Context) {
	var uz happydns.Domain
	err := c.ShouldBindJSON(&uz)
	if err != nil {
		log.Printf("%s sends invalid Domain JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if len(uz.DomainName) <= 2 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "The given domain is invalid."})
		return
	}

	uz.DomainName = dns.Fqdn(uz.DomainName)

	if _, ok := dns.IsDomainName(uz.DomainName); !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("%q is not a valid domain name.", uz.DomainName)})
		return
	}

	user := c.MustGet("LoggedUser").(*happydns.User)

	p, err := storage.MainStore.GetProvider(user, uz.IdProvider)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to find the provider.")})
		return
	}

	provider, err := p.ParseProvider()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": fmt.Errorf("Unable to retrieve provider's data: %s", err.Error())})
		return
	}

	if storage.MainStore.DomainExists(uz.DomainName) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "This domain has already been imported."})
		return

	} else if err := provider.DomainExists(uz.DomainName); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	} else if err := storage.MainStore.CreateDomain(user, &uz); err != nil {
		log.Printf("%s was unable to CreateDomain: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are unable to create your domain now."})
		return
	} else {
		storage.MainStore.CreateDomainLog(&uz, happydns.NewDomainLog(c.MustGet("LoggedUser").(*happydns.User), happydns.LOG_INFO, fmt.Sprintf("Domain name %s added.", uz.DomainName)))

		c.JSON(http.StatusOK, uz)
	}
}

func DomainHandler(c *gin.Context) {
	// Get a valid user
	user := myUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined."})
		return
	}

	dnid, err := happydns.NewIdentifierFromString(c.Param("domain"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Invalid domain identifier: %s", err.Error())})
		return
	}

	domain, err := storage.MainStore.GetDomain(user, dnid)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Domain not found"})
		return
	}

	// If provider is provided, check that the domain is a parent of the provider
	var provider *happydns.ProviderMeta
	if src, exists := c.Get("provider"); exists {
		provider = &src.(*happydns.ProviderCombined).ProviderMeta
	} else if src, exists := c.Get("providermeta"); exists {
		provider = src.(*happydns.ProviderMeta)
	}
	if provider != nil && !provider.Id.Equals(domain.IdProvider) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Domain not found (not child of provider)"})
		return
	}

	c.Set("domain", domain)

	c.Next()
}

type APIDomain struct {
	// Id is the Domain's identifier in the database.
	Id happydns.Identifier `json:"id" swaggertype:"string"`

	// IdUser is the identifier of the Domain's Owner.
	IdUser happydns.Identifier `json:"id_owner" swaggertype:"string"`

	// IsProvider is the identifier of the Provider used to access and edit the
	// Domain.
	IdProvider happydns.Identifier `json:"id_provider" swaggertype:"string"`

	// DomainName is the FQDN of the managed Domain.
	DomainName string `json:"domain"`
	// Group is a hint string aims to group domains.
	Group string `json:"group,omitempty"`

	// ZoneHistory are the metadata associated to each Zone saved with the
	// current Domain.
	ZoneHistory []happydns.ZoneMeta `json:"zone_history"`
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
//	@Success		200	{object}	APIDomain
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Failure		404	{object}	happydns.Error	"Domain not found"
//	@Router			/domains/{domainId} [get]
func GetDomain(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	ret := &APIDomain{
		Id:          domain.Id,
		IdUser:      domain.IdUser,
		IdProvider:  domain.IdProvider,
		DomainName:  domain.DomainName,
		ZoneHistory: []happydns.ZoneMeta{},
		Group:       domain.Group,
	}

	for _, zm := range domain.ZoneHistory {
		zoneMeta, err := storage.MainStore.GetZoneMeta(zm)

		if err != nil {
			log.Printf("%s: An error occurs in getDomain, when retrieving a meta history: %s", c.ClientIP(), err.Error())
		} else {
			ret.ZoneHistory = append(ret.ZoneHistory, *zoneMeta)
		}
	}

	c.JSON(http.StatusOK, ret)
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
//	@Failure		400	{object}	happydns.Error	"Invalid input"
//	@Failure		400	{object}	happydns.Error	"Identifier changed"
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Failure		404	{object}	happydns.Error	"Domain not found"
//	@Failure		500	{object}	happydns.Error	"Database writing error"
//	@Router			/domains/{domainId} [put]
func UpdateDomain(c *gin.Context) {
	old := c.MustGet("domain").(*happydns.Domain)

	var domain happydns.Domain
	err := c.ShouldBindJSON(&domain)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	if !old.Id.Equals(domain.Id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "You cannot change the domain reserved ID"})
		return
	}

	old.Group = domain.Group

	err = storage.MainStore.UpdateDomain(old)
	if err != nil {
		log.Printf("%s: Unable to UpdateDomain in UpdateDomain: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to update your domain. Please retry later."})
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
//	@Failure		400	{object}	happydns.Error	"Invalid input"
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Failure		404	{object}	happydns.Error	"Domain not found"
//	@Failure		500	{object}	happydns.Error	"Database writing error"
//	@Router			/domains/{domainId} [delete]
func delDomain(c *gin.Context) {
	if err := storage.MainStore.DeleteDomain(c.MustGet("domain").(*happydns.Domain)); err != nil {
		log.Printf("%s was unable to DeleteDomain: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": fmt.Sprintf("Unable to delete your domain: %s", err.Error())})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetDomainLogs retrieves actions recorded for the domain.
//
//	@Summary	Retrieve Domain actions history.
//	@Schemes
//	@Description	Retrieve information about the actions performed on the domain by users of happyDomain.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			domainId	path	string	true	"Domain identifier"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	[]happydns.DomainLog
//	@Failure		401	{object}	happydns.Error	"Authentication failure"
//	@Failure		404	{object}	happydns.Error	"Domain not found"
//	@Router			/domains/{domainId}/logs [get]
func GetDomainLogs(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	logs, err := storage.MainStore.GetDomainLogs(domain)

	if err != nil {
		log.Printf("%s: An error occurs in GetDomainLogs, when retrieving logs: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Unable to access the domain logs. Please try again later."})
		return
	}

	// Sort by date
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Date.After(logs[j].Date)
	})

	c.JSON(http.StatusOK, logs)
}
