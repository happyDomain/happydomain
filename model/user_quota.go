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

import "time"

// UserQuota holds admin-controlled per-user limits and flags. These fields are
// never modifiable by the user; they can only be updated through the admin API.
//
// Only checker-related fields are defined for now. Future paid-plan attributes
// (plan tier, domain caps, payment metadata, ...) will be added here later.
type UserQuota struct {
	// MaxChecksPerDay caps the number of checker executions per day for this
	// user. 0 means "use the system default".
	MaxChecksPerDay int `json:"max_checks_per_day,omitempty"`

	// RetentionDays is the maximum age (in days) of checker executions kept in
	// storage for this user. 0 means "use the system default".
	RetentionDays int `json:"retention_days,omitempty"`

	// InactivityPauseDays is the number of days without login after which the
	// scheduler stops running checks for this user. 0 means "use the system
	// default". A negative value disables the inactivity pause for this user.
	InactivityPauseDays int `json:"inactivity_pause_days,omitempty"`

	// SchedulingPaused, when true, completely disables the scheduler for this
	// user (admin kill switch).
	SchedulingPaused bool `json:"scheduling_paused,omitempty"`

	// UpdatedAt records the last time these quotas were modified.
	UpdatedAt time.Time `json:"updated_at,omitzero" format:"date-time"`
}
