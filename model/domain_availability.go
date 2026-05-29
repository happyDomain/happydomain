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
	"context"
	"time"
)

// DomainAvailabilityCheckerID is the id of the dedicated checker run against
// domain availability watches. It has all CheckerAvailability flags false so it
// is never auto-scheduled on managed domains; the scheduler enqueues it only
// from the watch enumeration.
const DomainAvailabilityCheckerID = "domain_availability"

// DomainAvailabilityWatch tracks a domain name the User does NOT own, so they
// can be notified the moment it becomes available for registration. Unlike a
// Domain, a watch is not tied to any Provider and never manages a zone.
type DomainAvailabilityWatch struct {
	// Id is the watch's identifier in the database.
	Id Identifier `json:"id" swaggertype:"string" binding:"required" readonly:"true"`

	// Owner is the identifier of the User watching the domain.
	Owner Identifier `json:"id_owner" swaggertype:"string" binding:"required" readonly:"true"`

	// DomainName is the FQDN being watched for availability.
	DomainName string `json:"domain" binding:"required"`

	// Interval optionally overrides how often the availability check runs.
	// When nil, the checker's default interval is used.
	Interval *time.Duration `json:"interval,omitempty" swaggertype:"integer"`

	// CreatedAt records when the watch was registered.
	CreatedAt time.Time `json:"created_at" readonly:"true"`
}

// DomainAvailabilityWatchCreationInput is used for swagger documentation as
// availability watch add.
type DomainAvailabilityWatchCreationInput struct {
	// DomainName is the FQDN to watch for availability.
	DomainName string `json:"domain" binding:"required"`

	// Interval optionally overrides the default check interval.
	Interval *time.Duration `json:"interval,omitempty" swaggertype:"integer"`
}

// NewDomainAvailabilityWatch validates the name and builds a watch owned by the
// given user.
func NewDomainAvailabilityWatch(user *User, name string) (*DomainAvailabilityWatch, error) {
	name, err := NormalizeDomainName(name)
	if err != nil {
		return nil, err
	}

	return &DomainAvailabilityWatch{
		Owner:      user.Id,
		DomainName: name,
		CreatedAt:  time.Now(),
	}, nil
}

// DomainAvailabilityWatchUsecase exposes owner-scoped operations on the
// availability watchlist.
type DomainAvailabilityWatchUsecase interface {
	CreateDomainAvailabilityWatch(context.Context, *User, *DomainAvailabilityWatchCreationInput) (*DomainAvailabilityWatch, error)
	DeleteDomainAvailabilityWatch(*User, Identifier) error
	GetUserDomainAvailabilityWatch(*User, Identifier) (*DomainAvailabilityWatch, error)
	ListUserDomainAvailabilityWatches(*User) ([]*DomainAvailabilityWatch, error)
}

// SchedulerWatchNotifier is an optional callback to notify the scheduler about
// availability-watch changes so it can incrementally update its job queue.
type SchedulerWatchNotifier interface {
	NotifyWatchChange(watch *DomainAvailabilityWatch)
	NotifyWatchRemoved(watchID Identifier)
}
