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
	"context"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	checkerPkg "git.happydns.org/happyDomain/internal/checker"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

// CheckerController handles checker-related API endpoints.
type CheckerController struct {
	engine          happydns.CheckerEngine
	OptionsUC       *checkerUC.CheckerOptionsUsecase
	planUC          *checkerUC.CheckPlanUsecase
	statusUC        *checkerUC.CheckStatusUsecase
	plannedProvider checkerUC.PlannedJobProvider
}

// NewCheckerController creates a new CheckerController.
func NewCheckerController(
	engine happydns.CheckerEngine,
	optionsUC *checkerUC.CheckerOptionsUsecase,
	planUC *checkerUC.CheckPlanUsecase,
	statusUC *checkerUC.CheckStatusUsecase,
	plannedProvider checkerUC.PlannedJobProvider,
) *CheckerController {
	return &CheckerController{
		engine:          engine,
		OptionsUC:       optionsUC,
		planUC:          planUC,
		statusUC:        statusUC,
		plannedProvider: plannedProvider,
	}
}

// StatusUC returns the CheckStatusUsecase for use by other controllers.
func (cc *CheckerController) StatusUC() *checkerUC.CheckStatusUsecase {
	return cc.statusUC
}

// targetFromContext builds a CheckTarget from middleware context values.
func targetFromContext(c *gin.Context) happydns.CheckTarget {
	user := middleware.MyUser(c)
	target := happydns.CheckTarget{}
	if user != nil {
		target.UserId = user.Id.String()
	}
	if domain, exists := c.Get("domain"); exists {
		d := domain.(*happydns.Domain)
		target.DomainId = d.Id.String()
	}
	if sid, exists := c.Get("serviceid"); exists {
		id := sid.(happydns.Identifier)
		target.ServiceId = id.String()
		if z, zExists := c.Get("zone"); zExists {
			zone := z.(*happydns.Zone)
			if _, svc := zone.FindService(id); svc != nil {
				target.ServiceType = svc.Type
			}
		}
	}
	return target
}

// --- Global checker routes ---

// ListCheckers returns all registered checker definitions.
//
//	@Summary	List available checkers
//	@Tags		checkers
//	@Produce	json
//	@Success	200	{object}	map[string]checker.CheckerDefinition
//	@Router		/checkers [get]
func (cc *CheckerController) ListCheckers(c *gin.Context) {
	c.JSON(http.StatusOK, checkerPkg.GetCheckers())
}

// GetChecker returns a specific checker definition.
//
//	@Summary	Get a checker definition
//	@Tags		checkers
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Success	200	{object}	checker.CheckerDefinition
//	@Failure	404	{object}	happydns.ErrorResponse
//	@Router		/checkers/{checkerId} [get]
func (cc *CheckerController) GetChecker(c *gin.Context) {
	def, _ := c.Get("checker")
	c.JSON(http.StatusOK, def)
}

// CheckerHandler is a middleware that validates the checkerId path parameter and sets "checker" in context.
func (cc *CheckerController) CheckerHandler(c *gin.Context) {
	checkerID := c.Param("checkerId")
	def := checkerPkg.FindChecker(checkerID)
	if def == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Checker not found"})
		return
	}
	c.Set("checker", def)
	c.Next()
}

// --- Scoped routes (domain/service) ---

// ListAvailableChecks lists all checkers with their latest status for a target.
//
//	@Summary	List available checks with status
//	@Tags		checkers
//	@Produce	json
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{array}	happydns.CheckerStatus
//	@Router		/domains/{domain}/checkers [get]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers [get]
func (cc *CheckerController) ListAvailableChecks(c *gin.Context) {
	target := targetFromContext(c)

	result, err := cc.statusUC.ListCheckerStatuses(target)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// TriggerCheck manually triggers a checker execution.
// By default the check runs asynchronously and returns an Execution (HTTP 202).
// Pass ?sync=true to block until the check completes and return a CheckEvaluation (HTTP 200).
//
//	@Summary	Trigger a manual check
//	@Tags		checkers
//	@Accept		json
//	@Produce	json
//	@Param		checkerId	path	string				true	"Checker ID"
//	@Param		sync		query	bool				false	"Run synchronously"
//	@Param		body		body	happydns.CheckerRunRequest	false	"Run request with options and enabled rules"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{object}	happydns.CheckEvaluation
//	@Success	202	{object}	happydns.Execution
//	@Failure	400	{object}	happydns.ErrorResponse
//	@Router		/domains/{domain}/checkers/{checkerId}/executions [post]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/executions [post]
func (cc *CheckerController) TriggerCheck(c *gin.Context) {
	cname := c.Param("checkerId")

	var req happydns.CheckerRunRequest
	// Body is optional; io.EOF means no body was sent, which is valid (no custom options or rules).
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	target := targetFromContext(c)
	if err := cc.OptionsUC.ValidateOptions(cname, happydns.TargetIdentifier(target.UserId), happydns.TargetIdentifier(target.DomainId), happydns.TargetIdentifier(target.ServiceId), req.Options, true); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	// Build a temporary plan from enabled rules if provided.
	var plan *happydns.CheckPlan
	if len(req.EnabledRules) > 0 {
		plan = &happydns.CheckPlan{
			CheckerID: cname,
			Target:    target,
			Enabled:   req.EnabledRules,
		}
	}

	exec, err := cc.engine.CreateExecution(cname, target, plan)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	if c.Query("sync") == "true" {
		eval, err := cc.engine.RunExecution(c.Request.Context(), exec, plan, req.Options)
		if err != nil {
			middleware.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, eval)
	} else {
		go func() {
			if _, err := cc.engine.RunExecution(context.WithoutCancel(c.Request.Context()), exec, plan, req.Options); err != nil {
				log.Printf("async RunExecution error for checker %q execution %v: %v", cname, exec.Id, err)
			}
		}()
		c.JSON(http.StatusAccepted, exec)
	}
}

// GetExecutionStatus returns the status of an execution.
//
//	@Summary	Get execution status
//	@Tags		checkers
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Param		executionId	path	string	true	"Execution ID"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{object}	happydns.Execution
//	@Failure	404	{object}	happydns.ErrorResponse
//	@Router		/domains/{domain}/checkers/{checkerId}/executions/{executionId} [get]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/executions/{executionId} [get]
func (cc *CheckerController) GetExecutionStatus(c *gin.Context) {
	exec := c.MustGet("execution").(*happydns.Execution)
	c.JSON(http.StatusOK, exec)
}
