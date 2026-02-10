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

// Check Result storage keys:
// checkresult|{plugin-name}|{target-type}|{target-id}|{result-id}
func makeCheckResultKey(checkName string, targetType happydns.CheckScopeType, targetId, resultId happydns.Identifier) string {
	return fmt.Sprintf("checkresult|%s|%d|%s|%s", checkName, targetType, targetId.String(), resultId.String())
}

func makeCheckResultPrefix(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier) string {
	return fmt.Sprintf("checkresult|%s|%d|%s|", checkName, targetType, targetId.String())
}

// ListCheckResults retrieves check results for a specific plugin+target combination
func (s *KVStorage) ListCheckResults(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, limit int) ([]*happydns.CheckResult, error) {
	prefix := makeCheckResultPrefix(checkName, targetType, targetId)
	iter := s.db.Search(prefix)
	defer iter.Release()

	var results []*happydns.CheckResult
	for iter.Next() {
		var r happydns.CheckResult
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

// ListCheckResultsByPlugin retrieves all check results for a plugin across all targets for a user
func (s *KVStorage) ListCheckResultsByPlugin(userId happydns.Identifier, checkName string, limit int) ([]*happydns.CheckResult, error) {
	prefix := fmt.Sprintf("checkresult|%s|", checkName)
	iter := s.db.Search(prefix)
	defer iter.Release()

	var results []*happydns.CheckResult
	for iter.Next() {
		var r happydns.CheckResult
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

// ListCheckResultsByUser retrieves all check results for a user
func (s *KVStorage) ListCheckResultsByUser(userId happydns.Identifier, limit int) ([]*happydns.CheckResult, error) {
	iter := s.db.Search("checkresult|")
	defer iter.Release()

	var results []*happydns.CheckResult
	for iter.Next() {
		var r happydns.CheckResult
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

// GetCheckResult retrieves a specific check result by its ID
func (s *KVStorage) GetCheckResult(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, resultId happydns.Identifier) (*happydns.CheckResult, error) {
	key := makeCheckResultKey(checkName, targetType, targetId, resultId)
	var result happydns.CheckResult
	err := s.db.Get(key, &result)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrCheckResultNotFound
	}
	return &result, err
}

// CreateCheckResult stores a new check result
func (s *KVStorage) CreateCheckResult(result *happydns.CheckResult) error {
	prefix := makeCheckResultPrefix(result.CheckerName, result.CheckType, result.TargetId)
	key, id, err := s.db.FindIdentifierKey(prefix)
	if err != nil {
		return err
	}

	result.Id = id
	return s.db.Put(key, result)
}

// DeleteCheckResult removes a specific check result
func (s *KVStorage) DeleteCheckResult(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, resultId happydns.Identifier) error {
	key := makeCheckResultKey(checkName, targetType, targetId, resultId)
	return s.db.Delete(key)
}

// DeleteOldCheckResults removes old check results keeping only the most recent N results
func (s *KVStorage) DeleteOldCheckResults(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, keepCount int) error {
	results, err := s.ListCheckResults(checkName, targetType, targetId, 0)
	if err != nil {
		return err
	}

	// Results are already sorted by ExecutedAt descending
	// Delete results beyond keepCount
	if len(results) > keepCount {
		for _, r := range results[keepCount:] {
			if err := s.DeleteCheckResult(checkName, targetType, targetId, r.Id); err != nil {
				return err
			}
		}
	}

	return nil
}

// Checker Schedule storage keys:
// checkschedule|{schedule-id}
// checkschedule.byuser|{user-id}|{schedule-id}
// checkschedule.bytarget|{target-type}|{target-id}|{schedule-id}

func makeCheckerScheduleKey(scheduleId happydns.Identifier) string {
	return fmt.Sprintf("checkschedule|%s", scheduleId.String())
}

func makeCheckerScheduleUserIndexKey(userId, scheduleId happydns.Identifier) string {
	return fmt.Sprintf("checkschedule.byuser|%s|%s", userId.String(), scheduleId.String())
}

func makeCheckerScheduleTargetIndexKey(targetType happydns.CheckScopeType, targetId, scheduleId happydns.Identifier) string {
	return fmt.Sprintf("checkschedule.bytarget|%d|%s|%s", targetType, targetId.String(), scheduleId.String())
}

// ListEnabledCheckerSchedules retrieves all enabled schedules
func (s *KVStorage) ListEnabledCheckerSchedules() ([]*happydns.CheckerSchedule, error) {
	iter := s.db.Search("checkschedule|")
	defer iter.Release()

	var schedules []*happydns.CheckerSchedule
	for iter.Next() {
		var sched happydns.CheckerSchedule
		if err := s.db.DecodeData(iter.Value(), &sched); err != nil {
			return nil, err
		}
		if sched.Enabled {
			schedules = append(schedules, &sched)
		}
	}

	return schedules, nil
}

// ListCheckerSchedulesByUser retrieves all schedules for a specific user
func (s *KVStorage) ListCheckerSchedulesByUser(userId happydns.Identifier) ([]*happydns.CheckerSchedule, error) {
	prefix := fmt.Sprintf("checkschedule.byuser|%s|", userId.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	var schedules []*happydns.CheckerSchedule
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
		var sched happydns.CheckerSchedule
		schedKey := makeCheckerScheduleKey(scheduleId)
		if err := s.db.Get(schedKey, &sched); err != nil {
			continue
		}

		schedules = append(schedules, &sched)
	}

	return schedules, nil
}

// ListCheckerSchedulesByTarget retrieves all schedules for a specific target
func (s *KVStorage) ListCheckerSchedulesByTarget(targetType happydns.CheckScopeType, targetId happydns.Identifier) ([]*happydns.CheckerSchedule, error) {
	prefix := fmt.Sprintf("checkschedule.bytarget|%d|%s|", targetType, targetId.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	var schedules []*happydns.CheckerSchedule
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
		var sched happydns.CheckerSchedule
		schedKey := makeCheckerScheduleKey(scheduleId)
		if err := s.db.Get(schedKey, &sched); err != nil {
			continue
		}

		schedules = append(schedules, &sched)
	}

	return schedules, nil
}

// GetCheckerSchedule retrieves a specific schedule by ID
func (s *KVStorage) GetCheckerSchedule(scheduleId happydns.Identifier) (*happydns.CheckerSchedule, error) {
	key := makeCheckerScheduleKey(scheduleId)
	var schedule happydns.CheckerSchedule
	err := s.db.Get(key, &schedule)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrCheckScheduleNotFound
	}
	return &schedule, err
}

// CreateCheckerSchedule creates a new check schedule
func (s *KVStorage) CreateCheckerSchedule(schedule *happydns.CheckerSchedule) error {
	key, id, err := s.db.FindIdentifierKey("checkschedule|")
	if err != nil {
		return err
	}

	schedule.Id = id

	// Store the schedule
	if err := s.db.Put(key, schedule); err != nil {
		return err
	}

	// Create indexes
	userIndexKey := makeCheckerScheduleUserIndexKey(schedule.OwnerId, schedule.Id)
	if err := s.db.Put(userIndexKey, []byte{}); err != nil {
		return err
	}

	targetIndexKey := makeCheckerScheduleTargetIndexKey(schedule.TargetType, schedule.TargetId, schedule.Id)
	if err := s.db.Put(targetIndexKey, []byte{}); err != nil {
		return err
	}

	return nil
}

// UpdateCheckerSchedule updates an existing schedule
func (s *KVStorage) UpdateCheckerSchedule(schedule *happydns.CheckerSchedule) error {
	key := makeCheckerScheduleKey(schedule.Id)
	return s.db.Put(key, schedule)
}

// DeleteCheckerSchedule removes a schedule and its indexes
func (s *KVStorage) DeleteCheckerSchedule(scheduleId happydns.Identifier) error {
	// Get the schedule first to know what indexes to delete
	schedule, err := s.GetCheckerSchedule(scheduleId)
	if err != nil {
		return err
	}

	// Delete indexes
	userIndexKey := makeCheckerScheduleUserIndexKey(schedule.OwnerId, schedule.Id)
	if err := s.db.Delete(userIndexKey); err != nil {
		return err
	}

	targetIndexKey := makeCheckerScheduleTargetIndexKey(schedule.TargetType, schedule.TargetId, schedule.Id)
	if err := s.db.Delete(targetIndexKey); err != nil {
		return err
	}

	// Delete the schedule itself
	key := makeCheckerScheduleKey(scheduleId)
	return s.db.Delete(key)
}

// Check Execution storage keys:
// checkexec|{execution-id}

func makeCheckExecutionKey(executionId happydns.Identifier) string {
	return fmt.Sprintf("checkexec|%s", executionId.String())
}

// ListActiveCheckExecutions retrieves all executions that are pending or running
func (s *KVStorage) ListActiveCheckExecutions() ([]*happydns.CheckExecution, error) {
	iter := s.db.Search("checkexec|")
	defer iter.Release()

	var executions []*happydns.CheckExecution
	for iter.Next() {
		var exec happydns.CheckExecution
		if err := s.db.DecodeData(iter.Value(), &exec); err != nil {
			return nil, err
		}
		if exec.Status == happydns.CheckExecutionPending || exec.Status == happydns.CheckExecutionRunning {
			executions = append(executions, &exec)
		}
	}

	return executions, nil
}

// GetCheckExecution retrieves a specific execution by ID
func (s *KVStorage) GetCheckExecution(executionId happydns.Identifier) (*happydns.CheckExecution, error) {
	key := makeCheckExecutionKey(executionId)
	var execution happydns.CheckExecution
	err := s.db.Get(key, &execution)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrCheckExecutionNotFound
	}
	return &execution, err
}

// CreateCheckExecution creates a new check execution record
func (s *KVStorage) CreateCheckExecution(execution *happydns.CheckExecution) error {
	key, id, err := s.db.FindIdentifierKey("checkexec|")
	if err != nil {
		return err
	}

	execution.Id = id
	return s.db.Put(key, execution)
}

// UpdateCheckExecution updates an existing execution record
func (s *KVStorage) UpdateCheckExecution(execution *happydns.CheckExecution) error {
	key := makeCheckExecutionKey(execution.Id)
	return s.db.Put(key, execution)
}

// DeleteCheckExecution removes an execution record
func (s *KVStorage) DeleteCheckExecution(executionId happydns.Identifier) error {
	key := makeCheckExecutionKey(executionId)
	return s.db.Delete(key)
}

// Scheduler state storage key:
// checkscheduler.lastrun

// CheckerSchedulerRun marks that the scheduler has run at current time
func (s *KVStorage) CheckSchedulerRun() error {
	now := time.Now()
	return s.db.Put("checkscheduler.lastrun", &now)
}

// LastCheckSchedulerRun retrieves the last time the scheduler ran
func (s *KVStorage) LastCheckSchedulerRun() (*time.Time, error) {
	var lastRun time.Time
	err := s.db.Get("checkscheduler.lastrun", &lastRun)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &lastRun, nil
}
