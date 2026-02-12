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
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

// TestResultController handles test result operations
type TestResultController struct {
	scope          happydns.TestScopeType
	testPluginUC   happydns.TestPluginUsecase
	testResultUC   happydns.TestResultUsecase
	testScheduleUC happydns.TestScheduleUsecase
	testScheduler  happydns.SchedulerUsecase
}

func NewTestResultController(
	scope happydns.TestScopeType,
	testPluginUC happydns.TestPluginUsecase,
	testResultUC happydns.TestResultUsecase,
	testScheduleUC happydns.TestScheduleUsecase,
	testScheduler happydns.SchedulerUsecase,
) *TestResultController {
	return &TestResultController{
		scope:          scope,
		testPluginUC:   testPluginUC,
		testResultUC:   testResultUC,
		testScheduleUC: testScheduleUC,
		testScheduler:  testScheduler,
	}
}

// getTargetFromContext extracts the target ID from context based on scope
func (tc *TestResultController) getTargetFromContext(c *gin.Context) (happydns.Identifier, error) {
	switch tc.scope {
	case happydns.TestScopeUser:
		user := c.MustGet("user").(*happydns.User)
		return user.Id, nil
	case happydns.TestScopeDomain:
		domain := c.MustGet("domain").(*happydns.Domain)
		return domain.Id, nil
	case happydns.TestScopeService:
		// Services are stored by ID in context
		serviceID := c.MustGet("serviceid").(happydns.Identifier)
		return serviceID, nil
	default:
		return happydns.Identifier{}, fmt.Errorf("unsupported scope")
	}
}

// ListAvailableTests lists all available test plugins for the target scope
//
//	@Summary		List available tests
//	@Description	Retrieves all available test plugins for the target scope with their last execution status if enabled
//	@Tags			tests
//	@Produce		json
//	@Param			domain	path		string	true	"Domain identifier"
//	@Success		200		{array}		object	"List of available tests"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/tests [get]
func (tc *TestResultController) ListAvailableTests(c *gin.Context) {
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	// Get all test plugins
	plugins, err := tc.testPluginUC.ListTestPlugins()
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	// Get schedules for this target
	schedules, err := tc.testScheduleUC.ListSchedulesByTarget(tc.scope, targetID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	// Build schedule map
	scheduleMap := make(map[string]*happydns.TestSchedule)
	for _, sched := range schedules {
		scheduleMap[sched.PluginName] = sched
	}

	// Build response with last results
	type TestInfo struct {
		PluginName string                 `json:"plugin_name"`
		Enabled    bool                   `json:"enabled"`
		Schedule   *happydns.TestSchedule `json:"schedule,omitempty"`
		LastResult *happydns.TestResult   `json:"last_result,omitempty"`
	}

	var tests []TestInfo
	for _, plugin := range plugins {
		// Get plugin version info
		versionInfo := plugin.Version()
		availability := versionInfo.AvailableOn

		// Filter plugins by scope
		if tc.scope == happydns.TestScopeDomain && !availability.ApplyToDomain {
			continue
		}
		if tc.scope == happydns.TestScopeService && !availability.ApplyToService {
			continue
		}

		pluginNames := plugin.PluginEnvName()
		if len(pluginNames) == 0 {
			continue
		}

		info := TestInfo{
			PluginName: pluginNames[0],
			Enabled:    true, // enabled by default unless explicitly disabled via a schedule
		}

		// Check if there's a schedule
		if sched, ok := scheduleMap[versionInfo.Name]; ok {
			info.Enabled = sched.Enabled
			info.Schedule = sched

			// Get last result
			results, err := tc.testResultUC.ListTestResultsByTarget(versionInfo.Name, tc.scope, targetID, 1)
			if err == nil && len(results) > 0 {
				info.LastResult = results[0]
			}
		}

		tests = append(tests, info)
	}

	c.JSON(http.StatusOK, tests)
}

// ListLatestTestResults retrieves the latest test results for a specific plugin
//
//	@Summary		Get latest test results
//	@Description	Retrieves the 5 most recent test results for a specific plugin and target
//	@Tags			tests
//	@Produce		json
//	@Param			domain	path		string	true	"Domain identifier"
//	@Param			tname	path		string	true	"Test plugin name"
//	@Success		200		{array}		happydns.TestResult
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/tests/{tname} [get]
func (tc *TestResultController) ListLatestTestResults(c *gin.Context) {
	pluginName := c.Param("tname")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	results, err := tc.testResultUC.ListTestResultsByTarget(pluginName, tc.scope, targetID, 5)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, results)
}

