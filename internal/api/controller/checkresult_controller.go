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
	scope             happydns.CheckScopeType
	checkerUC         happydns.CheckerUsecase
	checkResultUC     happydns.CheckResultUsecase
	checkerScheduleUC happydns.CheckerScheduleUsecase
	checkScheduler    happydns.SchedulerUsecase
}

func NewCheckResultController(
	scope happydns.CheckScopeType,
	checkerUC happydns.CheckerUsecase,
	checkResultUC happydns.CheckResultUsecase,
	checkerScheduleUC happydns.CheckerScheduleUsecase,
	checkScheduler happydns.SchedulerUsecase,
) *CheckResultController {
	return &CheckResultController{
		scope:             scope,
		checkerUC:         checkerUC,
		checkResultUC:     checkResultUC,
		checkerScheduleUC: checkerScheduleUC,
		checkScheduler:    checkScheduler,
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
//	@Param			domain	path		string	true	"Domain identifier"
//	@Success		200		{array}		object	"List of available checks"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks [get]
func (tc *CheckResultController) ListAvailableChecks(c *gin.Context) {
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	// Get all check plugins
	plugins, err := tc.checkerUC.ListCheckers()
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	// Get schedules for this target
	schedules, err := tc.checkerScheduleUC.ListSchedulesByTarget(tc.scope, targetID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	// Build schedule map
	scheduleMap := make(map[string]*happydns.CheckerSchedule)
	for _, sched := range schedules {
		scheduleMap[sched.CheckerName] = sched
	}

	// Build response with last results
	var checks []happydns.CheckerStatus
	for checkername, check := range *plugins {
		// Filter plugins by scope
		if tc.scope == happydns.CheckScopeDomain && !check.Availability().ApplyToDomain {
			continue
		}
		if tc.scope == happydns.CheckScopeService && !check.Availability().ApplyToService {
			continue
		}

		info := happydns.CheckerStatus{
			CheckerName: checkername,
			Enabled:     true, // enabled by default unless explicitly disabled via a schedule
		}

		// Check if there's a schedule
		if sched, ok := scheduleMap[checkername]; ok {
			info.Enabled = sched.Enabled
			info.Schedule = sched

			// Get last result
			results, err := tc.checkResultUC.ListCheckResultsByTarget(checkername, tc.scope, targetID, 1)
			if err == nil && len(results) > 0 {
				info.LastResult = results[0]
			}
		}

		checks = append(checks, info)
	}

	c.JSON(http.StatusOK, checks)
}

// ListLatestCheckResults retrieves the lacheck check results for a specific plugin
//
//	@Summary		Get lacheck check results
//	@Description	Retrieves the 5 most recent check results for a specific plugin and target
//	@Tags			checks
//	@Produce		json
//	@Param			domain	path		string	true	"Domain identifier"
//	@Param			cname	path		string	true	"Check plugin name"
//	@Success		200		{array}		happydns.CheckResult
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname} [get]
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
//	@Param			domain	path		string	true	"Domain identifier"
//	@Param			cname	path		string	true	"Check plugin name"
//	@Param			body	body		object	false	"Optional: Plugin options"
//	@Success		202		{object}	object{execution_id=string}
//	@Failure		400		{object}	happydns.ErrorResponse
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname} [post]
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

// GetCheckerOptions retrieves plugin options for the target scope
//
//	@Summary		Get check plugin options
//	@Description	Retrieves configuration options for a checker at the target scope
//	@Tags			checks
//	@Produce		json
//	@Param			domain	path		string	true	"Domain identifier"
//	@Param			cname	path		string	true	"Check plugin name"
//	@Success		200		{object}	happydns.CheckerOptions
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/options [get]
func (tc *CheckResultController) GetCheckerOptions(c *gin.Context) {
	user := middleware.MyUser(c)
	checkName := c.Param("cname")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	var domainID, serviceID *happydns.Identifier
	switch tc.scope {
	case happydns.CheckScopeDomain:
		domainID = &targetID
	case happydns.CheckScopeService:
		serviceID = &targetID
	}

	opts, err := tc.checkerUC.GetStoredCheckerOptionsNoDefault(checkName, &user.Id, domainID, serviceID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, opts)
}

// AddCheckerOptions adds or overwrites specific options
//
//	@Summary		Add check plugin options
//	@Description	Adds or overwrites specific options for a check plugin at the target scope
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Param			domain	path		string					true	"Domain identifier"
//	@Param			cname	path		string					true	"Check plugin name"
//	@Param			body	body		happydns.CheckerOptions	true	"Options to add"
//	@Success		200		{object}	bool
//	@Failure		400		{object}	happydns.ErrorResponse
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/options [post]
func (tc *CheckResultController) AddCheckerOptions(c *gin.Context) {
	user := middleware.MyUser(c)
	checkName := c.Param("cname")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	var options happydns.CheckerOptions
	if err = c.ShouldBindJSON(&options); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	var domainID, serviceID *happydns.Identifier
	switch tc.scope {
	case happydns.CheckScopeDomain:
		domainID = &targetID
	case happydns.CheckScopeService:
		serviceID = &targetID
	}

	err = tc.checkerUC.OverwriteSomeCheckerOptions(checkName, &user.Id, domainID, serviceID, options)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, true)
}

