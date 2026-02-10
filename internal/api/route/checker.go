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
	happydns "git.happydns.org/happyDomain/model"
)

func DeclareCheckersRoutes(router *gin.RouterGroup, checkerUC happydns.CheckerUsecase) *controller.CheckerController {
	tpc := controller.NewCheckerController(checkerUC)

	router.GET("/checks", tpc.ListCheckers)

	apiCheckRoutes := router.Group("/checks/:cname")
	apiCheckRoutes.Use(tpc.CheckerHandler)

	apiCheckRoutes.GET("", tpc.GetCheckerStatus)

	DeclareCheckerOptionsRoutes(apiCheckRoutes, tpc)

	return tpc
}

func DeclareScopedCheckersRoutes(
	scopedRouter *gin.RouterGroup,
	checkerUC happydns.CheckerUsecase,
	checkResultUC happydns.CheckResultUsecase,
	checkScheduler happydns.SchedulerUsecase,
	scope happydns.CheckScopeType,
	tpc *controller.CheckerController,
) {
	tc := controller.NewCheckResultController(
		scope,
		checkerUC,
		checkResultUC,
		checkScheduler,
	)

	// List all available tests with their status
	scopedRouter.GET("/checks", tc.ListAvailableChecks)

	apiChecksRoutes := scopedRouter.Group("/checks/:cname")
	{
		DeclareCheckerOptionsRoutes(apiChecksRoutes, tpc)

		// Get latest results for a test
		apiChecksRoutes.GET("", tc.ListLatestCheckResults)

		// Trigger an on-demand test
		apiChecksRoutes.POST("", tc.TriggerCheck)

		// Check execution routes
		apiCheckExecutionsRoutes := apiChecksRoutes.Group("/executions/:execution_id")
		{
			apiCheckExecutionsRoutes.GET("", tc.GetCheckExecutionStatus)
		}

		DeclareScopedCheckResultRoutes(apiChecksRoutes, tc)
	}
}

func DeclareCheckerOptionsRoutes(apiCheckRoutes *gin.RouterGroup, tpc *controller.CheckerController) {
	apiCheckRoutes.GET("/options", tpc.GetCheckerOptions)
	apiCheckRoutes.POST("/options", tpc.AddCheckerOptions)
	apiCheckRoutes.PUT("/options", tpc.ChangeCheckerOptions)

	apiCheckOptionsRoutes := apiCheckRoutes.Group("/options/:optname")
	apiCheckOptionsRoutes.Use(tpc.CheckerOptionHandler)
	apiCheckOptionsRoutes.GET("", tpc.GetCheckerOption)
	apiCheckOptionsRoutes.PUT("", tpc.SetCheckerOption)
}