// TriggerTest triggers an on-demand test execution
//
//	@Summary		Trigger test execution
//	@Description	Triggers an immediate test execution and returns the execution ID
//	@Tags			tests
//	@Accept			json
//	@Produce		json
//	@Param			domain	path		string	true	"Domain identifier"
//	@Param			tname	path		string	true	"Test plugin name"
//	@Param			body	body		object	false	"Optional: Plugin options"
//	@Success		202		{object}	object{execution_id=string}
//	@Failure		400		{object}	happydns.ErrorResponse
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/tests/{tname} [post]
func (tc *TestResultController) TriggerTest(c *gin.Context) {
	user := middleware.MyUser(c)
	pluginName := c.Param("tname")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	// Parse run options
	var options happydns.SetPluginOptionsRequest
	if err = c.ShouldBindJSON(&options); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	// Merge options with upper levels (user, domain, service)
	var domainID, serviceID *happydns.Identifier
	switch tc.scope {
	case happydns.TestScopeDomain:
		domainID = &targetID
	case happydns.TestScopeService:
		serviceID = &targetID
	}

	mergedOptions, err := tc.testPluginUC.BuildMergedTestPluginOptions(pluginName, &user.Id, domainID, serviceID, options.Options)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	// Trigger the test via scheduler (returns error if scheduler is disabled)
	executionID, err := tc.testScheduler.TriggerOnDemandTest(pluginName, tc.scope, targetID, user.Id, mergedOptions)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"execution_id": executionID.String()})
}

// GetTestPluginOptions retrieves plugin options for the target scope
//
//	@Summary		Get test plugin options
//	@Description	Retrieves configuration options for a test plugin at the target scope
//	@Tags			tests
//	@Produce		json
//	@Param			domain	path		string	true	"Domain identifier"
//	@Param			tname	path		string	true	"Test plugin name"
//	@Success		200		{object}	happydns.PluginOptions
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/tests/{tname}/options [get]
func (tc *TestResultController) GetTestPluginOptions(c *gin.Context) {
	user := middleware.MyUser(c)
	pluginName := c.Param("tname")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	var domainID, serviceID *happydns.Identifier
	switch tc.scope {
	case happydns.TestScopeDomain:
		domainID = &targetID
	case happydns.TestScopeService:
		serviceID = &targetID
	}

	opts, err := tc.testPluginUC.GetStoredTestPluginOptionsNoDefault(pluginName, &user.Id, domainID, serviceID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, opts)
}

// AddTestPluginOptions adds or overwrites specific options
//
//	@Summary		Add test plugin options
//	@Description	Adds or overwrites specific options for a test plugin at the target scope
//	@Tags			tests
//	@Accept			json
//	@Produce		json
//	@Param			domain	path		string					true	"Domain identifier"
//	@Param			tname	path		string					true	"Test plugin name"
//	@Param			body	body		happydns.PluginOptions	true	"Options to add"
//	@Success		200		{object}	bool
//	@Failure		400		{object}	happydns.ErrorResponse
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/tests/{tname}/options [post]
func (tc *TestResultController) AddTestPluginOptions(c *gin.Context) {
	user := middleware.MyUser(c)
	pluginName := c.Param("tname")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	var options happydns.PluginOptions
	if err = c.ShouldBindJSON(&options); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	var domainID, serviceID *happydns.Identifier
	switch tc.scope {
	case happydns.TestScopeDomain:
		domainID = &targetID
	case happydns.TestScopeService:
		serviceID = &targetID
	}

	err = tc.testPluginUC.OverwriteSomeTestPluginOptions(pluginName, &user.Id, domainID, serviceID, options)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, true)
}

// ChangeTestPluginOptions replaces all options
//
//	@Summary		Replace test plugin options
//	@Description	Replaces all options for a test plugin at the target scope
//	@Tags			tests
//	@Accept			json
//	@Produce		json
//	@Param			domain	path		string					true	"Domain identifier"
//	@Param			tname	path		string					true	"Test plugin name"
//	@Param			body	body		happydns.PluginOptions	true	"New complete options"
//	@Success		200		{object}	bool
//	@Failure		400		{object}	happydns.ErrorResponse
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/tests/{tname}/options [put]
func (tc *TestResultController) ChangeTestPluginOptions(c *gin.Context) {
	user := middleware.MyUser(c)
	pluginName := c.Param("tname")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	var options happydns.PluginOptions
	if err = c.ShouldBindJSON(&options); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	var domainID, serviceID *happydns.Identifier
	switch tc.scope {
	case happydns.TestScopeDomain:
		domainID = &targetID
	case happydns.TestScopeService:
		serviceID = &targetID
	}

	err = tc.testPluginUC.SetTestPluginOptions(pluginName, &user.Id, domainID, serviceID, options)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, true)
}