// ChangeCheckerOptions replaces all options
//
//	@Summary		Replace check plugin options
//	@Description	Replaces all options for a check plugin at the target scope
//	@Tags			checks
//	@Accept			json
//	@Produce		json
//	@Param			domain	path		string					true	"Domain identifier"
//	@Param			cname	path		string					true	"Check plugin name"
//	@Param			body	body		happydns.CheckerOptions	true	"New complete options"
//	@Success		200		{object}	bool
//	@Failure		400		{object}	happydns.ErrorResponse
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/options [put]
func (tc *CheckResultController) ChangeCheckerOptions(c *gin.Context) {
	user := middleware.MyUser(c)
	checkName := c.Param("cname")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	var options happydns.CheckerOptions
	if err = c.ShouldBindJSON(&options); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	var domainID, serviceID *happydns.Identifier
	switch tc.scope {
	case happydns.CheckScopeDomain:
		domainID = &targetID
	case happydns.CheckScopeService:
		serviceID = &targetID
	}

	err = tc.checkerUC.SetCheckerOptions(checkName, &user.Id, domainID, serviceID, options)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, true)
}

// GetCheckExecutionStatus retrieves the status of a check execution
//
//	@Summary		Get check execution status
//	@Description	Retrieves the current status of a check execution
//	@Tags			checks
//	@Produce		json
//	@Param			domain			path		string	true	"Domain identifier"
//	@Param			cname			path		string	true	"Check plugin name"
//	@Param			execution_id	path		string	true	"Execution ID"
//	@Success		200				{object}	happydns.CheckExecution
//	@Failure		404				{object}	happydns.ErrorResponse
//	@Failure		500				{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/executions/{execution_id} [get]
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
//	@Param			domain	path		string	true	"Domain identifier"
//	@Param			cname	path		string	true	"Check plugin name"
//	@Param			limit	query		int		false	"Maximum number of results to return (default: 10)"
//	@Success		200		{array}		happydns.CheckResult
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/results [get]
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
//	@Param			domain	path		string	true	"Domain identifier"
//	@Param			cname	path		string	true	"Check plugin name"
//	@Success		204		"No Content"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/results [delete]
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
//	@Param			cname		path		string	true	"Check plugin name"
//	@Param			result_id	path		string	true	"Result ID"
//	@Success		200			{object}	happydns.CheckResult
//	@Failure		404			{object}	happydns.ErrorResponse
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/results/{result_id} [get]
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
//	@Param			cname		path		string	true	"Check plugin name"
//	@Param			result_id	path		string	true	"Result ID"
//	@Success		200			{string}	string	"HTML document"
//	@Failure		404			{object}	happydns.ErrorResponse
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/results/{result_id}/report [get]
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

// DropCheckResult deletes a specific check result
//
//	@Summary		Delete check result
//	@Description	Deletes a specific check result by ID
//	@Tags			checks
//	@Produce		json
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			cname		path		string	true	"Check plugin name"
//	@Param			result_id	path		string	true	"Result ID"
//	@Success		204			"No Content"
//	@Failure		404			{object}	happydns.ErrorResponse
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/checks/{cname}/results/{result_id} [delete]
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
