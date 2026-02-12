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
	"errors"
	"fmt"
	"math/rand"
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
	domainLister   DomainLister
	checkerUsecase happydns.CheckerUsecase
}

// NewCheckScheduleUsecase creates a new check schedule usecase
func NewCheckScheduleUsecase(storage CheckResultStorage, options *happydns.Options, domainLister DomainLister, checkerUsecase happydns.CheckerUsecase) *CheckScheduleUsecase {
	return &CheckScheduleUsecase{
		storage:        storage,
		options:        options,
		domainLister:   domainLister,
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
		schedule.Interval = u.getDefaultInterval(schedule.TargetType)
	}

	// Validate interval
	if schedule.Interval < MinimumCheckInterval {
		return fmt.Errorf("check interval must be at least %v", MinimumCheckInterval)
	}

	// Calculate next run time: pick a random offset within the interval
	// to spread load evenly across all schedules
	// TODO: Use a smarter load balance function in the future
	if schedule.NextRun.IsZero() {
		offset := time.Duration(rand.Int63n(int64(schedule.Interval)))
		schedule.NextRun = time.Now().Add(offset)
	}

	return u.storage.CreateCheckerSchedule(schedule)
}

// UpdateSchedule updates an existing schedule
func (u *CheckScheduleUsecase) UpdateSchedule(schedule *happydns.CheckerSchedule) error {
	// Validate interval
	if schedule.Interval < MinimumCheckInterval {
		return fmt.Errorf("check interval must be at least %v", MinimumCheckInterval)
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
		if schedule.NextRun.Before(now) {
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

// getDefaultInterval returns the default check interval based on target type
func (u *CheckScheduleUsecase) getDefaultInterval(targetType happydns.CheckScopeType) time.Duration {
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
		Interval:    u.getDefaultInterval(targetType),
		Enabled:     enabled,
		NextRun:     time.Now().Add(u.getDefaultInterval(targetType)),
		Options:     make(happydns.CheckerOptions),
	}

	return u.CreateSchedule(schedule)
}

// rescheduleChecks reschedules each given schedule to a random time in [now, now+maxOffsetFn(schedule)].
func (u *CheckScheduleUsecase) rescheduleChecks(schedules []*happydns.CheckerSchedule, maxOffsetFn func(*happydns.CheckerSchedule) time.Duration) (int, error) {
	count := 0
	now := time.Now()
	for _, schedule := range schedules {
		maxOffset := maxOffsetFn(schedule)
		if maxOffset <= 0 {
			maxOffset = time.Second
		}
		schedule.NextRun = now.Add(time.Duration(rand.Int63n(int64(maxOffset))))
		if err := u.storage.UpdateCheckerSchedule(schedule); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// RescheduleUpcomingChecks randomizes the next run time of all enabled schedules
// within their respective intervals to spread load evenly. Useful after a restart.
func (u *CheckScheduleUsecase) RescheduleUpcomingChecks() (int, error) {
	schedules, err := u.storage.ListEnabledCheckerSchedules()
	if err != nil {
		return 0, err
	}
	return u.rescheduleChecks(schedules, func(s *happydns.CheckerSchedule) time.Duration {
		return s.Interval
	})
}

// RescheduleOverdueChecks reschedules checks whose NextRun is in the past,
// spreading them over a short window to avoid scheduler famine (e.g. after
// a long machine suspend or server downtime).
// If there are fewer than 10 overdue checks, they are left as-is so that the
// caller's immediate checkSchedules pass enqueues them directly.
func (u *CheckScheduleUsecase) RescheduleOverdueChecks() (int, error) {
	schedules, err := u.storage.ListEnabledCheckerSchedules()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	var overdue []*happydns.CheckerSchedule
	for _, s := range schedules {
		if s.NextRun.Before(now) {
			overdue = append(overdue, s)
		}
	}

	if len(overdue) == 0 {
		return 0, nil
	}

	// Small backlog: let the caller enqueue them directly on the next
	// checkSchedules pass rather than deferring them into the future.
	if len(overdue) < 10 {
		return 0, nil
	}

	// Spread overdue checks over a small window proportional to their count,
	// capped at MinimumCheckInterval, to prevent all of them from running at once.
	spreadWindow := time.Duration(len(overdue)) * 5 * time.Second
	if spreadWindow > MinimumCheckInterval {
		spreadWindow = MinimumCheckInterval
	}

	return u.rescheduleChecks(overdue, func(s *happydns.CheckerSchedule) time.Duration {
		return spreadWindow
	})
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

// DiscoverAndEnsureSchedules creates default enabled schedules for all (plugin, domain)
// pairs that don't yet have an explicit schedule record. This implements the opt-out
// model: checks run automatically unless a schedule with Enabled=false has been saved.
// Non-fatal per-domain errors are collected and returned together.
func (u *CheckScheduleUsecase) DiscoverAndEnsureSchedules() error {
	if u.domainLister == nil || u.checkerUsecase == nil {
		return nil
	}

	plugins, err := u.checkerUsecase.ListCheckers()
	if err != nil {
		return fmt.Errorf("listing check plugins for discovery: %w", err)
	}

	iter, err := u.domainLister.ListAllDomains()
	if err != nil {
		return fmt.Errorf("listing domains for schedule discovery: %w", err)
	}
	defer iter.Close()

	var errs []error
	for iter.Next() {
		domain := iter.Item()
		if domain == nil {
			continue
		}
		for checkerName, p := range *plugins {
			if !p.Availability().ApplyToDomain {
				continue
			}

			schedules, err := u.ListSchedulesByTarget(happydns.CheckScopeDomain, domain.Id)
			if err != nil {
				errs = append(errs, fmt.Errorf("listing schedules for domain %s: %w", domain.Id, err))
				continue
			}

			hasSchedule := false
			for _, sched := range schedules {
				if sched.CheckerName == checkerName {
					hasSchedule = true
					break
				}
			}

			if !hasSchedule {
				if err := u.CreateSchedule(&happydns.CheckerSchedule{
					CheckerName: checkerName,
					OwnerId:     domain.Owner,
					TargetType:  happydns.CheckScopeDomain,
					TargetId:    domain.Id,
					Enabled:     true,
				}); err != nil {
					errs = append(errs, fmt.Errorf("auto-creating schedule for domain %s / plugin %s: %w",
						domain.Id, checkerName, err))
				}
			}
		}
	}

	return errors.Join(errs...)
}