// GetTestExecutionStatus retrieves the status of a test execution
//
//	@Summary		Get test execution status
//	@Description	Retrieves the current status of a test execution
//	@Tags			tests
//	@Produce		json
//	@Param			domain			path		string	true	"Domain identifier"
//	@Param			tname			path		string	true	"Test plugin name"
//	@Param			execution_id	path		string	true	"Execution ID"
//	@Success		200				{object}	happydns.TestExecution
//	@Failure		404				{object}	happydns.ErrorResponse
//	@Failure		500				{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/tests/{tname}/executions/{execution_id} [get]
func (tc *TestResultController) GetTestExecutionStatus(c *gin.Context) {
	executionIDStr := c.Param("execution_id")
	executionID, err := happydns.NewIdentifierFromString(executionIDStr)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid execution ID"))
		return
	}

	execution, err := tc.testResultUC.GetTestExecution(executionID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	c.JSON(http.StatusOK, execution)
}

// ListTestPluginResults lists all results for a test plugin
//
//	@Summary		List test results
//	@Description	Lists all test results for a specific test plugin and target
//	@Tags			tests
//	@Produce		json
//	@Param			domain	path		string	true	"Domain identifier"
//	@Param			tname	path		string	true	"Test plugin name"
//	@Param			limit	query		int		false	"Maximum number of results to return (default: 10)"
//	@Success		200		{array}		happydns.TestResult
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/tests/{tname}/results [get]
func (tc *TestResultController) ListTestPluginResults(c *gin.Context) {
	pluginName := c.Param("tname")
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

	results, err := tc.testResultUC.ListTestResultsByTarget(pluginName, tc.scope, targetID, limit)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, results)
}

// DropTestPluginResults deletes all results for a test plugin
//
//	@Summary		Delete all test results
//	@Description	Deletes all test results for a specific test plugin and target
//	@Tags			tests
//	@Produce		json
//	@Param			domain	path		string	true	"Domain identifier"
//	@Param			tname	path		string	true	"Test plugin name"
//	@Success		204		"No Content"
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/tests/{tname}/results [delete]
func (tc *TestResultController) DropTestPluginResults(c *gin.Context) {
	pluginName := c.Param("tname")
	targetID, err := tc.getTargetFromContext(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	err = tc.testResultUC.DeleteAllTestResults(pluginName, tc.scope, targetID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetTestPluginResult retrieves a specific test result
//
//	@Summary		Get test result
//	@Description	Retrieves a specific test result by ID
//	@Tags			tests
//	@Produce		json
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			tname		path		string	true	"Test plugin name"
//	@Param			result_id	path		string	true	"Result ID"
//	@Success		200			{object}	happydns.TestResult
//	@Failure		404			{object}	happydns.ErrorResponse
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/tests/{tname}/results/{result_id} [get]
func (tc *TestResultController) GetTestPluginResult(c *gin.Context) {
	pluginName := c.Param("tname")
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

	result, err := tc.testResultUC.GetTestResult(pluginName, tc.scope, targetID, resultID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// DropTestPluginResult deletes a specific test result
//
//	@Summary		Delete test result
//	@Description	Deletes a specific test result by ID
//	@Tags			tests
//	@Produce		json
//	@Param			domain		path		string	true	"Domain identifier"
//	@Param			tname		path		string	true	"Test plugin name"
//	@Param			result_id	path		string	true	"Result ID"
//	@Success		204			"No Content"
//	@Failure		404			{object}	happydns.ErrorResponse
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Router			/domains/{domain}/tests/{tname}/results/{result_id} [delete]
func (tc *TestResultController) DropTestPluginResult(c *gin.Context) {
	pluginName := c.Param("tname")
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

	err = tc.testResultUC.DeleteTestResult(pluginName, tc.scope, targetID, resultID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	c.Status(http.StatusNoContent)
}
