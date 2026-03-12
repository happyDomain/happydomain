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

	"git.happydns.org/happyDomain/checks"
	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

// CheckResultController handles check result operations
type CheckResultController struct {
	scope          happydns.CheckScopeType
	checkerUC      happydns.CheckerUsecase
	checkResultUC  happydns.CheckResultUsecase
	checkScheduler happydns.SchedulerUsecase
}

func NewCheckResultController(
	scope happydns.CheckScopeType,
	checkerUC happydns.CheckerUsecase,
	checkResultUC happydns.CheckResultUsecase,
	checkScheduler happydns.SchedulerUsecase,
) *CheckResultController {
	return &CheckResultController{
		scope:          scope,
		checkerUC:      checkerUC,
		checkResultUC:  checkResultUC,
		checkScheduler: checkScheduler,
	}
}

// getTargetFromContext extracts the target ID from context based on scope
func (tc *CheckResultController) getTargetFromContext(c *gin.Context) (happydns.Identifier, error) {
	switch tc.scope {
	case happydns.CheckScopeUser:
		user := c.MustGet("user").(*happydns.User)
		return user.Id, nil
	case happydns.CheckScopeDomain:
		domain := c.MustGet("domain").(*happydns.Domain)
		return domain.Id, nil
	case happydns.CheckScopeService:
		// Services are stored by ID in context
		serviceID := c.MustGet("serviceid").(happydns.Identifier)
		return serviceID, nil
	default:
		return happydns.Identifier{}, fmt.Errorf("unsupported scope")
	}
}

// ListAvailableChecks lists all available check plugins for the target scope
//
//	@Summary		List available checks
//	@Description	Retrieves all available check plugins for the target scope with their last execution status if enabled
//	@Tags			checks
//	@Produce		json
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			zoneid		path		string	false	"Zone identifier"
//	@Param			subdomain	path		string	false	"Subdomain"
//	@Param			serviceid	path		string	false	"Service identifier"
//	@Success		200		{array}		object	"List of available checks"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks [get]
//	@Router			/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checks [get]
func (tc *CheckResultController) ListAvailableChecks(c *gin.Context) {
	domain := c.MustGet("domain").(*happydns.Domain)
	var service *happydns.Service

	if svc, ok := c.Get("service"); ok {
		service = svc.(*happydns.Service)
	}

	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	checks, err := tc.checkResultUC.ListCheckerStatuses(tc.scope, targetID, domain, service)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, checks)
}

// ListLatestCheckResults retrieves the lacheck check results for a specific plugin
//
//	@Summary		Get lacheck check results
//	@Description	Retrieves the 5 most recent check results for a specific plugin and target
//	@Tags			checks
//	@Produce		json
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			zoneid		path		string	false	"Zone identifier"
//	@Param			subdomain	path		string	false	"Subdomain"
//	@Param			serviceid	path		string	false	"Service identifier"
//	@Param			cname		path		string	true	"Check plugin name"
//	@Success		200		{array}		happydns.CheckResult
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname} [get]
//	@Router			/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checks/{cname} [get]
func (tc *CheckResultController) ListLatestCheckResults(c *gin.Context) {
	checkName := c.Param("cname")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	results, err := tc.checkResultUC.ListCheckResultsByTarget(checkName, tc.scope, targetID, 5)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, results)
}

// TriggerCheck triggers an on-demand check execution
//
//	@Summary		Trigger check execution
//	@Description	Triggers an immediate check execution and returns the execution ID
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			zoneid		path		string	false	"Zone identifier"
//	@Param			subdomain	path		string	false	"Subdomain"
//	@Param			serviceid	path		string	false	"Service identifier"
//	@Param			cname		path		string	true	"Check plugin name"
//	@Param			body	body		object	false	"Optional: Plugin options"
//	@Success		202		{object}	object{execution_id=string}
//	@Failure		400		{object}	happydns.ErrorResponse
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname} [post]
//	@Router			/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checks/{cname} [post]
func (tc *CheckResultController) TriggerCheck(c *gin.Context) {
	user := middleware.MyUser(c)
	checkName := c.Param("cname")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	// Parse run options
	var options happydns.SetCheckerOptionsRequest
	if err = c.ShouldBindJSON(&options); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	// Trigger the test via scheduler (returns error if scheduler is disabled)
	executionID, err := tc.checkScheduler.TriggerOnDemandCheck(checkName, tc.scope, targetID, user.Id, options.Options)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"execution_id": executionID.String()})
}

