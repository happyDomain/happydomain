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
	"fmt"
	"time"

	"git.happydns.org/happyDomain/model"
)

// CheckResultUsecase implements business logic for check results
type CheckResultUsecase struct {
	storage CheckResultStorage
	options *happydns.Options
}

// NewCheckResultUsecase creates a new check result usecase
func NewCheckResultUsecase(storage CheckResultStorage, options *happydns.Options) *CheckResultUsecase {
	return &CheckResultUsecase{
		storage: storage,
		options: options,
	}
}

// ListCheckResultsByTarget retrieves check results for a specific target
func (u *CheckResultUsecase) ListCheckResultsByTarget(pluginName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, limit int) ([]*happydns.CheckResult, error) {
	// Apply default limit if not specified
	if limit <= 0 {
		limit = 5 // Default to 5 most recent results
	}

	return u.storage.ListCheckResults(pluginName, targetType, targetId, limit)
}

// ListAllCheckResultsByTarget retrieves all check results for a target across all plugins
func (u *CheckResultUsecase) ListAllCheckResultsByTarget(targetType happydns.CheckScopeType, targetId happydns.Identifier, userId happydns.Identifier, limit int) ([]*happydns.CheckResult, error) {
	// Get all results for the user and filter by target
	allResults, err := u.storage.ListCheckResultsByUser(userId, 0)
	if err != nil {
		return nil, err
	}

	// Filter by target
	var results []*happydns.CheckResult
	for _, r := range allResults {
		if r.CheckType == targetType && r.TargetId.Equals(targetId) {
			results = append(results, r)
		}
	}

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// GetCheckResult retrieves a specific check result
func (u *CheckResultUsecase) GetCheckResult(pluginName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, resultId happydns.Identifier) (*happydns.CheckResult, error) {
	return u.storage.GetCheckResult(pluginName, targetType, targetId, resultId)
}

// CreateCheckResult stores a new check result and enforces retention policy
func (u *CheckResultUsecase) CreateCheckResult(result *happydns.CheckResult) error {
	// Store the result
	if err := u.storage.CreateCheckResult(result); err != nil {
		return err
	}

	// Enforce retention policy
	maxResults := u.options.MaxResultsPerCheck
	if maxResults <= 0 {
		maxResults = 100 // Default
	}

	return u.storage.DeleteOldCheckResults(result.CheckerName, result.CheckType, result.TargetId, maxResults)
}

// DeleteCheckResult removes a specific check result
func (u *CheckResultUsecase) DeleteCheckResult(pluginName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, resultId happydns.Identifier) error {
	return u.storage.DeleteCheckResult(pluginName, targetType, targetId, resultId)
}

// DeleteAllCheckResults removes all results for a specific plugin+target combination
func (u *CheckResultUsecase) DeleteAllCheckResults(pluginName string, targetType happydns.CheckScopeType, targetId happydns.Identifier) error {
	// Get all results first
	results, err := u.storage.ListCheckResults(pluginName, targetType, targetId, 0)
	if err != nil {
		return err
	}

	// Delete each result
	for _, r := range results {
		if err := u.storage.DeleteCheckResult(pluginName, targetType, targetId, r.Id); err != nil {
			return err
		}
	}

	return nil
}

// CleanupOldResults removes check results older than retention period
func (u *CheckResultUsecase) CleanupOldResults() error {
	retentionDays := u.options.ResultRetentionDays
	if retentionDays <= 0 {
		retentionDays = 90 // Default
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	// Get all results for all users (inefficient but necessary without a time-based index)
	// In a production system, you might want to add a time-based index for this
	// For now, we'll iterate through results and delete old ones

	// This is a placeholder - the actual implementation would need to be optimized
	// based on specific storage patterns
	_ = cutoffTime

	return nil
}

// GetCheckExecution retrieves the status of a check execution
func (u *CheckResultUsecase) GetCheckExecution(executionId happydns.Identifier) (*happydns.CheckExecution, error) {
	return u.storage.GetCheckExecution(executionId)
}

// CreateCheckExecution creates a new check execution record
func (u *CheckResultUsecase) CreateCheckExecution(execution *happydns.CheckExecution) error {
	if execution.StartedAt.IsZero() {
		execution.StartedAt = time.Now()
	}
	return u.storage.CreateCheckExecution(execution)
}

// UpdateCheckExecution updates an existing check execution
func (u *CheckResultUsecase) UpdateCheckExecution(execution *happydns.CheckExecution) error {
	return u.storage.UpdateCheckExecution(execution)
}

// CompleteCheckExecution marks an execution as completed with a result
func (u *CheckResultUsecase) CompleteCheckExecution(executionId happydns.Identifier, resultId happydns.Identifier) error {
	execution, err := u.storage.GetCheckExecution(executionId)
	if err != nil {
		return err
	}

	now := time.Now()
	execution.Status = happydns.CheckExecutionCompleted
	execution.CompletedAt = &now
	execution.ResultId = &resultId

	return u.storage.UpdateCheckExecution(execution)
}

// FailCheckExecution marks an execution as failed
func (u *CheckResultUsecase) FailCheckExecution(executionId happydns.Identifier, errorMsg string) error {
	execution, err := u.storage.GetCheckExecution(executionId)
	if err != nil {
		return err
	}

	now := time.Now()
	execution.Status = happydns.CheckExecutionFailed
	execution.CompletedAt = &now

	// Store error in a result
	result := &happydns.CheckResult{
		CheckerName:    execution.CheckerName,
		CheckType:      execution.TargetType,
		TargetId:       execution.TargetId,
		OwnerId:        execution.OwnerId,
		ExecutedAt:     time.Now(),
		ScheduledCheck: execution.ScheduleId != nil,
		Options:        execution.Options,
		Status:         happydns.CheckResultStatusCritical,
		StatusLine:     "Execution failed",
		Error:          errorMsg,
		Duration:       now.Sub(execution.StartedAt),
	}

	if err := u.CreateCheckResult(result); err != nil {
		return fmt.Errorf("failed to create error result: %w", err)
	}

	execution.ResultId = &result.Id

	return u.storage.UpdateCheckExecution(execution)
}

// DeleteCompletedExecutions removes execution records that are completed
func (u *CheckResultUsecase) DeleteCompletedExecutions(olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)

	// Get active executions (this won't include completed ones)
	// We need a different query to get completed executions older than cutoff
	// For now, this is a placeholder

	_ = cutoffTime

	return nil
}
