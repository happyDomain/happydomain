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
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

// policyHTTPCacheSeconds is how long intermediaries may cache the policy file.
//
// It is deliberately unrelated to the policy's own max_age: max_age tells
// senders how long to trust the policy they already fetched, whereas an HTTP
// cache of the same length would keep a *stale* policy in front of senders
// that did re-fetch after seeing a new id in DNS.
const policyHTTPCacheSeconds = 3600

// MTASTSController serves the public MTA-STS policy file (RFC 8461) for the
// domains that asked happyDomain to host it.
type MTASTSController struct {
	uc happydns.MTASTSUsecase
}

// NewMTASTSController constructs an MTASTSController.
func NewMTASTSController(uc happydns.MTASTSUsecase) *MTASTSController {
	return &MTASTSController{uc: uc}
}

// Policy serves https://mta-sts.<domain>/.well-known/mta-sts.txt.
//
// The domain is taken from the Host header: the policy file has no room for a
// query parameter, and RFC 8461 sec. 3.3 pins the URL down to that exact form.
//
//	@Summary	MTA-STS policy file
//	@Description	Returns the RFC 8461 policy file for the domain the request was addressed to.
//	@Tags			mta-sts
//	@Produce		text/plain
//	@Success		200	{string}	string
//	@Failure		400	{object}	happydns.ErrorResponse
//	@Failure		404	{object}	happydns.ErrorResponse
//	@Router			/.well-known/mta-sts.txt [get]
func (mc *MTASTSController) Policy(c *gin.Context) {
	host := c.Request.Host
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	if host == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: "missing domain"})
		return
	}

	body, err := mc.uc.Policy(dns.Fqdn(host))
	if err != nil {
		if errors.Is(err, happydns.ErrNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: "no MTA-STS policy found for this domain"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	c.Header("Cache-Control", "public, max-age="+strconv.Itoa(policyHTTPCacheSeconds))
	c.Data(http.StatusOK, "text/plain; charset=utf-8", body)
}
