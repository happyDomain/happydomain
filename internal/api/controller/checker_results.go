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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	checkerPkg "git.happydns.org/happyDomain/internal/checker"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

// buildReportContext returns a ReportContext for the given primary payload.
// When the engine exposes a related-observation lookup, the context resolves
// Related(key) against discovery storage; otherwise a static context is
// returned.
func (cc *CheckerController) buildReportContext(c *gin.Context, checkerID string, target happydns.CheckTarget, raw json.RawMessage, states []happydns.CheckState) happydns.ReportContext {
	var lookup checkerPkg.RelatedObservationLookup
	if r, ok := cc.engine.(interface {
		RelatedLookup() checkerPkg.RelatedObservationLookup
	}); ok {
		lookup = r.RelatedLookup()
	}
	return checkerPkg.BuildReportContext(c.Request.Context(), checkerID, target, raw, lookup, states)
}

// ExecutionHandler is a middleware that validates the executionId path parameter,
// checks target scope, and sets "execution" in context.
func (cc *CheckerController) ExecutionHandler(c *gin.Context) {
	execID, err := happydns.NewIdentifierFromString(c.Param("executionId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Invalid execution ID"})
		return
	}

	exec, err := cc.statusUC.GetExecution(targetFromContext(c), execID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Execution not found"})
		return
	}

	c.Set("execution", exec)
	c.Next()
}

// ListExecutions returns executions for a checker on a target.
//
//	@Summary	List executions for a checker
//	@Tags		checkers
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Param		limit		query	int		false	"Maximum number of results"
//	@Param		include_planned	query	bool	false	"Include upcoming planned executions from the scheduler"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{array}	happydns.Execution
//	@Router		/domains/{domain}/checkers/{checkerId}/executions [get]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/executions [get]
func (cc *CheckerController) ListExecutions(c *gin.Context) {
	cname := c.Param("checkerId")
	target := targetFromContext(c)

	limit := getLimitParam(c, 0)

	execs, err := cc.statusUC.ListExecutionsByChecker(cname, target, limit)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	if execs == nil {
		execs = []*happydns.Execution{}
	}

	if c.Query("include_planned") == "true" || c.Query("include_planned") == "1" {
		planned := checkerUC.ListPlannedExecutions(cc.plannedProvider, cc.budgetChecker, cname, target)
		execs = append(planned, execs...)
	}

	c.JSON(http.StatusOK, execs)
}

