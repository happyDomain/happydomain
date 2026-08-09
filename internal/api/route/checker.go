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
	"git.happydns.org/happyDomain/internal/api/middleware"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

// declareCheckerOptionsRoutes registers the options sub-routes on a checker
// group. ownerOnly guards the routes that write options; it is nil on the
// global /api/checkers group, where the options are the caller's own and no
// domain is in the context.
func declareCheckerOptionsRoutes(checkerID *gin.RouterGroup, cc *controller.CheckerController, ownerOnly gin.HandlerFunc) {
	checkerID.GET("/options", cc.GetCheckerOptions)
	checkerID.GET("/options/:optname", cc.GetCheckerOption)

	writeOptions := checkerID.Group("")
	if ownerOnly != nil {
		writeOptions.Use(ownerOnly)
	}
	writeOptions.POST("/options", cc.AddCheckerOptions)
	writeOptions.PUT("/options", cc.ChangeCheckerOptions)
	writeOptions.PUT("/options/:optname", cc.SetCheckerOption)
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

	declareCheckerOptionsRoutes(checkerID, cc, nil)

	return cc
}

// DeclareScopedCheckerRoutes registers checker routes scoped to a domain or service.
// Called for both /api/domains/:domain/checkers and .../services/:serviceid/checkers.
// nc may be nil if the notification system is not configured.
//
// Both mount points sit behind middleware.DomainHandler, which lets in the
// domain owner as well as the users the domain is shared with. Everything that
// only reads is therefore open to both, while the routes that write are kept
// behind middleware.DomainOwnerOnly: checks belong to the domain owner, and an
// invited user must not reschedule, reconfigure or erase them.
func DeclareScopedCheckerRoutes(scopedRouter *gin.RouterGroup, cc *controller.CheckerController, nc *controller.NotificationController) {
	ownerOnly := middleware.DomainOwnerOnly()

	checkers := scopedRouter.Group("/checkers")
	checkers.GET("", cc.ListAvailableChecks)
	checkers.GET("/metrics", cc.GetDomainMetrics)

	checkerID := checkers.Group("/:checkerId")
	checkerID.Use(cc.CheckerHandler)

	declareCheckerOptionsRoutes(checkerID, cc, ownerOnly)

	// Plans (schedules).
	checkerID.GET("/plans", cc.ListCheckPlans)
	checkerID.POST("/plans", ownerOnly, cc.CreateCheckPlan)

	planID := checkerID.Group("/plans/:planId")
	planID.Use(cc.PlanHandler)
	planID.GET("", cc.GetCheckPlan)
	planID.PUT("", ownerOnly, cc.UpdateCheckPlan)
	planID.DELETE("", ownerOnly, cc.DeleteCheckPlan)

	// Per-checker metrics.
	checkerID.GET("/metrics", cc.GetCheckerMetrics)

	// Executions.
	executions := checkerID.Group("/executions")
	executions.GET("", cc.ListExecutions)
	executions.POST("", ownerOnly, cc.TriggerCheck)
	executions.DELETE("", ownerOnly, cc.DeleteCheckerExecutions)

	executionID := executions.Group("/:executionId")
	executionID.Use(cc.ExecutionHandler)
	executionID.GET("", cc.GetExecutionStatus)
	executionID.DELETE("", ownerOnly, cc.DeleteExecution)

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
		checkerID.POST("/acknowledge", ownerOnly, nc.AcknowledgeIssue)
		checkerID.DELETE("/acknowledge", ownerOnly, nc.ClearAcknowledgement)
	}
}
