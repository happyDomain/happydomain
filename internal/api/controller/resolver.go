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

// DNSMsg is the documentation struct corresponding to dns.Msg
type DNSMsg struct {
	// Question is the Question section of the DNS response.
	Question []DNSQuestion

	// Answer is the list of Answer records in the DNS response.
	Answer []interface{} `swaggertype:"object"`

	// Ns is the list of Authoritative records in the DNS response.
	Ns []interface{} `swaggertype:"object"`

	// Extra is the list of extra records in the DNS response.
	Extra []interface{} `swaggertype:"object"`
}

type DNSQuestion struct {
	// Name is the domain name researched.
	Name string

	// Qtype is the type of record researched.
	Qtype uint16

	// Qclass is the class of record researched.
	Qclass uint16
}

// runResolver performs a NS resolution for a given domain, with options.
//
//	@Summary	Perform a DNS resolution.
//	@Schemes
//	@Description	Perform a NS resolution	for a given domain, with options.
//	@Tags			resolver
//	@Accept			json
//	@Produce		json
//	@Param			body	body		happydns.ResolverRequest	true	"Options to the resolution"
//	@Success		200		{object}	DNSMsg
//	@Success		204		{object}	happydns.ErrorResponse	"No content"
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

	c.JSON(http.StatusOK, r)
}