// GetCheckExecutionStatus retrieves the status of a check execution
//
//	@Summary		Get check execution status
//	@Description	Retrieves the current status of a check execution
//	@Tags			checks
//	@Produce		json
//	@Param			domain			path		string	true	"Domain identifier"
//	@Param			zoneid			path		string	false	"Zone identifier"
//	@Param			subdomain		path		string	false	"Subdomain"
//	@Param			serviceid		path		string	false	"Service identifier"
//	@Param			cname			path		string	true	"Check plugin name"
//	@Param			execution_id	path		string	true	"Execution ID"
//	@Success		200				{object}	happydns.CheckExecution
//	@Failure		404				{object}	happydns.ErrorResponse
//	@Failure		500				{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/executions/{execution_id} [get]
//	@Router			/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checks/{cname}/executions/{execution_id} [get]
func (tc *CheckResultController) GetCheckExecutionStatus(c *gin.Context) {
	executionIDStr := c.Param("execution_id")
	executionID, err := happydns.NewIdentifierFromString(executionIDStr)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid execution ID"))
		return
	}

	execution, err := tc.checkResultUC.GetCheckExecution(executionID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	c.JSON(http.StatusOK, execution)
}

// ListCheckResults lists all results for a check plugin
//
//	@Summary		List check results
//	@Description	Lists all check results for a specific check plugin and target
//	@Tags			checks
//	@Produce		json
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			zoneid		path		string	false	"Zone identifier"
//	@Param			subdomain	path		string	false	"Subdomain"
//	@Param			serviceid	path		string	false	"Service identifier"
//	@Param			cname		path		string	true	"Check plugin name"
//	@Param			limit	query		int		false	"Maximum number of results to return (default: 10)"
//	@Success		200		{array}		happydns.CheckResult
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/results [get]
//	@Router			/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checks/{cname}/results [get]
func (tc *CheckResultController) ListCheckResults(c *gin.Context) {
	checkName := c.Param("cname")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	// Parse limit parameter
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	results, err := tc.checkResultUC.ListCheckResultsByTarget(checkName, tc.scope, targetID, limit)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, results)
}

