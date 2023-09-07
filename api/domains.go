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

	provider, err := storage.MainStore.GetProvider(user, uz.IdProvider)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to find the provider.")})
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
