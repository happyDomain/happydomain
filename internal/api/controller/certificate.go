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
// Scoped to the domain the user owns (DomainHandler middleware + label-aware
// subdomain check) so it cannot be repurposed as an arbitrary TLS-probing
// proxy.
package controller

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"

	tls "git.happydns.org/checker-tls/checker"
	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

const fetchCertificateTimeout = 10 * time.Second

type CertificateController struct {
	// guard vets the endpoint before it is probed. The suffix check below is
	// authorization, not reachability: the caller owns the domain, so they also
	// control what it resolves to, and can point it at an internal address.
	guard *netguard.Guard
}

func NewCertificateController(guard *netguard.Guard) *CertificateController {
	return &CertificateController{guard: guard}
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

// isUnderDomain reports whether host is domain itself or one of its
// subdomains. Names are compared canonically (lowercased, fully qualified) and
// only at a label boundary, so "notexample.com" is not read as being under
// "example.com".
func isUnderDomain(host, domain string) bool {
	canonHost := dns.CanonicalName(host)
	canonDomain := dns.CanonicalName(domain)

	if canonDomain == "." {
		return false
	}

	return canonHost == canonDomain || strings.HasSuffix(canonHost, "."+canonDomain)
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

	// Authorization: Host must be the authenticated domain itself or one of its
	// subdomains, compared label by label, not as a raw byte suffix. We
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
	if _, ok := dns.IsDomainName(host); !ok || strings.ContainsAny(host, `/\`) {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid host %q", req.Host))
		return
	}
	if !isUnderDomain(host, dom.DomainName) {
		middleware.ErrorResponse(c, http.StatusForbidden, fmt.Errorf("host %q is not under %q", host, dom.DomainName))
		return
	}

	host = strings.TrimSuffix(dns.CanonicalName(host), ".")

	// checker-tls' FetchChain takes no dialer, so the best available check is
	// to resolve first and refuse a host that lands anywhere we are not allowed
	// to reach. It re-resolves internally, leaving a one-TTL rebinding window
	// we cannot close from here.
	if _, err := cc.guard.ResolveAllowed(c.Request.Context(), host); err != nil {
		middleware.ErrorResponse(c, http.StatusForbidden, errors.New(cc.guard.Refusal("This endpoint")))
		return
	}

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
