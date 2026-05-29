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

package route

import (
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/controller"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

// DeclareDomainAvailabilityWatchRoutes registers the availability watchlist
// routes. engine and statusUC may be nil when the checker system is disabled;
// the status and on-demand check endpoints are only registered when the status
// use case is available.
func DeclareDomainAvailabilityWatchRoutes(router *gin.RouterGroup, watchUC happydns.DomainAvailabilityWatchUsecase, engine happydns.CheckerEngine, statusUC *checkerUC.CheckStatusUsecase) {
	wc := controller.NewDomainAvailabilityWatchController(watchUC, engine, statusUC)

	router.GET("/availability", wc.ListDomainAvailabilityWatches)
	router.POST("/availability", wc.AddDomainAvailabilityWatch)
	router.GET("/availability/:watchId", wc.GetDomainAvailabilityWatch)
	router.DELETE("/availability/:watchId", wc.DeleteDomainAvailabilityWatch)

	if statusUC != nil {
		router.GET("/availability/:watchId/status", wc.GetWatchStatus)
		router.POST("/availability/:watchId/check", wc.TriggerWatchCheck)
	}
}