// DropCheckResults deletes all results for a check plugin
//
//	@Summary		Delete all check results
//	@Description	Deletes all check results for a specific check plugin and target
//	@Tags			checks
//	@Produce		json
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			zoneid		path		string	false	"Zone identifier"
//	@Param			subdomain	path		string	false	"Subdomain"
//	@Param			serviceid	path		string	false	"Service identifier"
//	@Param			cname		path		string	true	"Check plugin name"
//	@Success		204		"No Content"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/results [delete]
//	@Router			/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checks/{cname}/results [delete]
func (tc *CheckResultController) DropCheckResults(c *gin.Context) {
	checkName := c.Param("cname")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	err = tc.checkResultUC.DeleteAllCheckResults(checkName, tc.scope, targetID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetCheckPluginResult retrieves a specific check result
//
//	@Summary		Get check result
//	@Description	Retrieves a specific check result by ID
//	@Tags			checks
//	@Produce		json
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			zoneid		path		string	false	"Zone identifier"
//	@Param			subdomain	path		string	false	"Subdomain"
//	@Param			serviceid	path		string	false	"Service identifier"
//	@Param			cname		path		string	true	"Check plugin name"
//	@Param			result_id	path		string	true	"Result ID"
//	@Success		200			{object}	happydns.CheckResult
//	@Failure		404			{object}	happydns.ErrorResponse
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/results/{result_id} [get]
//	@Router			/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checks/{cname}/results/{result_id} [get]
func (tc *CheckResultController) GetCheckResult(c *gin.Context) {
	checkName := c.Param("cname")
	resultIDStr := c.Param("result_id")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	resultID, err := happydns.NewIdentifierFromString(resultIDStr)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid result ID"))
		return
	}

	result, err := tc.checkResultUC.GetCheckResult(checkName, tc.scope, targetID, resultID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetCheckResultHTMLReport returns the HTML report for a specific check result
//
//	@Summary		Get check result HTML report
//	@Description	Returns the full HTML document generated from the check result's report data. Only available for checkers that implement HTML reporting.
//	@Tags			checks
//	@Produce		html
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			zoneid		path		string	false	"Zone identifier"
//	@Param			subdomain	path		string	false	"Subdomain"
//	@Param			serviceid	path		string	false	"Service identifier"
//	@Param			cname		path		string	true	"Check plugin name"
//	@Param			result_id	path		string	true	"Result ID"
//	@Success		200			{string}	string	"HTML document"
//	@Failure		404			{object}	happydns.ErrorResponse
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/results/{result_id}/report [get]
//	@Router			/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checks/{cname}/results/{result_id}/report [get]
func (tc *CheckResultController) GetCheckResultHTMLReport(c *gin.Context) {
	checkName := c.Param("cname")
	resultIDStr := c.Param("result_id")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	resultID, err := happydns.NewIdentifierFromString(resultIDStr)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid result ID"))
		return
	}

	result, err := tc.checkResultUC.GetCheckResult(checkName, tc.scope, targetID, resultID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	checker, err := tc.checkerUC.GetChecker(checkName)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	raw, err := json.Marshal(result.Report)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	htmlContent, supported, err := checks.GetHTMLReport(checker, json.RawMessage(raw))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	if !supported {
		middleware.ErrorResponse(c, http.StatusNotFound, fmt.Errorf("checker %q does not support HTML reports", checkName))
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
}

// GetCheckResultMetrics returns time-series metrics extracted from check results
//
//	@Summary		Get check result metrics
//	@Description	Returns time-series metrics suitable for charting, extracted from recent check results. Only available for checkers that implement metrics reporting.
//	@Tags			checks
//	@Produce		json
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			zoneid		path		string	false	"Zone identifier"
//	@Param			subdomain	path		string	false	"Subdomain"
//	@Param			serviceid	path		string	false	"Service identifier"
//	@Param			cname		path		string	true	"Check plugin name"
//	@Param			limit		query		int		false	"Maximum number of results to extract metrics from (default: 100)"
//	@Success		200			{object}	happydns.MetricsReport
//	@Failure		404			{object}	happydns.ErrorResponse
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/metrics [get]
//	@Router			/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checks/{cname}/metrics [get]
func (tc *CheckResultController) GetCheckResultMetrics(c *gin.Context) {
	checkName := c.Param("cname")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	checker, err := tc.checkerUC.GetChecker(checkName)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	results, err := tc.checkResultUC.ListCheckResultsByTarget(checkName, tc.scope, targetID, limit)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	report, supported, err := checks.GetMetrics(checker, results)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	if !supported {
		middleware.ErrorResponse(c, http.StatusNotFound, fmt.Errorf("checker %q does not support metrics", checkName))
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetSingleCheckResultMetrics returns metrics extracted from a single check result
//
//	@Summary		Get single check result metrics
//	@Description	Returns metrics extracted from a single check result. Only available for checkers that implement metrics reporting.
//	@Tags			checks
//	@Produce		json
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			zoneid		path		string	false	"Zone identifier"
//	@Param			subdomain	path		string	false	"Subdomain"
//	@Param			serviceid	path		string	false	"Service identifier"
//	@Param			cname		path		string	true	"Check plugin name"
//	@Param			result_id	path		string	true	"Result ID"
//	@Success		200			{object}	happydns.MetricsReport
//	@Failure		404			{object}	happydns.ErrorResponse
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/results/{result_id}/metrics [get]
//	@Router			/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checks/{cname}/results/{result_id}/metrics [get]
func (tc *CheckResultController) GetSingleCheckResultMetrics(c *gin.Context) {
	checkName := c.Param("cname")
	resultIDStr := c.Param("result_id")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	resultID, err := happydns.NewIdentifierFromString(resultIDStr)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid result ID"))
		return
	}

	result, err := tc.checkResultUC.GetCheckResult(checkName, tc.scope, targetID, resultID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	checker, err := tc.checkerUC.GetChecker(checkName)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	report, supported, err := checks.GetMetrics(checker, []*happydns.CheckResult{result})
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	if !supported {
		middleware.ErrorResponse(c, http.StatusNotFound, fmt.Errorf("checker %q does not support metrics", checkName))
		return
	}

	c.JSON(http.StatusOK, report)
}

// DropCheckResult deletes a specific check result
//
//	@Summary		Delete check result
//	@Description	Deletes a specific check result by ID
//	@Tags			checks
//	@Produce		json
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			zoneid		path		string	false	"Zone identifier"
//	@Param			subdomain	path		string	false	"Subdomain"
//	@Param			serviceid	path		string	false	"Service identifier"
//	@Param			cname		path		string	true	"Check plugin name"
//	@Param			result_id	path		string	true	"Result ID"
//	@Success		204			"No Content"
//	@Failure		404			{object}	happydns.ErrorResponse
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/results/{result_id} [delete]
//	@Router			/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checks/{cname}/results/{result_id} [delete]
func (tc *CheckResultController) DropCheckResult(c *gin.Context) {
	checkName := c.Param("cname")
	resultIDStr := c.Param("result_id")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	resultID, err := happydns.NewIdentifierFromString(resultIDStr)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid result ID"))
		return
	}

	err = tc.checkResultUC.DeleteCheckResult(checkName, tc.scope, targetID, resultID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	c.Status(http.StatusNoContent)
}
