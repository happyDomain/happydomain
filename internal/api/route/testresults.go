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

// DeclareScopedTestResultRoutes declares test result routes for a specific scope (domain, zone, or service)
func DeclareScopedTestResultRoutes(
	scopedRouter *gin.RouterGroup,
	testPluginUC happydns.TestPluginUsecase,
	testResultUC happydns.TestResultUsecase,
	testScheduleUC happydns.TestScheduleUsecase,
	testScheduler happydns.TestSchedulerInterface,
	scope happydns.TestScopeType,
) {
	tc := controller.NewTestResultController(
		scope,
		testPluginUC,
		testResultUC,
		testScheduleUC,
		testScheduler,
	)

	// List all available tests with their status
	scopedRouter.GET("/tests", tc.ListAvailableTests)

	// Test-specific routes
	apiTestsRoutes := scopedRouter.Group("/tests/:tname")
	{
		// Get latest results for a test
		apiTestsRoutes.GET("", tc.ListLatestTestResults)

		// Trigger an on-demand test
		apiTestsRoutes.POST("", tc.TriggerTest)

		// Manage test plugin options at this scope
		apiTestsRoutes.GET("/options", tc.GetTestPluginOptions)
		apiTestsRoutes.POST("/options", tc.AddTestPluginOptions)
		apiTestsRoutes.PUT("/options", tc.ChangeTestPluginOptions)

		// Test execution routes
		apiTestExecutionsRoutes := apiTestsRoutes.Group("/executions/:execution_id")
		{
			apiTestExecutionsRoutes.GET("", tc.GetTestExecutionStatus)
		}

		// Test results routes
		apiTestsRoutes.GET("/results", tc.ListTestPluginResults)
		apiTestsRoutes.DELETE("/results", tc.DropTestPluginResults)

		apiTestResultsRoutes := apiTestsRoutes.Group("/results/:result_id")
		{
			apiTestResultsRoutes.GET("", tc.GetTestPluginResult)
			apiTestResultsRoutes.DELETE("", tc.DropTestPluginResult)
		}
	}
}
