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

package admin

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/api"
	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/storage"
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
	ud.Id = nil
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

	if !bytes.Equal(ud.IdUser, domain.IdUser) {
		if err := storage.MainStore.UpdateDomainOwner(domain, &happydns.User{Id: ud.IdUser}); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
			return
		}
	}

	ApiResponse(c, ud, storage.MainStore.UpdateDomain(ud))
}

func searchUserDomain(dn string) *happydns.User {
	users, err := storage.MainStore.GetUsers()
	if err != nil {
		log.Println("Unable to retrieve users list:", err.Error())
		return nil
	}
	for _, user := range users {
		usersDomains, err := storage.MainStore.GetDomains(user)
		if err != nil {
			log.Printf("Unable to retrieve %s's domains: %s", user.Email, err.Error())
			continue
		}

		for _, domain := range usersDomains {
			if domain.DomainName == dn {
				return user
			}
		}
	}

	return nil
}

func deleteUserDomain(c *gin.Context) {
	domainid, err := happydns.NewIdentifierFromString(c.Param("domain"))
	if err != nil {
		var user *happydns.User

		if u, ok := c.Get("user"); ok {
			user = u.(*happydns.User)
		} else {
			user = searchUserDomain(c.Param("domain"))
		}

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
