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
	"errors"
	"fmt"
	"math/rand"
	"sort"
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
	storage       TestResultStorage
	options       *happydns.Options
	domainLister  DomainLister
	pluginUsecase happydns.TestPluginUsecase
}

// NewTestScheduleUsecase creates a new test schedule usecase
func NewTestScheduleUsecase(storage TestResultStorage, options *happydns.Options, domainLister DomainLister, pluginUsecase happydns.TestPluginUsecase) *TestScheduleUsecase {
	return &TestScheduleUsecase{
		storage:       storage,
		options:       options,
		domainLister:  domainLister,
		pluginUsecase: pluginUsecase,
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
	// Set default interval if not specified
	if schedule.Interval == 0 {
		schedule.Interval = u.getDefaultInterval(schedule.TargetType)
	}

	// Validate interval
	if schedule.Interval < MinimumTestInterval {
		return fmt.Errorf("test interval must be at least %v", MinimumTestInterval)
	}

	// Calculate next run time: pick a random offset within the interval
	// to spread load evenly across all schedules
	// TODO: Use a smarter load balance function in the future
	if schedule.NextRun.IsZero() {
		offset := time.Duration(rand.Int63n(int64(schedule.Interval)))
		schedule.NextRun = time.Now().Add(offset)
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
		if schedule.NextRun.Before(now) {
			dueSchedules = append(dueSchedules, schedule)
		}
	}

	return dueSchedules, nil
}

// ListUpcomingSchedules retrieves the next `limit` enabled schedules sorted by NextRun ascending
func (u *TestScheduleUsecase) ListUpcomingSchedules(limit int) ([]*happydns.TestSchedule, error) {
	schedules, err := u.storage.ListEnabledTestSchedules()
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

// rescheduleTests reschedules each given schedule to a random time in [now, now+maxOffsetFn(schedule)].
func (u *TestScheduleUsecase) rescheduleTests(schedules []*happydns.TestSchedule, maxOffsetFn func(*happydns.TestSchedule) time.Duration) (int, error) {
	count := 0
	now := time.Now()
	for _, schedule := range schedules {
		maxOffset := maxOffsetFn(schedule)
		if maxOffset <= 0 {
			maxOffset = time.Second
		}
		schedule.NextRun = now.Add(time.Duration(rand.Int63n(int64(maxOffset))))
		if err := u.storage.UpdateTestSchedule(schedule); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// RescheduleUpcomingTests randomizes the next run time of all enabled schedules
// within their respective intervals to spread load evenly. Useful after a restart.
func (u *TestScheduleUsecase) RescheduleUpcomingTests() (int, error) {
	schedules, err := u.storage.ListEnabledTestSchedules()
	if err != nil {
		return 0, err
	}
	return u.rescheduleTests(schedules, func(s *happydns.TestSchedule) time.Duration {
		return s.Interval
	})
}

// RescheduleOverdueTests reschedules tests whose NextRun is in the past,
// spreading them over a short window to avoid scheduler famine (e.g. after
// a long machine suspend or server downtime).
// If there are fewer than 10 overdue tests, they are left as-is so that the
// caller's immediate checkSchedules pass enqueues them directly.
func (u *TestScheduleUsecase) RescheduleOverdueTests() (int, error) {
	schedules, err := u.storage.ListEnabledTestSchedules()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	var overdue []*happydns.TestSchedule
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

	// Spread overdue tests over a small window proportional to their count,
	// capped at MinimumTestInterval, to prevent all of them from running at once.
	spreadWindow := time.Duration(len(overdue)) * 5 * time.Second
	if spreadWindow > MinimumTestInterval {
		spreadWindow = MinimumTestInterval
	}

	return u.rescheduleTests(overdue, func(s *happydns.TestSchedule) time.Duration {
		return spreadWindow
	})
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

// DiscoverAndEnsureSchedules creates default enabled schedules for all (plugin, domain)
// pairs that don't yet have an explicit schedule record. This implements the opt-out
// model: tests run automatically unless a schedule with Enabled=false has been saved.
// Non-fatal per-domain errors are collected and returned together.
func (u *TestScheduleUsecase) DiscoverAndEnsureSchedules() error {
	if u.domainLister == nil || u.pluginUsecase == nil {
		return nil
	}

	plugins, err := u.pluginUsecase.ListTestPlugins()
	if err != nil {
		return fmt.Errorf("listing test plugins for discovery: %w", err)
	}

	var domainPlugins []happydns.TestPlugin
	for _, p := range plugins {
		if p.Version().AvailableOn.ApplyToDomain {
			domainPlugins = append(domainPlugins, p)
		}
	}

	if len(domainPlugins) == 0 {
		return nil
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
		for _, plugin := range domainPlugins {
			pluginName := plugin.PluginEnvName()[0]
			schedules, err := u.ListSchedulesByTarget(happydns.TestScopeDomain, domain.Id)
			if err != nil {
				errs = append(errs, fmt.Errorf("listing schedules for domain %s: %w", domain.Id, err))
				continue
			}

			hasSchedule := false
			for _, sched := range schedules {
				if sched.PluginName == pluginName {
					hasSchedule = true
					break
				}
			}

			if !hasSchedule {
				if err := u.CreateSchedule(&happydns.TestSchedule{
					PluginName: pluginName,
					OwnerId:    domain.Owner,
					TargetType: happydns.TestScopeDomain,
					TargetId:   domain.Id,
					Enabled:    true,
				}); err != nil {
					errs = append(errs, fmt.Errorf("auto-creating schedule for domain %s / plugin %s: %w",
						domain.Id, pluginName, err))
				}
			}
		}
	}

	return errors.Join(errs...)
}

