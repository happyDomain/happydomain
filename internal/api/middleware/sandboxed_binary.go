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

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// setSandboxHeaders applies the CSP and nosniff headers shared by every
// endpoint that serves bytes obtained from a third party as if they were
// served from our own origin, so a browser navigating straight to the URL
// cannot be talked into running any of it as a document. extraCSP lets a
// caller widen the otherwise fully locked down default-src (e.g. to allow the
// inline styles and data: images an HTML report needs).
func setSandboxHeaders(c *gin.Context, extraCSP string) {
	c.Header("Content-Security-Policy", "sandbox; default-src 'none';"+extraCSP+" base-uri 'none'; form-action 'none'; frame-ancestors 'self'")
	c.Header("X-Content-Type-Options", "nosniff")
}

// ServeSandboxedImage writes bytes fetched from a third party (an icon, a
// provider logo) as if they were served from our own origin.
func ServeSandboxedImage(c *gin.Context, contentType string, data []byte) {
	c.Header("Cache-Control", "public, max-age=86400")
	setSandboxHeaders(c, "")
	c.Data(http.StatusOK, contentType, data)
}

// ServeSandboxedHTML writes an HTML document built from third party data
// (e.g. a checker observation report) as if it were served from our own
// origin, allowing only the inline styles and data: images such reports need.
func ServeSandboxedHTML(c *gin.Context, html string) {
	setSandboxHeaders(c, " style-src 'unsafe-inline'; img-src 'self' data:;")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