// DeleteExecution deletes an execution record.
//
//	@Summary	Delete an execution
//	@Tags		checkers
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Param		executionId	path	string	true	"Execution ID"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	204
//	@Failure	400	{object}	happydns.ErrorResponse
//	@Failure	404	{object}	happydns.ErrorResponse
//	@Router		/domains/{domain}/checkers/{checkerId}/executions/{executionId} [delete]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/executions/{executionId} [delete]
func (cc *CheckerController) DeleteExecution(c *gin.Context) {
	exec := c.MustGet("execution").(*happydns.Execution)

	if err := cc.statusUC.DeleteExecution(targetFromContext(c), exec.Id); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// DeleteCheckerExecutions deletes all executions for a checker on a target.
//
//	@Summary	Delete all executions for a checker
//	@Tags		checkers
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	204
//	@Failure	400	{object}	happydns.ErrorResponse
//	@Failure	404	{object}	happydns.ErrorResponse
//	@Router		/domains/{domain}/checkers/{checkerId}/executions [delete]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/executions [delete]
func (cc *CheckerController) DeleteCheckerExecutions(c *gin.Context) {
	cname := c.Param("checkerId")
	target := targetFromContext(c)

	if err := cc.statusUC.DeleteExecutionsByChecker(cname, target); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetExecutionObservations returns the observation snapshot for an execution.
//
//	@Summary	Get observations for an execution
//	@Tags		checkers
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Param		executionId	path	string	true	"Execution ID"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{object}	happydns.ObservationSnapshot
//	@Failure	404	{object}	happydns.ErrorResponse
//	@Router		/domains/{domain}/checkers/{checkerId}/executions/{executionId}/observations [get]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/executions/{executionId}/observations [get]
func (cc *CheckerController) GetExecutionObservations(c *gin.Context) {
	exec := c.MustGet("execution").(*happydns.Execution)

	snap, err := cc.statusUC.GetObservationsByExecution(targetFromContext(c), exec.Id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Observations not available"})
		return
	}

	c.JSON(http.StatusOK, snap)
}

// GetExecutionObservation returns a specific observation key from an execution's snapshot.
//
//	@Summary	Get a specific observation for an execution
//	@Tags		checkers
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Param		executionId	path	string	true	"Execution ID"
//	@Param		obsKey		path	string	true	"Observation key"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{object}	any
//	@Failure	404	{object}	happydns.ErrorResponse
//	@Router		/domains/{domain}/checkers/{checkerId}/executions/{executionId}/observations/{obsKey} [get]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/executions/{executionId}/observations/{obsKey} [get]
func (cc *CheckerController) GetExecutionObservation(c *gin.Context) {
	exec := c.MustGet("execution").(*happydns.Execution)
	obsKey := c.Param("obsKey")

	val, err := cc.statusUC.GetSnapshotByExecution(targetFromContext(c), exec.Id, obsKey)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Observation not available"})
		return
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", val)
}

// GetExecutionResults returns the evaluation (per-rule states) for an execution.
//
//	@Summary	Get results for an execution
//	@Tags		checkers
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Param		executionId	path	string	true	"Execution ID"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{object}	happydns.CheckEvaluation
//	@Failure	404	{object}	happydns.ErrorResponse
//	@Router		/domains/{domain}/checkers/{checkerId}/executions/{executionId}/results [get]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/executions/{executionId}/results [get]
func (cc *CheckerController) GetExecutionResults(c *gin.Context) {
	exec := c.MustGet("execution").(*happydns.Execution)

	eval, err := cc.statusUC.GetResultsByExecution(targetFromContext(c), exec.Id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Results not available"})
		return
	}

	c.JSON(http.StatusOK, eval)
}

// GetExecutionResult returns a specific rule's result from an execution.
//
//	@Summary	Get a specific rule result for an execution
//	@Tags		checkers
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Param		executionId	path	string	true	"Execution ID"
//	@Param		ruleName	path	string	true	"Rule name"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{object}	checker.CheckState
//	@Failure	404	{object}	happydns.ErrorResponse
//	@Router		/domains/{domain}/checkers/{checkerId}/executions/{executionId}/results/{ruleName} [get]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/executions/{executionId}/results/{ruleName} [get]
func (cc *CheckerController) GetExecutionResult(c *gin.Context) {
	exec := c.MustGet("execution").(*happydns.Execution)

	eval, err := cc.statusUC.GetResultsByExecution(targetFromContext(c), exec.Id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Results not available"})
		return
	}

	ruleName := c.Param("ruleName")
	for _, state := range eval.States {
		if state.RuleName == ruleName {
			c.JSON(http.StatusOK, state)
			return
		}
	}

	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Rule result not found"})
}

// GetExecutionHTMLReport returns the HTML report for a specific observation of an execution.
//
//	@Summary		Get execution observation HTML report
//	@Description	Returns the full HTML document generated from an observation's data. Only available for observation providers that implement HTML reporting.
//	@Tags			checkers
//	@Produce		html
//	@Param			checkerId	path	string	true	"Checker ID"
//	@Param			executionId	path	string	true	"Execution ID"
//	@Param			obsKey		path	string	true	"Observation key"
//	@Param			domain		path	string	true	"Domain identifier"
//	@Param			zoneid		path	string	true	"Zone identifier"
//	@Param			subdomain	path	string	true	"Subdomain"
//	@Param			serviceid	path	string	true	"Service identifier"
//	@Success		200	{string}	string	"HTML document"
//	@Failure		404	{object}	happydns.ErrorResponse
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checkers/{checkerId}/executions/{executionId}/observations/{obsKey}/report [get]
//	@Router			/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/executions/{executionId}/observations/{obsKey}/report [get]
func (cc *CheckerController) GetExecutionHTMLReport(c *gin.Context) {
	exec := c.MustGet("execution").(*happydns.Execution)
	obsKey := c.Param("obsKey")

	val, err := cc.statusUC.GetSnapshotByExecution(targetFromContext(c), exec.Id, obsKey)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Observation not available"})
		return
	}

	var states []happydns.CheckState
	if eval, err := cc.statusUC.GetResultsByExecution(targetFromContext(c), exec.Id); err == nil && eval != nil {
		states = eval.States
	}
	rc := cc.buildReportContext(c, exec.CheckerID, targetFromContext(c), val, states)
	htmlContent, supported, err := checkerPkg.GetHTMLReport(obsKey, rc)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	if !supported {
		middleware.ErrorResponse(c, http.StatusNotFound, fmt.Errorf("observation %q does not support HTML reports", obsKey))
		return
	}

	c.Header("Content-Security-Policy", "sandbox; default-src 'none'; style-src 'unsafe-inline'; img-src 'self' data:; base-uri 'none'; form-action 'none'; frame-ancestors 'self'")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
}
