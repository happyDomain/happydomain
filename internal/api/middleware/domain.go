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

package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

func DomainHandler(domainService happydns.DomainUsecase, allowFQDN bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get a valid user
		user := MyUser(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined."})
			return
		}

		var domain *happydns.Domain

		dnid, err := happydns.NewIdentifierFromString(c.Param("domain"))
		if err != nil {
			if !allowFQDN {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Invalid domain identifier: %s", err.Error())})
				return
			}

			var domains []*happydns.Domain
			domains, err = domainService.GetUserDomainByFQDN(user, c.Param("domain"))
			if err != nil || len(domains) == 0 {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Domain not found"})
				return
			}

			if len(domains) != 1 {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "There are many domain names with this FQDN in your account, please use their ID to access it instead"})
				return
			}

			domain = domains[0]
		} else {
			domain, err = domainService.GetUserDomain(user, dnid)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Domain not found"})
				return
			}
		}

		// If provider is provided, check that the domain is a parent of the provider
		var provider *happydns.ProviderMeta
		if src, exists := c.Get("provider"); exists {
			provider = &src.(*happydns.Provider).ProviderMeta
		} else if src, exists := c.Get("providermeta"); exists {
			provider = src.(*happydns.ProviderMeta)
		}
		if provider != nil && !provider.Id.Equals(domain.ProviderId) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Domain not found (not child of provider)"})
			return
		}

		c.Set("domain", domain)

		c.Next()
	}
}

// DomainOwnerOnly restricts a route to the owner of the domain resolved by
// DomainHandler. It guards the operations that mutate state shared by everyone
// working on the domain (check plans, checker options, executions,
// acknowledgements): a user invited through a share grant keeps a read-only
// view of them.
func DomainOwnerOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		d, exists := c.Get("domain")
		if !exists {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Domain not resolved."})
			return
		}
		domain := d.(*happydns.Domain)

		user := MyUser(c)
		if user == nil || !user.Id.Equals(domain.Owner) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Only the domain owner can perform this action."})
			return
		}

		c.Next()
	}
}
