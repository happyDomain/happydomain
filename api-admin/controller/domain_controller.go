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

	"git.happydns.org/happyDomain/api/controller"
	"git.happydns.org/happyDomain/api/middleware"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type DomainController struct {
	domainService happydns.DomainUsecase
	store         storage.DomainStorage
}

func NewDomainController(duService happydns.DomainUsecase, store storage.DomainStorage) *DomainController {
	return &DomainController{
		duService,
		store,
	}
}

func (dc *DomainController) ListDomains(c *gin.Context) {
	user := middleware.MyUser(c)
	if user != nil {
		apidc := controller.NewDomainController(dc.domainService)
		apidc.GetDomains(c)
		return
	}

	domains, err := dc.store.ListAllDomains()
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, fmt.Errorf("unable to retrieve domains list: %w", err))
		return
	}

	happydns.ApiResponse(c, domains, nil)
}

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
	domains, err := dc.store.ListAllDomains()
	if err != nil {
		log.Println("Unable to retrieve domains list:", err.Error())
		return nil
	}
	for _, domain := range domains {
		if filter(domain) {
			// Create a fake minimal user, as only the Id is required to perform further actions on database
			return &happydns.User{Id: domain.Owner}
		}
	}

	return nil
}

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

func (dc *DomainController) UpdateZones(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	err := c.ShouldBindJSON(&domain.ZoneHistory)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, fmt.Errorf("something is wrong in received data: %w", err))
		return
	}

	happydns.ApiResponse(c, domain, dc.store.UpdateDomain(domain))
}
