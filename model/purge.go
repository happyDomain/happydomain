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

// PurgeOptions selects what a purge deletes. Every selected category is
// data the server recreates by itself, or whose loss is minor.
type PurgeOptions struct {
	// Checkers deletes every execution, evaluation, observation snapshot,
	// cached observation and discovery entry. Check plans and checker
	// options are kept.
	Checkers bool `json:"checkers"`

	// ExpiredSessions deletes the sessions past their expiration date.
	ExpiredSessions bool `json:"expired_sessions"`

	// NotificationRecords deletes the whole history of sent notifications.
	NotificationRecords bool `json:"notification_records"`

	// Compact asks the storage backend to reclaim the freed space once the
	// deletions are done.
	Compact bool `json:"compact"`
}

// PurgeReport tells how many keys a purge deleted in each category.
type PurgeReport struct {
	Checkers            int `json:"checkers"`
	ExpiredSessions     int `json:"expired_sessions"`
	NotificationRecords int `json:"notification_records"`

	// Compacted is true when the backend reclaimed the freed space.
	Compacted bool `json:"compacted"`

	// Error tells what failed, the counts above still being what was
	// deleted before and despite the failure.
	Error string `json:"error,omitempty"`
}

type PurgeUsecase interface {
	// Purge deletes the selected categories. It goes on with the next
	// categories when one fails, and returns every error joined.
	Purge(opts PurgeOptions) (*PurgeReport, error)
}
