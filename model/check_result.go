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

// CheckScopeType represents the scope level at which a check is performed
type CheckScopeType int

const (
	CheckScopeInstance CheckScopeType = iota
	CheckScopeUser
	CheckScopeDomain
	CheckScopeService
	CheckScopeOnDemand
)

// String returns a string representation of the check scope type
func (t CheckScopeType) String() string {
	switch t {
	case CheckScopeInstance:
		return "instance"
	case CheckScopeUser:
		return "user"
	case CheckScopeDomain:
		return "domain"
	case CheckScopeService:
		return "service"
	case CheckScopeOnDemand:
		return "ondemand"
	default:
		return "unknown"
	}
}

// CheckExecutionStatus represents the current state of a check execution
type CheckExecutionStatus int

const (
	CheckExecutionPending CheckExecutionStatus = iota
	CheckExecutionRunning
	CheckExecutionCompleted
	CheckExecutionFailed
)

// String returns a string representation of the check execution status
func (t CheckExecutionStatus) String() string {
	switch t {
	case CheckExecutionPending:
		return "pending"
	case CheckExecutionRunning:
		return "running"
	case CheckExecutionCompleted:
		return "completed"
	case CheckExecutionFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// CheckResult stores the result of a check execution
type CheckResult struct {
	// Id is the unique identifier for this check result
	Id Identifier `json:"id" swaggertype:"string"`

	// CheckerName identifies which checker was executed
	CheckerName string `json:"checker_name"`

	// CheckType indicates the scope level of the check
	CheckType CheckScopeType `json:"check_type"`

	// TargetId is the identifier of the target (User/Domain/Service)
	TargetId Identifier `json:"target_id" swaggertype:"string"`

	// OwnerId is the owner of the check
	OwnerId Identifier `json:"owner_id" swaggertype:"string"`

	// ExecutedAt is when the check was executed
	ExecutedAt time.Time `json:"executed_at"`

	// ScheduledCheck indicates if this was a scheduled (true) or on-demand (false) check
	ScheduledCheck bool `json:"scheduled_check"`

	// Options contains the merged checker configuration used for this check
	Options CheckerOptions `json:"options,omitempty"`

	// Status is the overall check result status
	Status CheckResultStatus `json:"status"`

	// StatusLine is a summary message of the check result
	StatusLine string `json:"status_line"`

	// Report contains the full check report (checker-specific structure)
	Report any `json:"report,omitempty"`

	// Duration is how long the check took to execute
	Duration time.Duration `json:"duration" swaggertype:"integer"`

	// Error contains any error message if the execution failed
	Error string `json:"error,omitempty"`
}

// CheckExecution tracks an in-progress or completed check execution
type CheckExecution struct {
	// Id is the unique identifier for this execution
	Id Identifier `json:"id" swaggertype:"string"`

	// ScheduleId is the schedule that triggered this execution (nil for on-demand)
	ScheduleId *Identifier `json:"schedule_id,omitempty" swaggertype:"string"`

	// CheckerName identifies which checker is being executed
	CheckerName string `json:"checker_name"`

	// OwnerId is the owner of the check
	OwnerId Identifier `json:"owner_id" swaggertype:"string"`

	// TargetType indicates the scope level of the check
	TargetType CheckScopeType `json:"target_type"`

	// TargetId is the identifier of the target being checked
	TargetId Identifier `json:"target_id" swaggertype:"string"`

	// Status is the current execution status
	Status CheckExecutionStatus `json:"status"`

	// StartedAt is when the execution began
	StartedAt time.Time `json:"started_at"`

	// CompletedAt is when the execution finished (nil if still running)
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// ResultId links to the CheckResult (nil if execution not completed)
	ResultId *Identifier `json:"result_id,omitempty" swaggertype:"string"`

	// Options contains the checker configuration for this execution
	Options CheckerOptions `json:"options,omitempty"`
}

// CheckResultUsecase defines business logic for check results
type CheckResultUsecase interface {
	// ListCheckResultsByTarget retrieves check results for a specific target
	ListCheckResultsByTarget(checkerName string, targetType CheckScopeType, targetId Identifier, limit int) ([]*CheckResult, error)

	// ListAllCheckResultsByTarget retrieves all check results for a target across all checkers
	ListAllCheckResultsByTarget(targetType CheckScopeType, targetId Identifier, userId Identifier, limit int) ([]*CheckResult, error)

	// GetCheckResult retrieves a specific check result
	GetCheckResult(checkName string, targetType CheckScopeType, targetId Identifier, resultId Identifier) (*CheckResult, error)

	// CreateCheckResult stores a new check result and enforces retention policy
	CreateCheckResult(result *CheckResult) error

	// DeleteCheckResult removes a specific check result
	DeleteCheckResult(checkName string, targetType CheckScopeType, targetId Identifier, resultId Identifier) error

	// DeleteAllCheckResults removes all results for a specific checker+target combination
	DeleteAllCheckResults(checkName string, targetType CheckScopeType, targetId Identifier) error

	// GetCheckExecution retrieves the status of a check execution
	GetCheckExecution(executionId Identifier) (*CheckExecution, error)

	// CreateCheckExecution creates a new check execution record
	CreateCheckExecution(execution *CheckExecution) error

	// UpdateCheckExecution updates an existing check execution
	UpdateCheckExecution(execution *CheckExecution) error

	// CompleteCheckExecution marks an execution as completed with a result
	CompleteCheckExecution(executionId Identifier, resultId Identifier) error

	// FailCheckExecution marks an execution as failed
	FailCheckExecution(executionId Identifier, errorMsg string) error

	// GetWorstCheckStatus returns the worst (most critical) status from the most
	// recent result of each checker for a given target. Returns nil if no results exist.
	GetWorstCheckStatus(targetType CheckScopeType, targetId Identifier, userId Identifier) (*CheckResultStatus, error)

	// GetWorstCheckStatusByUser returns a map from target ID string to worst check
	// status for all targets of the given type owned by the user, in a single pass.
	GetWorstCheckStatusByUser(targetType CheckScopeType, userId Identifier) (map[string]*CheckResultStatus, error)
}
