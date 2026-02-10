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
	"sort"
	"time"

	"git.happydns.org/happyDomain/model"
)

const (
	// Default check intervals
	DefaultUserCheckInterval    = 4 * time.Hour   // 4 hours for user checks
	DefaultDomainCheckInterval  = 24 * time.Hour  // 24 hours for domain checks
	DefaultServiceCheckInterval = 1 * time.Hour   // 1 hour for service checks
	MinimumCheckInterval        = 5 * time.Minute // Minimum interval allowed
)

// CheckScheduleUsecase implements business logic for check schedules
type CheckScheduleUsecase struct {
	storage        CheckResultStorage
	options        *happydns.Options
	checkerUsecase happydns.CheckerUsecase
}

// NewCheckScheduleUsecase creates a new check schedule usecase
func NewCheckScheduleUsecase(storage CheckResultStorage, options *happydns.Options, checkerUsecase happydns.CheckerUsecase) *CheckScheduleUsecase {
	return &CheckScheduleUsecase{
		storage:        storage,
		options:        options,
		checkerUsecase: checkerUsecase,
	}
}

// ListUserSchedules retrieves all schedules for a specific user
func (u *CheckScheduleUsecase) ListUserSchedules(userId happydns.Identifier) ([]*happydns.CheckerSchedule, error) {
	return u.storage.ListCheckerSchedulesByUser(userId)
}

// ListSchedulesByTarget retrieves all schedules for a specific target
func (u *CheckScheduleUsecase) ListSchedulesByTarget(targetType happydns.CheckScopeType, targetId happydns.Identifier) ([]*happydns.CheckerSchedule, error) {
	return u.storage.ListCheckerSchedulesByTarget(targetType, targetId)
}

// GetSchedule retrieves a specific schedule by ID
func (u *CheckScheduleUsecase) GetSchedule(scheduleId happydns.Identifier) (*happydns.CheckerSchedule, error) {
	return u.storage.GetCheckerSchedule(scheduleId)
}

// CreateSchedule creates a new check schedule with validation
func (u *CheckScheduleUsecase) CreateSchedule(schedule *happydns.CheckerSchedule) error {
	// Set default interval if not specified
	if schedule.Interval == 0 {
		schedule.Interval = u.getDefaultInterval(schedule.CheckerName, schedule.TargetType)
	}

	// Validate interval against per-checker and global bounds
	if err := u.validateInterval(schedule.CheckerName, &schedule.Interval); err != nil {
		return err
	}

	// Calculate next run time
	if schedule.NextRun.IsZero() {
		schedule.NextRun = time.Now().Add(schedule.Interval)
	}

	// Enable by default if not specified
	if !schedule.Enabled {
		schedule.Enabled = true
	}

	return u.storage.CreateCheckerSchedule(schedule)
}

// UpdateSchedule updates an existing schedule
func (u *CheckScheduleUsecase) UpdateSchedule(schedule *happydns.CheckerSchedule) error {
	// Validate interval against per-checker and global bounds
	if err := u.validateInterval(schedule.CheckerName, &schedule.Interval); err != nil {
		return err
	}

	// Get existing schedule to preserve certain fields
	existing, err := u.storage.GetCheckerSchedule(schedule.Id)
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

	return u.storage.UpdateCheckerSchedule(schedule)
}

// DeleteSchedule removes a schedule
func (u *CheckScheduleUsecase) DeleteSchedule(scheduleId happydns.Identifier) error {
	return u.storage.DeleteCheckerSchedule(scheduleId)
}

// EnableSchedule enables a schedule
func (u *CheckScheduleUsecase) EnableSchedule(scheduleId happydns.Identifier) error {
	schedule, err := u.storage.GetCheckerSchedule(scheduleId)
	if err != nil {
		return err
	}

	schedule.Enabled = true

	// Reset next run time if it's in the past
	if schedule.NextRun.Before(time.Now()) {
		schedule.NextRun = time.Now().Add(schedule.Interval)
	}

	return u.storage.UpdateCheckerSchedule(schedule)
}

// DisableSchedule disables a schedule
func (u *CheckScheduleUsecase) DisableSchedule(scheduleId happydns.Identifier) error {
	schedule, err := u.storage.GetCheckerSchedule(scheduleId)
	if err != nil {
		return err
	}

	schedule.Enabled = false
	return u.storage.UpdateCheckerSchedule(schedule)
}

// UpdateScheduleAfterRun updates a schedule after it has been executed
func (u *CheckScheduleUsecase) UpdateScheduleAfterRun(scheduleId happydns.Identifier) error {
	schedule, err := u.storage.GetCheckerSchedule(scheduleId)
	if err != nil {
		return err
	}

	now := time.Now()
	schedule.LastRun = &now
	schedule.NextRun = now.Add(schedule.Interval)

	return u.storage.UpdateCheckerSchedule(schedule)
}

