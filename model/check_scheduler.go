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

// SchedulerUsecase defines the interface for triggering on-demand checks
type SchedulerUsecase interface {
	Run()
	Close()
	TriggerOnDemandCheck(checkerName string, targetType CheckScopeType, targetID Identifier, userID Identifier, options CheckerOptions) (Identifier, error)
}

// CheckerSchedule defines a recurring check schedule
type CheckerSchedule struct {
	// Id is the unique identifier for this schedule
	Id Identifier `json:"id" swaggertype:"string"`

	// CheckerName identifies which checker to execute
	CheckerName string `json:"checker_name"`

	// OwnerId is the owner of the schedule
	OwnerId Identifier `json:"owner_id" swaggertype:"string"`

	// TargetType indicates what type of target to check
	TargetType CheckScopeType `json:"target_type"`

	// TargetId is the identifier of the target to check
	TargetId Identifier `json:"target_id" swaggertype:"string"`

	// Interval is how often to run the check
	Interval time.Duration `json:"interval" swaggertype:"integer"`

	// Enabled indicates if the schedule is active
	Enabled bool `json:"enabled"`

	// LastRun is when the check was last executed (nil if never run)
	LastRun *time.Time `json:"last_run,omitempty"`

	// NextRun is when the check should next be executed
	NextRun time.Time `json:"next_run"`

	// Options contains checker-specific configuration
	Options CheckerOptions `json:"options,omitempty"`
}

// SchedulerStatus holds a snapshot of the scheduler state for monitoring
type SchedulerStatus struct {
	// ConfigEnabled indicates if the scheduler is enabled in the configuration file
	ConfigEnabled bool `json:"config_enabled"`

	// RuntimeEnabled indicates if the scheduler is currently enabled at runtime
	RuntimeEnabled bool `json:"runtime_enabled"`

	// Running indicates if the scheduler goroutine is currently running
	Running bool `json:"running"`

	// WorkerCount is the number of worker goroutines
	WorkerCount int `json:"worker_count"`

	// QueueSize is the number of items currently waiting in the execution queue
	QueueSize int `json:"queue_size"`

	// ActiveCount is the number of checks currently being executed
	ActiveCount int `json:"active_count"`

	// NextSchedules contains the upcoming scheduled checks sorted by next run time
	NextSchedules []*CheckerSchedule `json:"next_schedules"`
}

// CheckerScheduleUsecase defines business logic for check schedules
type CheckerScheduleUsecase interface {
	// ListUserSchedules retrieves all schedules for a specific user
	ListUserSchedules(userId Identifier) ([]*CheckerSchedule, error)

	// ListSchedulesByTarget retrieves all schedules for a specific target
	ListSchedulesByTarget(targetType CheckScopeType, targetId Identifier) ([]*CheckerSchedule, error)

	// GetSchedule retrieves a specific schedule by ID
	GetSchedule(scheduleId Identifier) (*CheckerSchedule, error)

	// CreateSchedule creates a new check schedule with validation
	CreateSchedule(schedule *CheckerSchedule) error

	// UpdateSchedule updates an existing schedule
	UpdateSchedule(schedule *CheckerSchedule) error

	// DeleteSchedule removes a schedule
	DeleteSchedule(scheduleId Identifier) error

	// EnableSchedule enables a schedule
	EnableSchedule(scheduleId Identifier) error

	// DisableSchedule disables a schedule
	DisableSchedule(scheduleId Identifier) error

	// UpdateScheduleAfterRun updates a schedule after it has been executed
	UpdateScheduleAfterRun(scheduleId Identifier) error

	// ListDueSchedules retrieves all enabled schedules that are due to run
	ListDueSchedules() ([]*CheckerSchedule, error)

	// ValidateScheduleOwnership checks if a user owns a schedule
	ValidateScheduleOwnership(scheduleId Identifier, ownerId Identifier) error

	// DeleteSchedulesForTarget removes all schedules for a target
	DeleteSchedulesForTarget(targetType CheckScopeType, targetId Identifier) error
}
