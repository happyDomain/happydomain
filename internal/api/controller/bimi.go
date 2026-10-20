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
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

const (
	// bimiCacheControl lets intermediaries keep the files for a day: their
	// names change with their content, so a cached file is never stale.
	bimiCacheControl = "public, max-age=86400"

	// bimiCSP neuters the logo when a browser opens it directly: it is user
	// content, served from a host of the user's domain. Inline styles stay
	// allowed so that the style attributes of a logo still paint.
	bimiCSP = "default-src 'none'; style-src 'unsafe-inline'; sandbox"
)

// BIMIController serves the files the BIMI records point to, for the domains
// that asked happyDomain to host them.
type BIMIController struct {
	uc happydns.BIMIUsecase
}

// NewBIMIController constructs a BIMIController.
func NewBIMIController(uc happydns.BIMIUsecase) *BIMIController {
	return &BIMIController{uc: uc}
}

// File serves https://bimi.<domain>/bimi/<name>.svg (the logo) and
// https://bimi.<domain>/bimi/<name>.pem (the mark certificate).
//
// The domain is taken from the Host header, the URLs being the ones the BIMI
// record of the domain points to.
//
//	@Summary	BIMI hosted file
//	@Description	Returns the BIMI logo or mark certificate of the domain the request was addressed to.
//	@Tags			bimi
//	@Produce		image/svg+xml
//	@Produce		application/pem-certificate-chain
//	@Param			file	path	string	true	"File name, as found in the BIMI record"
//	@Success		200	{string}	string
//	@Failure		400	{object}	happydns.ErrorResponse
//	@Failure		404	{object}	happydns.ErrorResponse
//	@Router			/bimi/{file} [get]
func (bc *BIMIController) File(c *gin.Context) {
	host := resolveDomain(c)
	if host == "" {
		respondMissingDomain(c)
		return
	}

	name := c.Param("file")

	var (
		body        []byte
		err         error
		contentType string
	)
	switch {
	case strings.HasSuffix(name, ".svg"):
		body, err = bc.uc.Logo(dns.Fqdn(host), name)
		contentType = "image/svg+xml"
	case strings.HasSuffix(name, ".pem"):
		body, err = bc.uc.Certificate(dns.Fqdn(host), name)
		contentType = "application/pem-certificate-chain"
	default:
		err = happydns.ErrNotFound
	}
	if err != nil {
		respondUsecaseError(c, err, "no such BIMI file for this domain")
		return
	}

	c.Header("Cache-Control", bimiCacheControl)
	c.Header("Content-Security-Policy", bimiCSP)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, contentType, body)
}
