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
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/helpers"
)

type RecordController struct{}

func NewRecordController() *RecordController {
	return &RecordController{}
}

// ParseRecords parses records.
//
//	@Summary	Parse given text to retrieve inner records.
//	@Schemes
//	@Description	Parse given text to retrieve inner records.
//	@Tags			records
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			origin			query		int		false	"Origin to use if not provided"
//	@Param			body			body		string		true	"Zone file as text"
//	@Success		200			{object}	happydns.Record
//	@Failure		400			{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401			{object}	happydns.ErrorResponse	"Authentication failure"
//	@Router			/records [post]
func (rc *RecordController) ParseRecords(c *gin.Context) {
	origin := c.DefaultQuery("origin", "")

	var text string
	err := c.ShouldBindJSON(&text)
	if err != nil {
		log.Printf("%s sends invalid JSON record: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	rrs, err := helpers.ParseRecord(text, origin)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, rrs)
}