// ListDueSchedules retrieves all enabled schedules that are due to run
func (u *CheckScheduleUsecase) ListDueSchedules() ([]*happydns.CheckerSchedule, error) {
	schedules, err := u.storage.ListEnabledCheckerSchedules()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var dueSchedules []*happydns.CheckerSchedule

	for _, schedule := range schedules {
		if schedule.Enabled && schedule.NextRun.Before(now) {
			dueSchedules = append(dueSchedules, schedule)
		}
	}

	return dueSchedules, nil
}

// ListUpcomingSchedules retrieves the next `limit` enabled schedules sorted by NextRun ascending
func (u *CheckScheduleUsecase) ListUpcomingSchedules(limit int) ([]*happydns.CheckerSchedule, error) {
	schedules, err := u.storage.ListEnabledCheckerSchedules()
	if err != nil {
		return nil, err
	}

	sort.Slice(schedules, func(i, j int) bool {
		return schedules[i].NextRun.Before(schedules[j].NextRun)
	})

	if limit > 0 && len(schedules) > limit {
		schedules = schedules[:limit]
	}

	return schedules, nil
}

// getDefaultInterval returns the default check interval. If the checker
// implements CheckerIntervalProvider, its default is used; otherwise a
// scope-based fallback is returned.
func (u *CheckScheduleUsecase) getDefaultInterval(checkerName string, targetType happydns.CheckScopeType) time.Duration {
	if spec := u.getCheckerIntervalSpec(checkerName); spec != nil {
		return spec.Default
	}

	switch targetType {
	case happydns.CheckScopeUser:
		return DefaultUserCheckInterval
	case happydns.CheckScopeDomain:
		return DefaultDomainCheckInterval
	case happydns.CheckScopeService:
		return DefaultServiceCheckInterval
	default:
		return DefaultDomainCheckInterval
	}
}

// validateInterval clamps the interval to per-checker bounds (if available)
// and enforces the global MinimumCheckInterval as an absolute floor.
func (u *CheckScheduleUsecase) validateInterval(checkerName string, interval *time.Duration) error {
	if spec := u.getCheckerIntervalSpec(checkerName); spec != nil {
		if *interval < spec.Min {
			*interval = spec.Min
		}
		if *interval > spec.Max {
			*interval = spec.Max
		}
	}

	if *interval < MinimumCheckInterval {
		return fmt.Errorf("check interval must be at least %v", MinimumCheckInterval)
	}

	return nil
}

// getCheckerIntervalSpec returns the CheckIntervalSpec for a checker if it
// implements CheckerIntervalProvider, or nil otherwise.
func (u *CheckScheduleUsecase) getCheckerIntervalSpec(checkerName string) *happydns.CheckIntervalSpec {
	if u.checkerUsecase == nil {
		return nil
	}
	checker, err := u.checkerUsecase.GetChecker(checkerName)
	if err != nil {
		return nil
	}
	ip, ok := checker.(happydns.CheckerIntervalProvider)
	if !ok {
		return nil
	}
	spec := ip.CheckInterval()
	return &spec
}

// ValidateScheduleOwnership checks if a user owns a schedule
func (u *CheckScheduleUsecase) ValidateScheduleOwnership(scheduleId happydns.Identifier, userId happydns.Identifier) error {
	schedule, err := u.storage.GetCheckerSchedule(scheduleId)
	if err != nil {
		return err
	}

	if !schedule.OwnerId.Equals(userId) {
		return fmt.Errorf("user does not own this schedule")
	}

	return nil
}

// CreateDefaultSchedulesForTarget creates default schedules for a new target
func (u *CheckScheduleUsecase) CreateDefaultSchedulesForTarget(
	checkerName string,
	targetType happydns.CheckScopeType,
	targetId happydns.Identifier,
	ownerId happydns.Identifier,
	enabled bool,
) error {
	schedule := &happydns.CheckerSchedule{
		CheckerName: checkerName,
		OwnerId:     ownerId,
		TargetType:  targetType,
		TargetId:    targetId,
		Interval:    u.getDefaultInterval(checkerName, targetType),
		Enabled:     enabled,
		NextRun:     time.Now().Add(u.getDefaultInterval(checkerName, targetType)),
		Options:     make(happydns.CheckerOptions),
	}

	return u.CreateSchedule(schedule)
}

// DeleteSchedulesForTarget removes all schedules for a target
func (u *CheckScheduleUsecase) DeleteSchedulesForTarget(targetType happydns.CheckScopeType, targetId happydns.Identifier) error {
	schedules, err := u.storage.ListCheckerSchedulesByTarget(targetType, targetId)
	if err != nil {
		return err
	}

	for _, schedule := range schedules {
		if err := u.storage.DeleteCheckerSchedule(schedule.Id); err != nil {
			return err
		}
	}

	return nil
}
