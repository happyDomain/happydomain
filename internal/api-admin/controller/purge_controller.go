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
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

type PurgeController struct {
	purgeService happydns.PurgeUsecase
}

func NewPurgeController(purgeService happydns.PurgeUsecase) *PurgeController {
	return &PurgeController{
		purgeService: purgeService,
	}
}

// Purge deletes the selected low value data to free space in an emergency.
//
//	@Summary	Purge low value data
//	@Schemes
//	@Description	Deletes the selected categories of data that the server recreates by itself or whose loss is minor: every checker output (executions, evaluations, observation snapshots and cache, discovery data, keeping check plans and checker options), the expired sessions, and the whole history of sent notifications. With compact, the storage backend then reclaims the freed space.
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		body	body		happydns.PurgeOptions	true	"Categories to purge"
//	@Security	securitydefinitions.basic
//	@Success	200	{object}	happydns.PurgeReport
//	@Failure	400	{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure	500	{object}	happydns.PurgeReport	"Partial report, with the error"
//	@Router		/purge [post]
func (pc *PurgeController) Purge(c *gin.Context) {
	var opts happydns.PurgeOptions
	if err := c.ShouldBindJSON(&opts); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	// A failing category does not stop the others, so the report is sent
	// even on error: it tells what was deleted all the same.
	report, err := pc.purgeService.Purge(opts)
	if err != nil {
		if report == nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}
		log.Printf("Purge: %s", err)
		report.Error = err.Error()
		c.JSON(http.StatusInternalServerError, report)
		return
	}

	c.JSON(http.StatusOK, report)
}
