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

// declareCheckerOptionsRoutes registers the options sub-routes on a checker group.
func declareCheckerOptionsRoutes(checkerID *gin.RouterGroup, cc *controller.CheckerController) {
	checkerID.GET("/options", cc.GetCheckerOptions)
	checkerID.POST("/options", cc.AddCheckerOptions)
	checkerID.PUT("/options", cc.ChangeCheckerOptions)
	checkerID.GET("/options/:optname", cc.GetCheckerOption)
	checkerID.PUT("/options/:optname", cc.SetCheckerOption)
}

// DeclareCheckerRoutes registers global checker routes under /api/checkers.
// Returns the controller so it can be reused for scoped routes.
func DeclareCheckerRoutes(
	apiRoutes *gin.RouterGroup,
	engine happydns.CheckerEngine,
	optionsUC *checkerUC.CheckerOptionsUsecase,
	planUC *checkerUC.CheckPlanUsecase,
	statusUC *checkerUC.CheckStatusUsecase,
	plannedProvider checkerUC.PlannedJobProvider,
	budgetChecker checkerUC.BudgetChecker,
	countManualTriggers bool,
) *controller.CheckerController {
	cc := controller.NewCheckerController(engine, optionsUC, planUC, statusUC, plannedProvider, budgetChecker, countManualTriggers)

	// Global: /api/checkers
	checkers := apiRoutes.Group("/checkers")
	checkers.GET("", cc.ListCheckers)
	checkers.GET("/metrics", cc.GetUserMetrics)

	checkerID := checkers.Group("/:checkerId")
	checkerID.Use(cc.CheckerHandler)
	checkerID.GET("", cc.GetChecker)

	declareCheckerOptionsRoutes(checkerID, cc)

	return cc
}

// DeclareScopedCheckerRoutes registers checker routes scoped to a domain or service.
// Called for both /api/domains/:domain/checkers and .../services/:serviceid/checkers.
// nc may be nil if the notification system is not configured.
func DeclareScopedCheckerRoutes(scopedRouter *gin.RouterGroup, cc *controller.CheckerController, nc *controller.NotificationController) {
	checkers := scopedRouter.Group("/checkers")
	checkers.GET("", cc.ListAvailableChecks)
	checkers.GET("/metrics", cc.GetDomainMetrics)

	checkerID := checkers.Group("/:checkerId")
	checkerID.Use(cc.CheckerHandler)

	declareCheckerOptionsRoutes(checkerID, cc)

	// Plans (schedules).
	checkerID.GET("/plans", cc.ListCheckPlans)
	checkerID.POST("/plans", cc.CreateCheckPlan)

	planID := checkerID.Group("/plans/:planId")
	planID.Use(cc.PlanHandler)
	planID.GET("", cc.GetCheckPlan)
	planID.PUT("", cc.UpdateCheckPlan)
	planID.DELETE("", cc.DeleteCheckPlan)

	// Per-checker metrics.
	checkerID.GET("/metrics", cc.GetCheckerMetrics)

	// Executions.
	executions := checkerID.Group("/executions")
	executions.GET("", cc.ListExecutions)
	executions.POST("", cc.TriggerCheck)
	executions.DELETE("", cc.DeleteCheckerExecutions)

	executionID := executions.Group("/:executionId")
	executionID.Use(cc.ExecutionHandler)
	executionID.GET("", cc.GetExecutionStatus)
	executionID.DELETE("", cc.DeleteExecution)

	// Metrics (under execution).
	executionID.GET("/metrics", cc.GetExecutionMetrics)

	// Observations (under execution).
	executionID.GET("/observations", cc.GetExecutionObservations)
	executionID.GET("/observations/:obsKey", cc.GetExecutionObservation)
	executionID.GET("/observations/:obsKey/report", cc.GetExecutionHTMLReport)

	// Results (under execution).
	executionID.GET("/results", cc.GetExecutionResults)
	executionID.GET("/results/:ruleName", cc.GetExecutionResult)

	// Acknowledgement (requires notification system).
	if nc != nil {
		checkerID.POST("/acknowledge", nc.AcknowledgeIssue)
		checkerID.DELETE("/acknowledge", nc.ClearAcknowledgement)
	}
}
