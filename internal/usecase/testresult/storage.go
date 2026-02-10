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

package testresult

import (
	"time"

	"git.happydns.org/happyDomain/model"
)

// TestResultStorage defines the storage interface for test results and related data
type TestResultStorage interface {
	// Test Results
	// ListTestResults retrieves test results for a specific plugin+target combination
	ListTestResults(pluginName string, targetType happydns.TestScopeType, targetId happydns.Identifier, limit int) ([]*happydns.TestResult, error)

	// ListTestResultsByPlugin retrieves all test results for a plugin across all targets for a user
	ListTestResultsByPlugin(userId happydns.Identifier, pluginName string, limit int) ([]*happydns.TestResult, error)

	// ListTestResultsByUser retrieves all test results for a user
	ListTestResultsByUser(userId happydns.Identifier, limit int) ([]*happydns.TestResult, error)

	// GetTestResult retrieves a specific test result by its ID
	GetTestResult(pluginName string, targetType happydns.TestScopeType, targetId happydns.Identifier, resultId happydns.Identifier) (*happydns.TestResult, error)

	// CreateTestResult stores a new test result
	CreateTestResult(result *happydns.TestResult) error

	// DeleteTestResult removes a specific test result
	DeleteTestResult(pluginName string, targetType happydns.TestScopeType, targetId happydns.Identifier, resultId happydns.Identifier) error

	// DeleteOldTestResults removes old test results keeping only the most recent N results
	DeleteOldTestResults(pluginName string, targetType happydns.TestScopeType, targetId happydns.Identifier, keepCount int) error

	// DeleteTestResultsBefore removes all test results older than the given time
	DeleteTestResultsBefore(cutoff time.Time) error

	// Test Schedules
	// ListEnabledTestSchedules retrieves all enabled schedules (for scheduler)
	ListEnabledTestSchedules() ([]*happydns.TestSchedule, error)

	// ListTestSchedulesByUser retrieves all schedules for a specific user
	ListTestSchedulesByUser(userId happydns.Identifier) ([]*happydns.TestSchedule, error)

	// ListTestSchedulesByTarget retrieves all schedules for a specific target
	ListTestSchedulesByTarget(targetType happydns.TestScopeType, targetId happydns.Identifier) ([]*happydns.TestSchedule, error)

	// GetTestSchedule retrieves a specific schedule by ID
	GetTestSchedule(scheduleId happydns.Identifier) (*happydns.TestSchedule, error)

	// CreateTestSchedule creates a new test schedule
	CreateTestSchedule(schedule *happydns.TestSchedule) error

	// UpdateTestSchedule updates an existing schedule
	UpdateTestSchedule(schedule *happydns.TestSchedule) error

	// DeleteTestSchedule removes a schedule
	DeleteTestSchedule(scheduleId happydns.Identifier) error

	// Test Executions
	// ListActiveTestExecutions retrieves all executions that are pending or running
	ListActiveTestExecutions() ([]*happydns.TestExecution, error)

	// GetTestExecution retrieves a specific execution by ID
	GetTestExecution(executionId happydns.Identifier) (*happydns.TestExecution, error)

	// CreateTestExecution creates a new test execution record
	CreateTestExecution(execution *happydns.TestExecution) error

	// UpdateTestExecution updates an existing execution record
	UpdateTestExecution(execution *happydns.TestExecution) error

	// DeleteTestExecution removes an execution record
	DeleteTestExecution(executionId happydns.Identifier) error

	// DeleteCompletedExecutionsBefore removes completed or failed execution records older than the given time
	DeleteCompletedExecutionsBefore(cutoff time.Time) error

	// Scheduler State
	// TestSchedulerRun marks that the scheduler has run at current time
	TestSchedulerRun() error

	// LastTestSchedulerRun retrieves the last time the scheduler ran
	LastTestSchedulerRun() (*time.Time, error)
}
