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
	"fmt"
	"time"

	"git.happydns.org/happyDomain/model"
)

// TestResultUsecase implements business logic for test results
type TestResultUsecase struct {
	storage TestResultStorage
	options *happydns.Options
}

// NewTestResultUsecase creates a new test result usecase
func NewTestResultUsecase(storage TestResultStorage, options *happydns.Options) *TestResultUsecase {
	return &TestResultUsecase{
		storage: storage,
		options: options,
	}
}

// ListTestResultsByTarget retrieves test results for a specific target
func (u *TestResultUsecase) ListTestResultsByTarget(pluginName string, targetType happydns.TestScopeType, targetId happydns.Identifier, limit int) ([]*happydns.TestResult, error) {
	// Apply default limit if not specified
	if limit <= 0 {
		limit = 5 // Default to 5 most recent results
	}

	return u.storage.ListTestResults(pluginName, targetType, targetId, limit)
}

// ListAllTestResultsByTarget retrieves all test results for a target across all plugins
func (u *TestResultUsecase) ListAllTestResultsByTarget(targetType happydns.TestScopeType, targetId happydns.Identifier, userId happydns.Identifier, limit int) ([]*happydns.TestResult, error) {
	// Get all results for the user and filter by target
	allResults, err := u.storage.ListTestResultsByUser(userId, 0)
	if err != nil {
		return nil, err
	}

	// Filter by target
	var results []*happydns.TestResult
	for _, r := range allResults {
		if r.TestType == targetType && r.TargetId.Equals(targetId) {
			results = append(results, r)
		}
	}

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// GetTestResult retrieves a specific test result
func (u *TestResultUsecase) GetTestResult(pluginName string, targetType happydns.TestScopeType, targetId happydns.Identifier, resultId happydns.Identifier) (*happydns.TestResult, error) {
	return u.storage.GetTestResult(pluginName, targetType, targetId, resultId)
}

// CreateTestResult stores a new test result and enforces retention policy
func (u *TestResultUsecase) CreateTestResult(result *happydns.TestResult) error {
	// Store the result
	if err := u.storage.CreateTestResult(result); err != nil {
		return err
	}

	// Enforce retention policy
	maxResults := u.options.MaxResultsPerTest
	if maxResults <= 0 {
		maxResults = 100 // Default
	}

	return u.storage.DeleteOldTestResults(result.PluginName, result.TestType, result.TargetId, maxResults)
}

// DeleteTestResult removes a specific test result
func (u *TestResultUsecase) DeleteTestResult(pluginName string, targetType happydns.TestScopeType, targetId happydns.Identifier, resultId happydns.Identifier) error {
	return u.storage.DeleteTestResult(pluginName, targetType, targetId, resultId)
}

// DeleteAllTestResults removes all results for a specific plugin+target combination
func (u *TestResultUsecase) DeleteAllTestResults(pluginName string, targetType happydns.TestScopeType, targetId happydns.Identifier) error {
	// Get all results first
	results, err := u.storage.ListTestResults(pluginName, targetType, targetId, 0)
	if err != nil {
		return err
	}

	// Delete each result
	for _, r := range results {
		if err := u.storage.DeleteTestResult(pluginName, targetType, targetId, r.Id); err != nil {
			return err
		}
	}

	return nil
}

// CleanupOldResults removes test results older than the configured retention period
func (u *TestResultUsecase) CleanupOldResults() error {
	retentionDays := u.options.ResultRetentionDays
	if retentionDays <= 0 {
		retentionDays = 90 // Default
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	return u.storage.DeleteTestResultsBefore(cutoffTime)
}

// GetTestExecution retrieves the status of a test execution
func (u *TestResultUsecase) GetTestExecution(executionId happydns.Identifier) (*happydns.TestExecution, error) {
	return u.storage.GetTestExecution(executionId)
}

// CreateTestExecution creates a new test execution record
func (u *TestResultUsecase) CreateTestExecution(execution *happydns.TestExecution) error {
	if execution.StartedAt.IsZero() {
		execution.StartedAt = time.Now()
	}
	return u.storage.CreateTestExecution(execution)
}

// UpdateTestExecution updates an existing test execution
func (u *TestResultUsecase) UpdateTestExecution(execution *happydns.TestExecution) error {
	return u.storage.UpdateTestExecution(execution)
}

// CompleteTestExecution marks an execution as completed with a result
func (u *TestResultUsecase) CompleteTestExecution(executionId happydns.Identifier, resultId happydns.Identifier) error {
	execution, err := u.storage.GetTestExecution(executionId)
	if err != nil {
		return err
	}

	now := time.Now()
	execution.Status = happydns.TestExecutionCompleted
	execution.CompletedAt = &now
	execution.ResultId = &resultId

	return u.storage.UpdateTestExecution(execution)
}

// FailTestExecution marks an execution as failed
func (u *TestResultUsecase) FailTestExecution(executionId happydns.Identifier, errorMsg string) error {
	execution, err := u.storage.GetTestExecution(executionId)
	if err != nil {
		return err
	}

	now := time.Now()
	execution.Status = happydns.TestExecutionFailed
	execution.CompletedAt = &now

	// Store error in a result
	result := &happydns.TestResult{
		PluginName:    execution.PluginName,
		TestType:      execution.TargetType,
		TargetId:      execution.TargetId,
		OwnerId:       execution.OwnerId,
		ExecutedAt:    time.Now(),
		ScheduledTest: execution.ScheduleId != nil,
		Options:       execution.Options,
		Status:        happydns.PluginResultStatusKO,
		StatusLine:    "Execution failed",
		Error:         errorMsg,
		Duration:      now.Sub(execution.StartedAt),
	}

	if err := u.CreateTestResult(result); err != nil {
		return fmt.Errorf("failed to create error result: %w", err)
	}

	execution.ResultId = &result.Id

	return u.storage.UpdateTestExecution(execution)
}

// DeleteCompletedExecutions removes completed or failed execution records older than olderThan
func (u *TestResultUsecase) DeleteCompletedExecutions(olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)
	return u.storage.DeleteCompletedExecutionsBefore(cutoffTime)
}
