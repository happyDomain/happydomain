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
	"log"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

type DomainLogController struct {
	domainLogService happydns.DomainLogUsecase
}

func NewDomainLogController(domainLogService happydns.DomainLogUsecase) *DomainLogController {
	return &DomainLogController{
		domainLogService: domainLogService,
	}
}

// GetDomainLogs retrieves actions recorded for the domain.
//
//	@Summary	Retrieve Domain actions history.
//	@Schemes
//	@Description	Retrieve information about the actions performed on the domain by users of happyDomain.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			domainId	path	string	true	"Domain identifier"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	[]happydns.DomainLog
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Domain not found"
//	@Router			/domains/{domainId}/logs [get]
func (dlc *DomainLogController) GetDomainLogs(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)

	logs, err := dlc.domainLogService.GetDomainLogs(domain)

	if err != nil {
		log.Printf("%s: An error occurs in GetDomainLogs, when retrieving logs: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Unable to access the domain logs. Please try again later."})
		return
	}

	// Sort by date
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Date.After(logs[j].Date)
	})

	c.JSON(http.StatusOK, logs)
}
