// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

// CaddyController implements the Caddy on-demand TLS "ask" endpoint.
//
// Caddy's on_demand_tls only accepts a single ask URL, so this one endpoint
// answers for every hosted service: it says yes as soon as any of them claims
// the host name.
type CaddyController struct {
	validators []happydns.HostedDomainValidator
}

// NewCaddyController constructs a CaddyController over the given validators.
// nil validators are dropped, so callers can pass optional usecases straight
// through.
func NewCaddyController(validators ...happydns.HostedDomainValidator) *CaddyController {
	kept := make([]happydns.HostedDomainValidator, 0, len(validators))
	for _, v := range validators {
		if v != nil {
			kept = append(kept, v)
		}
	}
	return &CaddyController{validators: kept}
}

// HasValidators reports whether the endpoint could ever answer positively.
// When it cannot, there is no point in exposing it at all.
func (cc *CaddyController) HasValidators() bool {
	return len(cc.validators) > 0
}

// Ask implements the Caddy on-demand TLS "ask" endpoint. Caddy treats any 2xx
// response as "go ahead and issue the cert" and any other status as "deny".
// Each validator is scoped strictly to the host prefixes it serves, so
// happyDomain never authorises certs for arbitrary domains.
//
//	@Summary	Caddy on-demand TLS validation
//	@Description	Returns 200 when happyDomain hosts content for the requested domain.
//	@Tags			service-hosting
//	@Param			domain	query	string	true	"FQDN Caddy is about to obtain a certificate for"
//	@Success		200
//	@Failure		400
//	@Failure		404
//	@Router			/caddy/ask [get]
func (cc *CaddyController) Ask(c *gin.Context) {
	domain := strings.TrimSpace(c.Query("domain"))
	if domain == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	for _, validator := range cc.validators {
		managed, err := validator.IsManaged(domain)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if managed {
			c.Status(http.StatusOK)
			return
		}
	}

	c.AbortWithStatus(http.StatusNotFound)
}
