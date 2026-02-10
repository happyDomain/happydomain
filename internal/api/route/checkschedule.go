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
	"git.happydns.org/happyDomain/model"
)

// DeclareTestScheduleRoutes declares test schedule management routes
func DeclareTestScheduleRoutes(router *gin.RouterGroup, checkerScheduleUC happydns.CheckerScheduleUsecase) {
	sc := controller.NewCheckerScheduleController(checkerScheduleUC)

	schedulesRoutes := router.Group("/plugins/tests/schedules")
	{
		schedulesRoutes.GET("", sc.ListCheckerSchedules)
		schedulesRoutes.POST("", sc.CreateCheckerSchedule)

		scheduleRoutes := schedulesRoutes.Group("/:schedule_id")
		{
			scheduleRoutes.GET("", sc.GetCheckerSchedule)
			scheduleRoutes.PUT("", sc.UpdateCheckerSchedule)
			scheduleRoutes.DELETE("", sc.DeleteCheckerSchedule)
		}
	}
}
