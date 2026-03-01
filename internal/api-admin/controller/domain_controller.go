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
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/controller"
	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/usecase/domain"
	"git.happydns.org/happyDomain/model"
)

type DomainController struct {
	domainService      happydns.DomainUsecase
	remoteZoneImporter happydns.RemoteZoneImporterUsecase
	zoneImporter       happydns.ZoneImporterUsecase
	store              domain.DomainStorage
}

func NewDomainController(duService happydns.DomainUsecase, remoteZoneImporter happydns.RemoteZoneImporterUsecase, zoneImporter happydns.ZoneImporterUsecase, store domain.DomainStorage) *DomainController {
	return &DomainController{
		duService,
		remoteZoneImporter,
		zoneImporter,
		store,
	}
}

// ListDomains retrieves all domains in the system or user-specific domains if authenticated.
//
//	@Summary		List all domains
//	@Schemes
//	@Description	Retrieve all domains in the system. If a user is authenticated, returns only their domains using the regular API controller.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					false	"User ID or email"
//	@Param			pid		path		string					false	"Provider identifier"
//	@Success		200	{array}		happydns.Domain
//	@Failure		500	{object}	happydns.ErrorResponse	"Unable to retrieve domains list"
//	@Router			/domains [get]
//	@Router			/users/{uid}/domains [get]
//	@Router			/users/{uid}/providers/{pid}/domains [get]
func (dc *DomainController) ListDomains(c *gin.Context) {
	user := middleware.MyUser(c)
	if user != nil {
		apidc := controller.NewDomainController(dc.domainService, dc.remoteZoneImporter, dc.zoneImporter, nil)
		apidc.GetDomains(c)
		return
	}

	iter, err := dc.store.ListAllDomains()
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, fmt.Errorf("unable to retrieve domains list: %w", err))
		return
	}
	defer iter.Close()

	var domains []*happydns.Domain
	for iter.Next() {
		domains = append(domains, iter.Item())
	}

	happydns.ApiResponse(c, domains, nil)
}

// NewDomain creates a new domain in the system.
//
//	@Summary		Create a new domain
//	@Schemes
//	@Description	Create a new domain and assign it to the authenticated user.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string					false	"User ID or email"
//	@Param			pid		path		string					false	"Provider identifier"
//	@Param			body	body		happydns.Domain	true	"Domain object to create"
//	@Success		200		{object}	happydns.Domain
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input data"
//	@Failure		500		{object}	happydns.ErrorResponse	"Unable to create domain"
//	@Router			/domains [post]
//	@Router			/users/{uid}/domains [post]
//	@Router			/users/{uid}/providers/{pid}/domains [post]
func (dc *DomainController) NewDomain(c *gin.Context) {
	user := c.MustGet("user").(*happydns.User)

	ud := &happydns.Domain{}
	err := c.ShouldBindJSON(&ud)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("something is wrong in received data: %w", err))
		return
	}
	ud.Id = nil
	ud.Owner = user.Id

	happydns.ApiResponse(c, ud, dc.store.CreateDomain(ud))
}

// DeleteDomain removes a domain from the system by identifier or domain name.
//
//	@Summary		Delete a domain
//	@Schemes
//	@Description	Delete a domain by its identifier or fully qualified domain name. Searches for the domain owner if user context is not available.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string	false	"User ID or email"
//	@Param			pid		path		string	false	"Provider identifier"
//	@Param			domain	path		string	true	"Domain identifier or fully qualified domain name"
//	@Success		200		{boolean}	true
//	@Failure		404		{object}	happydns.ErrorResponse	"Domain not found"
//	@Failure		500		{object}	happydns.ErrorResponse	"Unable to delete domain"
//	@Router			/domains/{domain} [delete]
//	@Router			/users/{uid}/domains/{domain} [delete]
//	@Router			/users/{uid}/providers/{pid}/domains/{domain} [delete]
func (dc *DomainController) DeleteDomain(c *gin.Context) {
	domainid, err := happydns.NewIdentifierFromString(c.Param("domain"))
	if err != nil {
		var user *happydns.User

		if u, ok := c.Get("user"); ok {
			user = u.(*happydns.User)
		} else {
			user = dc.searchUserDomain(func(dn *happydns.Domain) bool {
				return dn.DomainName == c.Param("domain")
			})
		}

		domains, err := dc.store.GetDomainByDN(user, c.Param("domain"))
		if err != nil {
			middleware.ErrorResponse(c, http.StatusNotFound, err)
			return
		}

		if len(domains) != 1 {
			middleware.ErrorResponse(c, http.StatusNotFound, fmt.Errorf("too many domains with this FQDN, use domain identifier instead"))
			return
		}

		domainid = domains[0].Id
	}

	happydns.ApiResponse(c, true, dc.store.DeleteDomain(domainid))
}

func (dc *DomainController) searchUserDomain(filter func(*happydns.Domain) bool) *happydns.User {
	iter, err := dc.store.ListAllDomains()
	if err != nil {
		log.Println("Unable to retrieve domains list:", err.Error())
		return nil
	}
	defer iter.Close()

	for iter.Next() {
		domain := iter.Item()
		if filter(domain) {
			// Create a fake minimal user, as only the Id is required to perform further actions on database
			return &happydns.User{Id: domain.Owner}
		}
	}

	return nil
}

