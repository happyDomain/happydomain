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

// DeclareScopedCheckResultRoutes declares test result routes for a specific scope (domain, zone, or service)
func DeclareScopedCheckResultRoutes(
	scopedRouter *gin.RouterGroup,
	checkerUC happydns.CheckerUsecase,
	checkResultUC happydns.CheckResultUsecase,
	checkerScheduleUC happydns.CheckerScheduleUsecase,
	checkScheduler happydns.SchedulerUsecase,
	scope happydns.CheckScopeType,
) {
	tc := controller.NewCheckResultController(
		scope,
		checkerUC,
		checkResultUC,
		checkerScheduleUC,
		checkScheduler,
	)

	// List all available tests with their status
	scopedRouter.GET("/checks", tc.ListAvailableChecks)

	// Check-specific routes
	apiChecksRoutes := scopedRouter.Group("/checks/:cname")
	{
		// Get latest results for a test
		apiChecksRoutes.GET("", tc.ListLatestCheckResults)

		// Trigger an on-demand test
		apiChecksRoutes.POST("", tc.TriggerCheck)

		// Manage check options at this scope
		apiChecksRoutes.GET("/options", tc.GetCheckerOptions)
		apiChecksRoutes.POST("/options", tc.AddCheckerOptions)
		apiChecksRoutes.PUT("/options", tc.ChangeCheckerOptions)

		// Check execution routes
		apiCheckExecutionsRoutes := apiChecksRoutes.Group("/executions/:execution_id")
		{
			apiCheckExecutionsRoutes.GET("", tc.GetCheckExecutionStatus)
		}

		// Check results routes
		apiChecksRoutes.GET("/results", tc.ListCheckResults)
		apiChecksRoutes.DELETE("/results", tc.DropCheckResults)

		apiCheckResultsRoutes := apiChecksRoutes.Group("/results/:result_id")
		{
			apiCheckResultsRoutes.GET("", tc.GetCheckResult)
			apiCheckResultsRoutes.DELETE("", tc.DropCheckResult)
			apiCheckResultsRoutes.GET("/report", tc.GetCheckResultHTMLReport)
		}
	}
}
