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
	"strings"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

func ParseZoneId(c *gin.Context, paramName string) (happydns.Identifier, error) {
	zoneid, err := happydns.NewIdentifierFromString(c.Param(paramName))
	if err != nil {
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("bad zone identifier format (%s): %s", paramName, err.Error())}
	}

	return zoneid, nil
}

func ZoneHandler(zuService happydns.ZoneUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		domain := c.MustGet("domain").(*happydns.Domain)

		zoneid, err := ParseZoneId(c, "zoneid")
		if err != nil {
			ErrorResponse(c, http.StatusInternalServerError, err)
		}

		zone, err := zuService.LoadZoneFromId(domain, zoneid)
		if err != nil {
			ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}

		c.Set("zone", zone)

		c.Next()
	}
}

func SubdomainHandler(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	subdomain := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(c.Param("subdomain"), "."+domain.DomainName), "@"), domain.DomainName)

	c.Set("subdomain", subdomain)

	c.Next()
}
