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

// Package controller exposes the "fetch certificate" endpoint used by the
// TLSA editor to prefill Certificate hashes from a live TLS endpoint.
//
// Scoped to the domain the user owns (DomainHandler middleware + suffix
// check) so it cannot be repurposed as an arbitrary TLS-probing proxy.
package controller

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	tls "git.happydns.org/checker-tls/checker"
	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

const fetchCertificateTimeout = 10 * time.Second

type CertificateController struct{}

func NewCertificateController() *CertificateController {
	return &CertificateController{}
}

// fetchCertificateRequest is the editor's selection. Host is the owner
// subdomain (without "_port._proto"); STARTTLS is optional and when empty
// we auto-map a handful of common ports.
type fetchCertificateRequest struct {
	Host     string `json:"host" binding:"required"`
	Port     uint16 `json:"port" binding:"required"`
	Proto    string `json:"proto"`
	STARTTLS string `json:"starttls"`
}

// fetchCertificateResponse carries the full chain (leaf first) so the editor
// can offer DANE-EE and DANE-TA hashes side by side.
type fetchCertificateResponse struct {
	Endpoint string         `json:"endpoint"`
	Chain    []tls.CertInfo `json:"chain"`
}

// FetchCertificate dials the requested endpoint and returns DANE-friendly
// pre-hashed views of the server's certificate chain.
//
//	@Summary	Fetch a live certificate for a subdomain
//	@Tags		domains
//	@Accept		json
//	@Produce	json
//	@Param		domain	path		string					true	"Domain identifier"
//	@Param		body	body		fetchCertificateRequest	true	"Endpoint to probe"
//	@Success	200		{object}	fetchCertificateResponse
//	@Failure	400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure	403		{object}	happydns.ErrorResponse	"Host not under this domain"
//	@Failure	502		{object}	happydns.ErrorResponse	"Upstream TLS error"
//	@Router		/domains/{domain}/fetch-certificate [post]
func (cc *CertificateController) FetchCertificate(c *gin.Context) {
	var req fetchCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}
	if req.Port == 0 {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("port is required"))
		return
	}
	proto := strings.ToLower(strings.TrimSpace(req.Proto))
	if proto == "" {
		proto = "tcp"
	}
	if proto != "tcp" && proto != "udp" {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("unsupported proto %q", req.Proto))
		return
	}

	// Authorization: the authenticated domain must be a suffix of Host. We
	// trust c.Get("domain") (set by DomainHandler), not the client-supplied
	// Host, so the endpoint can't double as an arbitrary TLS-probing proxy.
	domVal, ok := c.Get("domain")
	if !ok {
		middleware.ErrorResponse(c, http.StatusForbidden, fmt.Errorf("domain context missing"))
		return
	}
	dom, ok := domVal.(*happydns.Domain)
	if !ok {
		middleware.ErrorResponse(c, http.StatusInternalServerError, fmt.Errorf("unexpected domain context type"))
		return
	}
	host := strings.TrimSpace(req.Host)
	if !strings.HasSuffix(host, dom.DomainName) {
		middleware.ErrorResponse(c, http.StatusForbidden, fmt.Errorf("host %q is not under %q", host, dom.DomainName))
		return
	}

	host = strings.TrimSuffix(host, ".")

	starttls := req.STARTTLS
	if starttls == "" {
		starttls = tls.AutoSTARTTLS(req.Port)
	}

	chain, err := tls.FetchChain(c.Request.Context(), host, req.Port, starttls, fetchCertificateTimeout)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadGateway, err)
		return
	}

	c.JSON(http.StatusOK, fetchCertificateResponse{
		Endpoint: net.JoinHostPort(host, strconv.FormatUint(uint64(req.Port), 10)),
		Chain:    tls.BuildChain(chain),
	})
}
