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
	"slices"
	"time"

	"git.happydns.org/happyDomain/model"
)

// CheckResultUsecase implements business logic for check results
type CheckResultUsecase struct {
	storage           CheckResultStorage
	options           *happydns.Options
	checkerUC         happydns.CheckerUsecase
	checkerScheduleUC happydns.CheckerScheduleUsecase
}

// NewCheckResultUsecase creates a new check result usecase
func NewCheckResultUsecase(storage CheckResultStorage, options *happydns.Options, checkerUC happydns.CheckerUsecase, checkerScheduleUC happydns.CheckerScheduleUsecase) *CheckResultUsecase {
	return &CheckResultUsecase{
		storage:           storage,
		options:           options,
		checkerUC:         checkerUC,
		checkerScheduleUC: checkerScheduleUC,
	}
}

// ListCheckerStatuses returns all checkers applicable to scope with their schedule
// and most recent result for the given target.
func (u *CheckResultUsecase) ListCheckerStatuses(scope happydns.CheckScopeType, targetID happydns.Identifier, insideScope *happydns.CheckScopeType, insideID *happydns.Identifier, user *happydns.User, domain *happydns.Domain, service *happydns.Service) ([]happydns.CheckerStatus, error) {
	plugins, err := u.checkerUC.ListCheckers()
	if err != nil {
		return nil, err
	}

	// Get schedules for this target
	schedules, err := u.checkerScheduleUC.ListSchedulesByTarget(scope, targetID, insideScope, insideID)
	if err != nil {
		return nil, err
	}

	// Build schedule map
	scheduleMap := make(map[string]*happydns.CheckerSchedule, len(schedules))
	for _, sched := range schedules {
		if sched.OwnerId.Equals(user.Id) {
			scheduleMap[sched.CheckerName] = sched
		}
	}

	// Get service type for LimitToServices filtering
	var serviceType string
	if scope == happydns.CheckScopeService && service != nil {
		serviceType = service.Type
	}

	// Build response with last results
	var statuses []happydns.CheckerStatus
	for checkername, check := range *plugins {
		// Filter plugins by scope
		if scope == happydns.CheckScopeDomain && !check.Availability().ApplyToDomain {
			continue
		}
		if scope == happydns.CheckScopeService && !check.Availability().ApplyToService {
			continue
		}

		// Filter plugins by service type if LimitToServices is set
		if scope == happydns.CheckScopeService && serviceType != "" {
			limitTo := check.Availability().LimitToServices
			if len(limitTo) > 0 && !slices.Contains(limitTo, serviceType) {
				continue
			}
		}

		info := happydns.CheckerStatus{
			CheckerName:   checkername,
			NotDiscovered: true,
		}

		// Check if there's a schedule
		if sched, ok := scheduleMap[checkername]; ok {
			info.Enabled = sched.Enabled
			info.Schedule = sched
			info.NotDiscovered = false

			// Get last result
			results, err := u.ListCheckResultsByTarget(checkername, scope, targetID, insideScope, insideID, user.Id, 1)
			if err == nil && len(results) > 0 {
				info.LastResult = results[0]
			}
		}

		statuses = append(statuses, info)
	}

	return statuses, nil
}

// ListCheckResultsByTarget retrieves check results for a specific target owned by userId
func (u *CheckResultUsecase) ListCheckResultsByTarget(pluginName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, insideScope *happydns.CheckScopeType, insideID *happydns.Identifier, userId happydns.Identifier, limit int) ([]*happydns.CheckResult, error) {
	// Apply default limit if not specified
	if limit <= 0 {
		limit = 5 // Default to 5 most recent results
	}

	results, err := u.storage.ListCheckResults(pluginName, targetType, targetId, limit)
	if err != nil {
		return nil, err
	}

	results = filterResultsByInside(results, insideScope, insideID)

	// Filter by owner
	owned := results[:0]
	for _, r := range results {
		if r.OwnerId.Equals(userId) {
			owned = append(owned, r)
		}
	}

	return owned, nil
}

