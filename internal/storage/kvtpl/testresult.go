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

package database

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"git.happydns.org/happyDomain/model"
)

// Test Result storage keys:
// testresult|{plugin-name}|{target-type}|{target-id}|{result-id}
func makeTestResultKey(pluginName string, targetType happydns.TestScopeType, targetId, resultId happydns.Identifier) string {
	return fmt.Sprintf("testresult|%s|%d|%s|%s", pluginName, targetType, targetId.String(), resultId.String())
}

func makeTestResultPrefix(pluginName string, targetType happydns.TestScopeType, targetId happydns.Identifier) string {
	return fmt.Sprintf("testresult|%s|%d|%s|", pluginName, targetType, targetId.String())
}

// ListTestResults retrieves test results for a specific plugin+target combination
func (s *KVStorage) ListTestResults(pluginName string, targetType happydns.TestScopeType, targetId happydns.Identifier, limit int) ([]*happydns.TestResult, error) {
	prefix := makeTestResultPrefix(pluginName, targetType, targetId)
	iter := s.db.Search(prefix)
	defer iter.Release()

	var results []*happydns.TestResult
	for iter.Next() {
		var r happydns.TestResult
		if err := s.db.DecodeData(iter.Value(), &r); err != nil {
			return nil, err
		}
		results = append(results, &r)
	}

	// Sort by ExecutedAt descending (most recent first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].ExecutedAt.After(results[j].ExecutedAt)
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// ListTestResultsByPlugin retrieves all test results for a plugin across all targets for a user
func (s *KVStorage) ListTestResultsByPlugin(userId happydns.Identifier, pluginName string, limit int) ([]*happydns.TestResult, error) {
	prefix := fmt.Sprintf("testresult|%s|", pluginName)
	iter := s.db.Search(prefix)
	defer iter.Release()

	var results []*happydns.TestResult
	for iter.Next() {
		var r happydns.TestResult
		if err := s.db.DecodeData(iter.Value(), &r); err != nil {
			return nil, err
		}
		// Filter by user
		if r.OwnerId.Equals(userId) {
			results = append(results, &r)
		}
	}

	// Sort by ExecutedAt descending (most recent first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].ExecutedAt.After(results[j].ExecutedAt)
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// ListTestResultsByUser retrieves all test results for a user
func (s *KVStorage) ListTestResultsByUser(userId happydns.Identifier, limit int) ([]*happydns.TestResult, error) {
	iter := s.db.Search("testresult|")
	defer iter.Release()

	var results []*happydns.TestResult
	for iter.Next() {
		var r happydns.TestResult
		if err := s.db.DecodeData(iter.Value(), &r); err != nil {
			return nil, err
		}
		// Filter by user
		if r.OwnerId.Equals(userId) {
			results = append(results, &r)
		}
	}

	// Sort by ExecutedAt descending (most recent first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].ExecutedAt.After(results[j].ExecutedAt)
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// GetTestResult retrieves a specific test result by its ID
func (s *KVStorage) GetTestResult(pluginName string, targetType happydns.TestScopeType, targetId happydns.Identifier, resultId happydns.Identifier) (*happydns.TestResult, error) {
	key := makeTestResultKey(pluginName, targetType, targetId, resultId)
	var result happydns.TestResult
	err := s.db.Get(key, &result)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrTestResultNotFound
	}
	return &result, err
}

// CreateTestResult stores a new test result
func (s *KVStorage) CreateTestResult(result *happydns.TestResult) error {
	prefix := makeTestResultPrefix(result.PluginName, result.TestType, result.TargetId)
	key, id, err := s.db.FindIdentifierKey(prefix)
	if err != nil {
		return err
	}

	result.Id = id
	return s.db.Put(key, result)
}

// DeleteTestResult removes a specific test result
func (s *KVStorage) DeleteTestResult(pluginName string, targetType happydns.TestScopeType, targetId happydns.Identifier, resultId happydns.Identifier) error {
	key := makeTestResultKey(pluginName, targetType, targetId, resultId)
	return s.db.Delete(key)
}

// DeleteOldTestResults removes old test results keeping only the most recent N results
func (s *KVStorage) DeleteOldTestResults(pluginName string, targetType happydns.TestScopeType, targetId happydns.Identifier, keepCount int) error {
	results, err := s.ListTestResults(pluginName, targetType, targetId, 0)
	if err != nil {
		return err
	}

	// Results are already sorted by ExecutedAt descending
	// Delete results beyond keepCount
	if len(results) > keepCount {
		for _, r := range results[keepCount:] {
			if err := s.DeleteTestResult(pluginName, targetType, targetId, r.Id); err != nil {
				return err
			}
		}
	}

	return nil
}

// DeleteTestResultsBefore removes all test results with ExecutedAt older than cutoff
func (s *KVStorage) DeleteTestResultsBefore(cutoff time.Time) error {
	iter := s.db.Search("testresult|")
	defer iter.Release()

	var toDelete []string
	for iter.Next() {
		var r happydns.TestResult
		if err := s.db.DecodeData(iter.Value(), &r); err != nil {
			continue
		}
		if r.ExecutedAt.Before(cutoff) {
			toDelete = append(toDelete, string(iter.Key()))
		}
	}

	for _, key := range toDelete {
		if err := s.db.Delete(key); err != nil {
			return err
		}
	}

	return nil
}

// Test Schedule storage keys:
// testschedule|{schedule-id}
// testschedule.byuser|{user-id}|{schedule-id}
// testschedule.bytarget|{target-type}|{target-id}|{schedule-id}

func makeTestScheduleKey(scheduleId happydns.Identifier) string {
	return fmt.Sprintf("testschedule|%s", scheduleId.String())
}

func makeTestScheduleUserIndexKey(userId, scheduleId happydns.Identifier) string {
	return fmt.Sprintf("testschedule.byuser|%s|%s", userId.String(), scheduleId.String())
}

func makeTestScheduleTargetIndexKey(targetType happydns.TestScopeType, targetId, scheduleId happydns.Identifier) string {
	return fmt.Sprintf("testschedule.bytarget|%d|%s|%s", targetType, targetId.String(), scheduleId.String())
}

// ListEnabledTestSchedules retrieves all enabled schedules
func (s *KVStorage) ListEnabledTestSchedules() ([]*happydns.TestSchedule, error) {
	iter := s.db.Search("testschedule|")
	defer iter.Release()

	var schedules []*happydns.TestSchedule
	for iter.Next() {
		var sched happydns.TestSchedule
		if err := s.db.DecodeData(iter.Value(), &sched); err != nil {
			return nil, err
		}
		if sched.Enabled {
			schedules = append(schedules, &sched)
		}
	}

	return schedules, nil
}

// ListTestSchedulesByUser retrieves all schedules for a specific user
func (s *KVStorage) ListTestSchedulesByUser(userId happydns.Identifier) ([]*happydns.TestSchedule, error) {
	prefix := fmt.Sprintf("testschedule.byuser|%s|", userId.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	var schedules []*happydns.TestSchedule
	for iter.Next() {
		// Extract schedule ID from index key
		key := string(iter.Key())
		parts := strings.Split(key, "|")
		if len(parts) < 3 {
			continue
		}

		scheduleId, err := happydns.NewIdentifierFromString(parts[2])
		if err != nil {
			continue
		}

		// Get the actual schedule
		var sched happydns.TestSchedule
		schedKey := makeTestScheduleKey(scheduleId)
		if err := s.db.Get(schedKey, &sched); err != nil {
			continue
		}

		schedules = append(schedules, &sched)
	}

	return schedules, nil
}

// ListTestSchedulesByTarget retrieves all schedules for a specific target
func (s *KVStorage) ListTestSchedulesByTarget(targetType happydns.TestScopeType, targetId happydns.Identifier) ([]*happydns.TestSchedule, error) {
	prefix := fmt.Sprintf("testschedule.bytarget|%d|%s|", targetType, targetId.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	var schedules []*happydns.TestSchedule
	for iter.Next() {
		// Extract schedule ID from index key
		key := string(iter.Key())
		parts := strings.Split(key, "|")
		if len(parts) < 4 {
			continue
		}

		scheduleId, err := happydns.NewIdentifierFromString(parts[3])
		if err != nil {
			continue
		}

		// Get the actual schedule
		var sched happydns.TestSchedule
		schedKey := makeTestScheduleKey(scheduleId)
		if err := s.db.Get(schedKey, &sched); err != nil {
			continue
		}

		schedules = append(schedules, &sched)
	}

	return schedules, nil
}

// GetTestSchedule retrieves a specific schedule by ID
func (s *KVStorage) GetTestSchedule(scheduleId happydns.Identifier) (*happydns.TestSchedule, error) {
	key := makeTestScheduleKey(scheduleId)
	var schedule happydns.TestSchedule
	err := s.db.Get(key, &schedule)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrTestScheduleNotFound
	}
	return &schedule, err
}

// CreateTestSchedule creates a new test schedule
func (s *KVStorage) CreateTestSchedule(schedule *happydns.TestSchedule) error {
	key, id, err := s.db.FindIdentifierKey("testschedule|")
	if err != nil {
		return err
	}

	schedule.Id = id

	// Store the schedule
	if err := s.db.Put(key, schedule); err != nil {
		return err
	}

	// Create indexes
	userIndexKey := makeTestScheduleUserIndexKey(schedule.OwnerId, schedule.Id)
	if err := s.db.Put(userIndexKey, []byte{}); err != nil {
		return err
	}

	targetIndexKey := makeTestScheduleTargetIndexKey(schedule.TargetType, schedule.TargetId, schedule.Id)
	if err := s.db.Put(targetIndexKey, []byte{}); err != nil {
		return err
	}

	return nil
}

// UpdateTestSchedule updates an existing schedule
func (s *KVStorage) UpdateTestSchedule(schedule *happydns.TestSchedule) error {
	key := makeTestScheduleKey(schedule.Id)
	return s.db.Put(key, schedule)
}

// DeleteTestSchedule removes a schedule and its indexes
func (s *KVStorage) DeleteTestSchedule(scheduleId happydns.Identifier) error {
	// Get the schedule first to know what indexes to delete
	schedule, err := s.GetTestSchedule(scheduleId)
	if err != nil {
		return err
	}

	// Delete indexes
	userIndexKey := makeTestScheduleUserIndexKey(schedule.OwnerId, schedule.Id)
	if err := s.db.Delete(userIndexKey); err != nil {
		return err
	}

	targetIndexKey := makeTestScheduleTargetIndexKey(schedule.TargetType, schedule.TargetId, schedule.Id)
	if err := s.db.Delete(targetIndexKey); err != nil {
		return err
	}

	// Delete the schedule itself
	key := makeTestScheduleKey(scheduleId)
	return s.db.Delete(key)
}

// Test Execution storage keys:
// testexec|{execution-id}

func makeTestExecutionKey(executionId happydns.Identifier) string {
	return fmt.Sprintf("testexec|%s", executionId.String())
}

// ListActiveTestExecutions retrieves all executions that are pending or running
func (s *KVStorage) ListActiveTestExecutions() ([]*happydns.TestExecution, error) {
	iter := s.db.Search("testexec|")
	defer iter.Release()

	var executions []*happydns.TestExecution
	for iter.Next() {
		var exec happydns.TestExecution
		if err := s.db.DecodeData(iter.Value(), &exec); err != nil {
			return nil, err
		}
		if exec.Status == happydns.TestExecutionPending || exec.Status == happydns.TestExecutionRunning {
			executions = append(executions, &exec)
		}
	}

	return executions, nil
}

// GetTestExecution retrieves a specific execution by ID
func (s *KVStorage) GetTestExecution(executionId happydns.Identifier) (*happydns.TestExecution, error) {
	key := makeTestExecutionKey(executionId)
	var execution happydns.TestExecution
	err := s.db.Get(key, &execution)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrTestExecutionNotFound
	}
	return &execution, err
}

// CreateTestExecution creates a new test execution record
func (s *KVStorage) CreateTestExecution(execution *happydns.TestExecution) error {
	key, id, err := s.db.FindIdentifierKey("testexec|")
	if err != nil {
		return err
	}

	execution.Id = id
	return s.db.Put(key, execution)
}

// UpdateTestExecution updates an existing execution record
func (s *KVStorage) UpdateTestExecution(execution *happydns.TestExecution) error {
	key := makeTestExecutionKey(execution.Id)
	return s.db.Put(key, execution)
}

// DeleteTestExecution removes an execution record
func (s *KVStorage) DeleteTestExecution(executionId happydns.Identifier) error {
	key := makeTestExecutionKey(executionId)
	return s.db.Delete(key)
}

// DeleteCompletedExecutionsBefore removes completed or failed execution records older than cutoff
func (s *KVStorage) DeleteCompletedExecutionsBefore(cutoff time.Time) error {
	iter := s.db.Search("testexec|")
	defer iter.Release()

	var toDelete []string
	for iter.Next() {
		var exec happydns.TestExecution
		if err := s.db.DecodeData(iter.Value(), &exec); err != nil {
			continue
		}
		if exec.Status != happydns.TestExecutionCompleted && exec.Status != happydns.TestExecutionFailed {
			continue
		}
		if exec.CompletedAt != nil && exec.CompletedAt.Before(cutoff) {
			toDelete = append(toDelete, string(iter.Key()))
		}
	}

	for _, key := range toDelete {
		if err := s.db.Delete(key); err != nil {
			return err
		}
	}

	return nil
}

// Scheduler state storage key:
// testscheduler.lastrun

// TestSchedulerRun marks that the scheduler has run at current time
func (s *KVStorage) TestSchedulerRun() error {
	now := time.Now()
	return s.db.Put("testscheduler.lastrun", &now)
}

// LastTestSchedulerRun retrieves the last time the scheduler ran
func (s *KVStorage) LastTestSchedulerRun() (*time.Time, error) {
	var lastRun time.Time
	err := s.db.Get("testscheduler.lastrun", &lastRun)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &lastRun, nil
}
