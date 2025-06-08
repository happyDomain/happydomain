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
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

type DomainInfoController struct {
	diuService happydns.DomainInfoUsecase
}

func NewDomainInfoController(diuService happydns.DomainInfoUsecase) *DomainInfoController {
	return &DomainInfoController{
		diuService: diuService,
	}
}

// GetDomainInfo retrieves domain's administrative information.
//
//	@Summary	Get domain administrative information
//	@Schemes
//	@Description	Retrieve domain's administrative information.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domain	path		string			true	"Domain name"
//	@Success		200		{object}	happydns.DomainInfo
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domaininfo/{domain} [post]
func (dc *DomainInfoController) GetDomainInfo(c *gin.Context) {
	domain := c.Param("domain")
	if dn, ok := c.Get("domain"); ok {
		domain = dn.(*happydns.Domain).DomainName
	}

	info, err := dc.diuService.GetDomainInfo(happydns.Origin(domain))
	if err != nil {
		if errors.Is(err, happydns.DomainDoesNotExist) {
			c.AbortWithStatusJSON(http.StatusNotFound, happydns.ErrorResponse{Message: err.Error()})
		} else {
			c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, info)
}
