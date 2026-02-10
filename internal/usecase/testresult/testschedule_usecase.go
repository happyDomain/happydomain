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

const (
	// Default test intervals
	DefaultUserTestInterval    = 4 * time.Hour   // 4 hours for user tests
	DefaultDomainTestInterval  = 24 * time.Hour  // 24 hours for domain tests
	DefaultServiceTestInterval = 1 * time.Hour   // 1 hour for service tests
	MinimumTestInterval        = 5 * time.Minute // Minimum interval allowed
)

// TestScheduleUsecase implements business logic for test schedules
type TestScheduleUsecase struct {
	storage TestResultStorage
	options *happydns.Options
}

// NewTestScheduleUsecase creates a new test schedule usecase
func NewTestScheduleUsecase(storage TestResultStorage, options *happydns.Options) *TestScheduleUsecase {
	return &TestScheduleUsecase{
		storage: storage,
		options: options,
	}
}

// ListUserSchedules retrieves all schedules for a specific user
func (u *TestScheduleUsecase) ListUserSchedules(userId happydns.Identifier) ([]*happydns.TestSchedule, error) {
	return u.storage.ListTestSchedulesByUser(userId)
}

// ListSchedulesByTarget retrieves all schedules for a specific target
func (u *TestScheduleUsecase) ListSchedulesByTarget(targetType happydns.TestScopeType, targetId happydns.Identifier) ([]*happydns.TestSchedule, error) {
	return u.storage.ListTestSchedulesByTarget(targetType, targetId)
}

// GetSchedule retrieves a specific schedule by ID
func (u *TestScheduleUsecase) GetSchedule(scheduleId happydns.Identifier) (*happydns.TestSchedule, error) {
	return u.storage.GetTestSchedule(scheduleId)
}

// CreateSchedule creates a new test schedule with validation
func (u *TestScheduleUsecase) CreateSchedule(schedule *happydns.TestSchedule) error {
	// Validate interval
	if schedule.Interval < MinimumTestInterval {
		return fmt.Errorf("test interval must be at least %v", MinimumTestInterval)
	}

	// Set default interval if not specified
	if schedule.Interval == 0 {
		schedule.Interval = u.getDefaultInterval(schedule.TargetType)
	}

	// Calculate next run time
	if schedule.NextRun.IsZero() {
		schedule.NextRun = time.Now().Add(schedule.Interval)
	}

	// Enable by default if not specified
	if !schedule.Enabled {
		schedule.Enabled = true
	}

	return u.storage.CreateTestSchedule(schedule)
}

// UpdateSchedule updates an existing schedule
func (u *TestScheduleUsecase) UpdateSchedule(schedule *happydns.TestSchedule) error {
	// Validate interval
	if schedule.Interval < MinimumTestInterval {
		return fmt.Errorf("test interval must be at least %v", MinimumTestInterval)
	}

	// Get existing schedule to preserve certain fields
	existing, err := u.storage.GetTestSchedule(schedule.Id)
	if err != nil {
		return err
	}

	// Preserve LastRun if not explicitly changed
	if schedule.LastRun == nil {
		schedule.LastRun = existing.LastRun
	}

	// Recalculate next run time if interval changed
	if schedule.Interval != existing.Interval {
		if schedule.LastRun != nil {
			schedule.NextRun = schedule.LastRun.Add(schedule.Interval)
		} else {
			schedule.NextRun = time.Now().Add(schedule.Interval)
		}
	}

	return u.storage.UpdateTestSchedule(schedule)
}

// DeleteSchedule removes a schedule
func (u *TestScheduleUsecase) DeleteSchedule(scheduleId happydns.Identifier) error {
	return u.storage.DeleteTestSchedule(scheduleId)
}

// EnableSchedule enables a schedule
func (u *TestScheduleUsecase) EnableSchedule(scheduleId happydns.Identifier) error {
	schedule, err := u.storage.GetTestSchedule(scheduleId)
	if err != nil {
		return err
	}

	schedule.Enabled = true

	// Reset next run time if it's in the past
	if schedule.NextRun.Before(time.Now()) {
		schedule.NextRun = time.Now().Add(schedule.Interval)
	}

	return u.storage.UpdateTestSchedule(schedule)
}

