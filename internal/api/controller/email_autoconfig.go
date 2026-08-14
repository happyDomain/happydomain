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
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

// EmailAutoconfigController serves the public mail-client auto-configuration
// endpoints used by Thunderbird (Mozilla Autoconfig) and Outlook (Microsoft
// Autodiscover).
type EmailAutoconfigController struct {
	uc happydns.EmailAutoconfigUsecase
}

// NewEmailAutoconfigController constructs an EmailAutoconfigController.
func NewEmailAutoconfigController(uc happydns.EmailAutoconfigUsecase) *EmailAutoconfigController {
	return &EmailAutoconfigController{uc: uc}
}

// resolveDomain extracts the domain to look up. Priority: emailaddress query
// param → Host header (with the autoconfig./autodiscover. prefix stripped).
func resolveDomain(c *gin.Context, emailParamNames ...string) string {
	for _, name := range emailParamNames {
		if v := c.Query(name); v != "" {
			if at := strings.LastIndex(v, "@"); at >= 0 {
				return v[at+1:]
			}
		}
	}
	host := c.Request.Host
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	return host
}

// MozillaAutoconfig serves the Thunderbird config-v1.1.xml format.
//
//	@Summary	Mail-client auto-configuration (Mozilla Autoconfig)
//	@Description	Returns the Thunderbird-style XML configuration for the requested domain.
//	@Tags			email-autoconfig
//	@Produce		application/xml
//	@Param			emailaddress	query	string	false	"Email address (used to derive the domain)"
//	@Success		200	{string}	string
//	@Failure		404	{object}	happydns.ErrorResponse
//	@Router			/mail/config-v1.1.xml [get]
func (ec *EmailAutoconfigController) MozillaAutoconfig(c *gin.Context) {
	domain := resolveDomain(c, "emailaddress")
	if domain == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: "missing domain"})
		return
	}

	body, err := ec.uc.MozillaConfig(dns.Fqdn(domain), c.Query("emailaddress"))
	if err != nil {
		if errors.Is(err, happydns.ErrNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: "no auto-configuration found for this domain"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/xml; charset=utf-8", body)
}

// autodiscoverRequest is the (very small) subset of Outlook's POST body we
// actually need to read.
type autodiscoverRequest struct {
	XMLName xml.Name `xml:"Autodiscover"`
	Request struct {
		EMailAddress string `xml:"EMailAddress"`
	} `xml:"Request"`
}

// MSAutodiscover serves the Microsoft Autodiscover POX format. Outlook may
// hit this endpoint with either GET or POST; both are handled identically
// from happyDomain's perspective (we only need the email address).
//
//	@Summary	Mail-client auto-configuration (Microsoft Autodiscover)
//	@Description	Returns the Outlook-style XML configuration for the requested domain.
//	@Tags			email-autoconfig
//	@Produce		application/xml
//	@Success		200	{string}	string
//	@Failure		404	{object}	happydns.ErrorResponse
//	@Router			/Autodiscover/Autodiscover.xml [post]
func (ec *EmailAutoconfigController) MSAutodiscover(c *gin.Context) {
	emailAddress := c.Query("emailaddress")
	if emailAddress == "" {
		emailAddress = c.Query("Email")
	}

	if c.Request.Method == http.MethodPost && c.Request.Body != nil {
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, 64*1024))
		if err == nil && len(body) > 0 {
			var req autodiscoverRequest
			if xmlErr := xml.Unmarshal(body, &req); xmlErr == nil && req.Request.EMailAddress != "" {
				emailAddress = req.Request.EMailAddress
			}
		}
	}

	domain := resolveDomain(c, "emailaddress", "Email")
	if at := strings.LastIndex(emailAddress, "@"); at >= 0 {
		domain = emailAddress[at+1:]
	}
	if domain == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: "missing domain"})
		return
	}

	body, err := ec.uc.AutodiscoverConfig(dns.Fqdn(domain), emailAddress)
	if err != nil {
		if errors.Is(err, happydns.ErrNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: "no auto-configuration found for this domain"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/xml; charset=utf-8", body)
}
