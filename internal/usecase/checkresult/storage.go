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

package checkresult

import (
	"time"

	"git.happydns.org/happyDomain/model"
)

// CheckResultStorage defines the storage interface for check results and related data
type CheckResultStorage interface {
	// Check Results
	// ListCheckResults retrieves check results for a specific plugin+target combination
	ListCheckResults(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, limit int) ([]*happydns.CheckResult, error)

	// ListCheckResultsByPlugin retrieves all check results for a plugin across all targets for a user
	ListCheckResultsByPlugin(userId happydns.Identifier, checkName string, limit int) ([]*happydns.CheckResult, error)

	// ListCheckResultsByUser retrieves all check results for a user
	ListCheckResultsByUser(userId happydns.Identifier, limit int) ([]*happydns.CheckResult, error)

	// GetCheckResult retrieves a specific check result by its ID
	GetCheckResult(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, resultId happydns.Identifier) (*happydns.CheckResult, error)

	// CreateCheckResult stores a new check result
	CreateCheckResult(result *happydns.CheckResult) error

	// DeleteCheckResult removes a specific check result
	DeleteCheckResult(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, resultId happydns.Identifier) error

	// DeleteOldCheckResults removes old check results keeping only the most recent N results
	DeleteOldCheckResults(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, keepCount int) error

	// Checker Schedules
	// ListEnabledCheckerSchedules retrieves all enabled schedules (for scheduler)
	ListEnabledCheckerSchedules() ([]*happydns.CheckerSchedule, error)

	// ListCheckerSchedulesByUser retrieves all schedules for a specific user
	ListCheckerSchedulesByUser(userId happydns.Identifier) ([]*happydns.CheckerSchedule, error)

	// ListCheckerSchedulesByTarget retrieves all schedules for a specific target
	ListCheckerSchedulesByTarget(targetType happydns.CheckScopeType, targetId happydns.Identifier) ([]*happydns.CheckerSchedule, error)

	// GetCheckerSchedule retrieves a specific schedule by ID
	GetCheckerSchedule(scheduleId happydns.Identifier) (*happydns.CheckerSchedule, error)

	// CreateCheckerSchedule creates a new check schedule
	CreateCheckerSchedule(schedule *happydns.CheckerSchedule) error

	// UpdateCheckerSchedule updates an existing schedule
	UpdateCheckerSchedule(schedule *happydns.CheckerSchedule) error

	// DeleteCheckerSchedule removes a schedule
	DeleteCheckerSchedule(scheduleId happydns.Identifier) error

	// Check Executions
	// ListActiveCheckExecutions retrieves all executions that are pending or running
	ListActiveCheckExecutions() ([]*happydns.CheckExecution, error)

	// GetCheckExecution retrieves a specific execution by ID
	GetCheckExecution(executionId happydns.Identifier) (*happydns.CheckExecution, error)

	// CreateCheckExecution creates a new check execution record
	CreateCheckExecution(execution *happydns.CheckExecution) error

	// UpdateCheckExecution updates an existing execution record
	UpdateCheckExecution(execution *happydns.CheckExecution) error

	// DeleteCheckExecution removes an execution record
	DeleteCheckExecution(executionId happydns.Identifier) error

	// Scheduler State
	// CheckSchedulerRun marks that the scheduler has run at current time
	CheckSchedulerRun() error

	// LastCheckSchedulerRun retrieves the last time the scheduler ran
	LastCheckSchedulerRun() (*time.Time, error)
}