// ListAllCheckResultsByTarget retrieves all check results for a target across all plugins
func (u *CheckResultUsecase) ListAllCheckResultsByTarget(targetType happydns.CheckScopeType, targetId happydns.Identifier, insideScope *happydns.CheckScopeType, insideID *happydns.Identifier, userId happydns.Identifier, limit int) ([]*happydns.CheckResult, error) {
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

	results = filterResultsByInside(results, insideScope, insideID)

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// filterResultsByInside filters check results by insideScope/insideID.
// If insideScope is nil, only results with nil InsideType are returned.
func filterResultsByInside(results []*happydns.CheckResult, insideScope *happydns.CheckScopeType, insideID *happydns.Identifier) []*happydns.CheckResult {
	filtered := results[:0]
	for _, r := range results {
		if insideScope == nil {
			if r.InsideType == nil {
				filtered = append(filtered, r)
			}
		} else {
			if r.InsideType != nil && *r.InsideType == *insideScope && insideID != nil && r.InsideId != nil && r.InsideId.Equals(*insideID) {
				filtered = append(filtered, r)
			}
		}
	}
	return filtered
}

// GetCheckResult retrieves a specific check result owned by userId
func (u *CheckResultUsecase) GetCheckResult(pluginName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, resultId happydns.Identifier, insideScope *happydns.CheckScopeType, insideID *happydns.Identifier, userId happydns.Identifier) (*happydns.CheckResult, error) {
	result, err := u.storage.GetCheckResult(pluginName, targetType, targetId, resultId)
	if err != nil {
		return nil, err
	}

	// Verify the result belongs to the expected inside scope
	if insideScope == nil {
		if result.InsideType != nil {
			return nil, happydns.ErrNotFound
		}
	} else {
		if result.InsideType == nil || *result.InsideType != *insideScope || insideID == nil || result.InsideId == nil || !result.InsideId.Equals(*insideID) {
			return nil, happydns.ErrNotFound
		}
	}

	// Verify ownership
	if !result.OwnerId.Equals(userId) {
		return nil, happydns.ErrNotFound
	}

	return result, nil
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

// DeleteCheckResult removes a specific check result owned by userId
func (u *CheckResultUsecase) DeleteCheckResult(pluginName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, resultId happydns.Identifier, insideScope *happydns.CheckScopeType, insideID *happydns.Identifier, userId happydns.Identifier) error {
	result, err := u.storage.GetCheckResult(pluginName, targetType, targetId, resultId)
	if err != nil {
		return err
	}
	if !result.OwnerId.Equals(userId) {
		return happydns.ErrNotFound
	}
	return u.storage.DeleteCheckResult(pluginName, targetType, targetId, resultId)
}

// DeleteAllCheckResults removes all results for a specific plugin+target combination owned by userId
func (u *CheckResultUsecase) DeleteAllCheckResults(pluginName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, insideScope *happydns.CheckScopeType, insideID *happydns.Identifier, userId happydns.Identifier) error {
	// Get all results first
	results, err := u.storage.ListCheckResults(pluginName, targetType, targetId, 0)
	if err != nil {
		return err
	}

	results = filterResultsByInside(results, insideScope, insideID)

	// Delete only results owned by the requesting user
	for _, r := range results {
		if !r.OwnerId.Equals(userId) {
			continue
		}
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
		InsideType:     execution.InsideType,
		InsideId:       execution.InsideId,
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

// GetWorstCheckStatus returns the worst (most critical) status from the most
// recent result of each checker for a given target. Returns nil if no results exist.
func (u *CheckResultUsecase) GetWorstCheckStatus(targetType happydns.CheckScopeType, targetId happydns.Identifier, insideScope *happydns.CheckScopeType, insideID *happydns.Identifier, userId happydns.Identifier) (*happydns.CheckResultStatus, error) {
	results, err := u.ListAllCheckResultsByTarget(targetType, targetId, insideScope, insideID, userId, 0)
	if err != nil || len(results) == 0 {
		return nil, err
	}

	// Keep only the latest result per checker
	latest := map[string]*happydns.CheckResult{}
	for _, r := range results {
		if prev, ok := latest[r.CheckerName]; !ok || r.ExecutedAt.After(prev.ExecutedAt) {
			latest[r.CheckerName] = r
		}
	}

	// Find minimum (worst) status among latest results, ignoring Unknown (which
	// means the check couldn't run, not that the domain is in a bad state).
	var worst *happydns.CheckResultStatus
	for _, r := range latest {
		s := r.Status
		if s == happydns.CheckResultStatusUnknown {
			continue
		}
		if worst == nil || s < *worst {
			worst = &s
		}
	}

	return worst, nil
}

// GetWorstCheckStatusByUser fetches all results for the user once and returns
// a map from target ID string to worst (most critical) status per target.
func (u *CheckResultUsecase) GetWorstCheckStatusByUser(targetType happydns.CheckScopeType, userId happydns.Identifier) (map[string]*happydns.CheckResultStatus, error) {
	allResults, err := u.storage.ListCheckResultsByUser(userId, 0)
	if err != nil {
		return nil, err
	}

	type key struct {
		target  string
		checker string
	}
	latest := map[key]*happydns.CheckResult{}
	for _, r := range allResults {
		if r.CheckType != targetType || r.CheckerName == "" {
			continue
		}
		k := key{target: r.TargetId.String(), checker: r.CheckerName}
		if prev, ok := latest[k]; !ok || r.ExecutedAt.After(prev.ExecutedAt) {
			latest[k] = r
		}
	}

	worst := map[string]*happydns.CheckResultStatus{}
	for k, r := range latest {
		s := r.Status
		if prev, ok := worst[k.target]; !ok || s < *prev {
			worst[k.target] = &s
		}
	}

	return worst, nil
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