// DisableSchedule disables a schedule
func (u *TestScheduleUsecase) DisableSchedule(scheduleId happydns.Identifier) error {
	schedule, err := u.storage.GetTestSchedule(scheduleId)
	if err != nil {
		return err
	}

	schedule.Enabled = false
	return u.storage.UpdateTestSchedule(schedule)
}

// UpdateScheduleAfterRun updates a schedule after it has been executed
func (u *TestScheduleUsecase) UpdateScheduleAfterRun(scheduleId happydns.Identifier) error {
	schedule, err := u.storage.GetTestSchedule(scheduleId)
	if err != nil {
		return err
	}

	now := time.Now()
	schedule.LastRun = &now
	schedule.NextRun = now.Add(schedule.Interval)

	return u.storage.UpdateTestSchedule(schedule)
}

// ListDueSchedules retrieves all enabled schedules that are due to run
func (u *TestScheduleUsecase) ListDueSchedules() ([]*happydns.TestSchedule, error) {
	schedules, err := u.storage.ListEnabledTestSchedules()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var dueSchedules []*happydns.TestSchedule

	for _, schedule := range schedules {
		if schedule.Enabled && schedule.NextRun.Before(now) {
			dueSchedules = append(dueSchedules, schedule)
		}
	}

	return dueSchedules, nil
}

// getDefaultInterval returns the default test interval based on target type
func (u *TestScheduleUsecase) getDefaultInterval(targetType happydns.TestScopeType) time.Duration {
	switch targetType {
	case happydns.TestScopeUser:
		return DefaultUserTestInterval
	case happydns.TestScopeDomain:
		return DefaultDomainTestInterval
	case happydns.TestScopeService:
		return DefaultServiceTestInterval
	default:
		return DefaultDomainTestInterval
	}
}

// MergePluginOptions merges plugin options from different scopes
// Priority: schedule options > domain options > user options > global options
func (u *TestScheduleUsecase) MergePluginOptions(
	globalOpts happydns.PluginOptions,
	userOpts happydns.PluginOptions,
	domainOpts happydns.PluginOptions,
	scheduleOpts happydns.PluginOptions,
) happydns.PluginOptions {
	merged := make(happydns.PluginOptions)

	// Start with global options
	for k, v := range globalOpts {
		merged[k] = v
	}

	// Override with user options
	for k, v := range userOpts {
		merged[k] = v
	}

	// Override with domain options
	for k, v := range domainOpts {
		merged[k] = v
	}

	// Override with schedule options (highest priority)
	for k, v := range scheduleOpts {
		merged[k] = v
	}

	return merged
}

// ValidateScheduleOwnership checks if a user owns a schedule
func (u *TestScheduleUsecase) ValidateScheduleOwnership(scheduleId happydns.Identifier, userId happydns.Identifier) error {
	schedule, err := u.storage.GetTestSchedule(scheduleId)
	if err != nil {
		return err
	}

	if !schedule.OwnerId.Equals(userId) {
		return fmt.Errorf("user does not own this schedule")
	}

	return nil
}

// CreateDefaultSchedulesForTarget creates default schedules for a new target
func (u *TestScheduleUsecase) CreateDefaultSchedulesForTarget(
	pluginName string,
	targetType happydns.TestScopeType,
	targetId happydns.Identifier,
	ownerId happydns.Identifier,
	enabled bool,
) error {
	schedule := &happydns.TestSchedule{
		PluginName: pluginName,
		OwnerId:    ownerId,
		TargetType: targetType,
		TargetId:   targetId,
		Interval:   u.getDefaultInterval(targetType),
		Enabled:    enabled,
		NextRun:    time.Now().Add(u.getDefaultInterval(targetType)),
		Options:    make(happydns.PluginOptions),
	}

	return u.CreateSchedule(schedule)
}

// DeleteSchedulesForTarget removes all schedules for a target
func (u *TestScheduleUsecase) DeleteSchedulesForTarget(targetType happydns.TestScopeType, targetId happydns.Identifier) error {
	schedules, err := u.storage.ListTestSchedulesByTarget(targetType, targetId)
	if err != nil {
		return err
	}

	for _, schedule := range schedules {
		if err := u.storage.DeleteTestSchedule(schedule.Id); err != nil {
			return err
		}
	}

	return nil
}
