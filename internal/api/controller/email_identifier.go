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

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/helpers"
)

type EmailIdentifierController struct{}

func NewEmailIdentifierController() *EmailIdentifierController {
	return &EmailIdentifierController{}
}

type emailIdentifierRequest struct {
	Username string `json:"username" binding:"required"`
}

type emailIdentifierResponse struct {
	Identifier string `json:"identifier"`
}

// ComputeEmailIdentifier hashes an email local-part into the owner name prefix
// expected by OPENPGPKEY and SMIMEA records.
//
// The editors normally compute it in the browser, but the Web Crypto API is
// only exposed in a secure context: an instance served over plain HTTP has
// none, and relies on this endpoint instead.
//
//	@Summary	Compute an OPENPGPKEY/SMIMEA owner name prefix
//	@Tags		domains
//	@Accept		json
//	@Produce	json
//	@Param		domain	path		string					true	"Domain identifier"
//	@Param		body	body		emailIdentifierRequest	true	"Email local-part to hash"
//	@Success	200		{object}	emailIdentifierResponse
//	@Failure	400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Router		/domains/{domain}/email-identifier [post]
func (eic *EmailIdentifierController) ComputeEmailIdentifier(c *gin.Context) {
	var req emailIdentifierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, emailIdentifierResponse{
		Identifier: helpers.EmailIdentifier(req.Username),
	})
}
