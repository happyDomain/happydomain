// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/favicon"
)

type FaviconController struct {
	faviconService *favicon.FaviconService
}

func NewFaviconController(faviconService *favicon.FaviconService) *FaviconController {
	return &FaviconController{
		faviconService: faviconService,
	}
}

// GetDomainFavicon returns the favicon for a given domain.
//
//	@Summary	Get the favicon for a domain name.
//	@Schemes
//	@Description	Return the favicon for the given domain name.
//	@Tags			favicon
//	@Produce		octet-stream
//	@Param			domain	path		string	true	"The domain name"
//	@Success		200		{file}		binary
//	@Failure		404		{object}	happydns.ErrorResponse	"Favicon not found"
//	@Router			/favicon/{domain} [get]
func (fc *FaviconController) GetDomainFavicon(c *gin.Context) {
	domain := c.Param("domain")

	iconBytes, contentType, err := fc.faviconService.FetchDomain(domain)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, contentType, iconBytes)
}