// GetDomain retrieves a specific domain by identifier or domain name.
//
//	@Summary		Get domain information
//	@Schemes
//	@Description	Retrieve a domain by its identifier or fully qualified domain name. Validates user ownership.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string	false	"User ID or email"
//	@Param			pid		path		string	false	"Provider identifier"
//	@Param			domain	path		string	true	"Domain identifier or fully qualified domain name"
//	@Success		200		{object}	happydns.Domain
//	@Success		200		{array}		happydns.Domain	"When queried by domain name, may return multiple matches"
//	@Failure		404		{object}	happydns.ErrorResponse	"Domain not found"
//	@Failure		500		{object}	happydns.ErrorResponse	"Unable to retrieve domain"
//	@Router			/domains/{domain} [get]
//	@Router			/users/{uid}/domains/{domain} [get]
//	@Router			/users/{uid}/providers/{pid}/domains/{domain} [get]
func (dc *DomainController) GetDomain(c *gin.Context) {
	domainid, err := happydns.NewIdentifierFromString(c.Param("domain"))
	if err != nil {
		var user *happydns.User

		if u, ok := c.Get("user"); ok {
			user = u.(*happydns.User)
		} else {
			user = dc.searchUserDomain(func(dn *happydns.Domain) bool {
				return dn.DomainName == c.Param("domain")
			})
		}

		domain, err := dc.store.GetDomainByDN(user, c.Param("domain"))
		happydns.ApiResponse(c, domain, err)
	} else {
		var user *happydns.User

		if u, ok := c.Get("user"); ok {
			user = u.(*happydns.User)
		} else {
			user = dc.searchUserDomain(func(dn *happydns.Domain) bool {
				return dn.Id.Equals(domainid)
			})
		}

		domain, err := dc.store.GetDomain(domainid)
		if err != nil {
			happydns.ApiResponse(c, nil, err)
			return
		}

		if !user.Id.Equals(domain.Owner) {
			happydns.ApiResponse(c, nil, fmt.Errorf("domain not found"))
			return
		}

		happydns.ApiResponse(c, domain, err)
	}
}

// UpdateDomain updates an existing domain's information.
//
//	@Summary		Update domain information
//	@Schemes
//	@Description	Update the information of an existing domain. The domain ID is preserved from the existing domain.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string			false	"User ID or email"
//	@Param			pid		path		string			false	"Provider identifier"
//	@Param			domain	path		string			true	"Domain identifier"
//	@Param			body	body		happydns.Domain	true	"Updated domain object"
//	@Success		200		{object}	happydns.Domain
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input data"
//	@Failure		404		{object}	happydns.ErrorResponse	"Domain not found"
//	@Failure		500		{object}	happydns.ErrorResponse	"Unable to update domain"
//	@Router			/domains/{domain} [put]
//	@Router			/users/{uid}/domains/{domain} [put]
//	@Router			/users/{uid}/providers/{pid}/domains/{domain} [put]
func (dc *DomainController) UpdateDomain(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	ud := &happydns.Domain{}
	err := c.ShouldBindJSON(&ud)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("something is wrong in received data: %w", err))
		return
	}
	ud.Id = domain.Id

	happydns.ApiResponse(c, ud, dc.store.UpdateDomain(ud))
}

// ClearDomains removes all domains from the system or all domains belonging to a specific user.
//
//	@Summary		Clear all domains
//	@Schemes
//	@Description	Delete all domains in the system. If a user is authenticated, only deletes their domains.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string	false	"User ID or email"
//	@Param			pid		path		string	false	"Provider identifier"
//	@Success		200	{boolean}	true
//	@Failure		500	{object}	happydns.ErrorResponse	"Unable to clear domains"
//	@Router			/domains [delete]
//	@Router			/users/{uid}/domains [delete]
//	@Router			/users/{uid}/providers/{pid}/domains [delete]
func (dc *DomainController) ClearDomains(c *gin.Context) {
	user := middleware.MyUser(c)
	if user != nil {
		domains, err := dc.domainService.ListUserDomains(user)
		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}

		for _, dn := range domains {
			e := dc.store.DeleteDomain(dn.Id)
			if e != nil {
				err = errors.Join(err, e)
			}
		}

		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusOK)
		return
	}

	happydns.ApiResponse(c, true, dc.store.ClearDomains())
}

// UpdateZones updates the zone history for a specific domain.
//
//	@Summary		Update domain zone history
//	@Schemes
//	@Description	Replace the zone history of a domain with new data.
//	@Tags			zones
//	@Accept			json
//	@Produce		json
//	@Param			uid		path		string						false	"User ID or email"
//	@Param			pid		path		string						false	"Provider identifier"
//	@Param			domain	path		string						true	"Domain identifier"
//	@Param			body	body		[]happydns.Identifier		true	"Array of zone identifiers representing the new history"
//	@Success		200		{object}	happydns.Domain
//	@Failure		400		{object}	happydns.ErrorResponse		"Invalid input data"
//	@Failure		404		{object}	happydns.ErrorResponse		"Domain not found"
//	@Failure		500		{object}	happydns.ErrorResponse		"Unable to update domain"
//	@Router			/domains/{domain}/zones [put]
//	@Router			/users/{uid}/domains/{domain}/zones [put]
//	@Router			/users/{uid}/providers/{pid}/domains/{domain}/zones [put]
func (dc *DomainController) UpdateZones(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	err := c.ShouldBindJSON(&domain.ZoneHistory)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, fmt.Errorf("something is wrong in received data: %w", err))
		return
	}

	happydns.ApiResponse(c, domain, dc.store.UpdateDomain(domain))
}
