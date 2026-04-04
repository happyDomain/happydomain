// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

type ResolverController struct {
	resolverService happydns.ResolverUsecase
}

func NewResolverController(resolverService happydns.ResolverUsecase) *ResolverController {
	return &ResolverController{
		resolverService: resolverService,
	}
}

// RunResolver performs a NS resolution for a given domain, with options.
//
//	@Summary	Perform a DNS resolution.
//	@Schemes
//	@Description	Perform a NS resolution	for a given domain, with options.
//	@Tags			resolver
//	@Accept			json
//	@Produce		json
//	@Param			body	body		happydns.ResolverRequest	true	"Options to the resolution"
//	@Success		200		{object}	happydns.ResolverResponse
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401		{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403		{object}	happydns.ErrorResponse	"The resolver refused to treat our request"
//	@Failure		404		{object}	happydns.ErrorResponse	"The domain doesn't exist"
//	@Failure		406		{object}	happydns.ErrorResponse	"The resolver returned an error"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/resolver [post]
func (rc *ResolverController) RunResolver(c *gin.Context) {
	var urr happydns.ResolverRequest
	if err := c.ShouldBindJSON(&urr); err != nil {
		log.Printf("%s sends invalid ResolverRequest JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	r, err := rc.resolverService.ResolveQuestion(urr)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, happydns.NewResolverResponseFromMsg(r))
}
