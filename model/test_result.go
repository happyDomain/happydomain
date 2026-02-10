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

package happydns

import (
	"time"
)

// TestScopeType represents the scope level at which a test is performed
type TestScopeType int

const (
	TestScopeInstance TestScopeType = iota
	TestScopeUser
	TestScopeDomain
	TestScopeService
	TestScopeOnDemand
)

// String returns a string representation of the test scope type
func (t TestScopeType) String() string {
	switch t {
	case TestScopeInstance:
		return "instance"
	case TestScopeUser:
		return "user"
	case TestScopeDomain:
		return "domain"
	case TestScopeService:
		return "service"
	case TestScopeOnDemand:
		return "ondemand"
	default:
		return "unknown"
	}
}

// TestExecutionStatus represents the current state of a test execution
type TestExecutionStatus int

const (
	TestExecutionPending TestExecutionStatus = iota
	TestExecutionRunning
	TestExecutionCompleted
	TestExecutionFailed
)

// String returns a string representation of the test execution status
func (t TestExecutionStatus) String() string {
	switch t {
	case TestExecutionPending:
		return "pending"
	case TestExecutionRunning:
		return "running"
	case TestExecutionCompleted:
		return "completed"
	case TestExecutionFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// TestResult stores the result of a test execution
type TestResult struct {
	// Id is the unique identifier for this test result
	Id Identifier `json:"id" swaggertype:"string"`

	// PluginName identifies which test plugin was executed
	PluginName string `json:"plugin_name"`

	// TestType indicates the scope level of the test
	TestType TestScopeType `json:"test_type"`

	// TargetId is the identifier of the target (User/Domain/Service)
	TargetId Identifier `json:"target_id" swaggertype:"string"`

	// OwnerId is the owner of the test
	OwnerId Identifier `json:"owner_id" swaggertype:"string"`

	// ExecutedAt is when the test was executed
	ExecutedAt time.Time `json:"executed_at"`

	// ScheduledTest indicates if this was a scheduled (true) or on-demand (false) test
	ScheduledTest bool `json:"scheduled_test"`

	// Options contains the merged plugin configuration used for this test
	Options PluginOptions `json:"options,omitempty"`

	// Status is the overall test result status
	Status PluginResultStatus `json:"status"`

	// StatusLine is a summary message of the test result
	StatusLine string `json:"status_line"`

	// Report contains the full test report (plugin-specific structure)
	Report interface{} `json:"report,omitempty"`

	// Duration is how long the test took to execute
	Duration time.Duration `json:"duration" swaggertype:"integer"`

	// Error contains any error message if the execution failed
	Error string `json:"error,omitempty"`
}

// TestExecution tracks an in-progress or completed test execution
type TestExecution struct {
	// Id is the unique identifier for this execution
	Id Identifier `json:"id" swaggertype:"string"`

	// ScheduleId is the schedule that triggered this execution (nil for on-demand)
	ScheduleId *Identifier `json:"schedule_id,omitempty" swaggertype:"string"`

	// PluginName identifies which test plugin is being executed
	PluginName string `json:"plugin_name"`

	// OwnerId is the owner of the test
	OwnerId Identifier `json:"owner_id" swaggertype:"string"`

	// TargetType indicates the scope level of the test
	TargetType TestScopeType `json:"target_type"`

	// TargetId is the identifier of the target being tested
	TargetId Identifier `json:"target_id" swaggertype:"string"`

	// Status is the current execution status
	Status TestExecutionStatus `json:"status"`

	// StartedAt is when the execution began
	StartedAt time.Time `json:"started_at"`

	// CompletedAt is when the execution finished (nil if still running)
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// ResultId links to the TestResult (nil if execution not completed)
	ResultId *Identifier `json:"result_id,omitempty" swaggertype:"string"`

	// Options contains the plugin configuration for this execution
	Options PluginOptions `json:"options,omitempty"`
}

// TestResultUsecase defines business logic for test results
type TestResultUsecase interface {
	// ListTestResultsByTarget retrieves test results for a specific target
	ListTestResultsByTarget(pluginName string, targetType TestScopeType, targetId Identifier, limit int) ([]*TestResult, error)

	// ListAllTestResultsByTarget retrieves all test results for a target across all plugins
	ListAllTestResultsByTarget(targetType TestScopeType, targetId Identifier, userId Identifier, limit int) ([]*TestResult, error)

	// GetTestResult retrieves a specific test result
	GetTestResult(pluginName string, targetType TestScopeType, targetId Identifier, resultId Identifier) (*TestResult, error)

	// CreateTestResult stores a new test result and enforces retention policy
	CreateTestResult(result *TestResult) error

	// DeleteTestResult removes a specific test result
	DeleteTestResult(pluginName string, targetType TestScopeType, targetId Identifier, resultId Identifier) error

	// DeleteAllTestResults removes all results for a specific plugin+target combination
	DeleteAllTestResults(pluginName string, targetType TestScopeType, targetId Identifier) error

	// GetTestExecution retrieves the status of a test execution
	GetTestExecution(executionId Identifier) (*TestExecution, error)

	// CreateTestExecution creates a new test execution record
	CreateTestExecution(execution *TestExecution) error

	// UpdateTestExecution updates an existing test execution
	UpdateTestExecution(execution *TestExecution) error

	// CompleteTestExecution marks an execution as completed with a result
	CompleteTestExecution(executionId Identifier, resultId Identifier) error

	// FailTestExecution marks an execution as failed
	FailTestExecution(executionId Identifier, errorMsg string) error
}
