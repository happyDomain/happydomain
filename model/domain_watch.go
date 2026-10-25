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

// DomainWatchPhase is where a watched domain stands, for someone waiting to
// register it. Apart from unknown and expiring, the phases are those of the
// registry life cycle told by domaininfo.PhaseOf, under the same names.
type DomainWatchPhase string

const (
	// DomainWatchUnknown is a domain never looked up successfully yet.
	DomainWatchUnknown DomainWatchPhase = "unknown"
	// DomainWatchRegistered is a domain held by someone, far from expiring.
	DomainWatchRegistered DomainWatchPhase = "registered"
	// DomainWatchExpiring is a registered domain close to its expiration
	// date.
	DomainWatchExpiring DomainWatchPhase = "expiring"
	// DomainWatchGrace is an expired domain its owner can still renew.
	DomainWatchGrace DomainWatchPhase = "grace"
	// DomainWatchRedemption is a deleted domain its owner can only restore.
	DomainWatchRedemption DomainWatchPhase = "redemption"
	// DomainWatchPendingDelete is a domain the registry releases within days.
	DomainWatchPendingDelete DomainWatchPhase = "pending_delete"
	// DomainWatchAvailable is a domain anyone can register.
	DomainWatchAvailable DomainWatchPhase = "available"
)

// DomainWatch is a domain name a user follows, without managing it, to be
// told when it is about to be released and when it can be registered.
type DomainWatch struct {
	Id     Identifier `json:"id" swaggertype:"string" readonly:"true"`
	UserId Identifier `json:"userId" swaggertype:"string" readonly:"true"`

	// DomainName is the registrable domain watched, in ASCII, without
	// trailing dot.
	DomainName string `json:"domain" readonly:"true"`

	// DisplayName is DomainName in Unicode, for internationalized domains.
	DisplayName string `json:"displayName" readonly:"true"`

	// Note is a free text from the user, such as why they want the domain.
	Note string `json:"note,omitempty"`

	CreatedAt time.Time `json:"createdAt" readonly:"true"`

	// Phase is where the domain stood at the last successful lookup.
	Phase DomainWatchPhase `json:"phase" readonly:"true"`

	// PhaseSince is when Phase was first seen.
	PhaseSince *time.Time `json:"phaseSince,omitempty" readonly:"true"`

	// ExpirationDate, Registrar and Status are the registration facts of
	// the last successful lookup; empty for an available domain.
	ExpirationDate *time.Time `json:"expiration,omitempty" readonly:"true"`
	Registrar      string     `json:"registrar,omitempty" readonly:"true"`
	Status         []string   `json:"status,omitempty" readonly:"true"`

	// EstimatedRelease is the latest date the registry should release the
	// domain at, once it is in redemption or pending deletion.
	EstimatedRelease *time.Time `json:"estimatedRelease,omitempty" readonly:"true"`

	// LastCheck is when the domain was last looked up, and LastError why
	// that lookup failed, if it did.
	LastCheck *time.Time `json:"lastCheck,omitempty" readonly:"true"`
	LastError string     `json:"lastError,omitempty" readonly:"true"`

	// NextCheck is when the domain will be looked up again.
	NextCheck time.Time `json:"nextCheck" readonly:"true"`

	// History lists the last phase changes, oldest first.
	History []DomainWatchEvent `json:"history,omitempty" readonly:"true"`
}

// DomainWatchEvent is a phase change of a watched domain.
type DomainWatchEvent struct {
	At   time.Time        `json:"at"`
	From DomainWatchPhase `json:"from"`
	To   DomainWatchPhase `json:"to"`
}

// DomainWatchInput is what a user tells about a domain to watch.
type DomainWatchInput struct {
	// Domain to watch; a subdomain or a URL is reduced to the registrable
	// domain. Ignored on update.
	Domain string `json:"domain,omitempty"`
	Note   string `json:"note"`
}

// DomainWatchUsecase manages the domains a user watches.
type DomainWatchUsecase interface {
	ListDomainWatches(user *User) ([]*DomainWatch, error)
	GetDomainWatch(user *User, id Identifier) (*DomainWatch, error)
	// CreateDomainWatch starts watching a domain, looking it up at once.
	CreateDomainWatch(ctx context.Context, user *User, input DomainWatchInput) (*DomainWatch, error)
	// UpdateDomainWatch changes the note of a watch.
	UpdateDomainWatch(user *User, id Identifier, input DomainWatchInput) (*DomainWatch, error)
	DeleteDomainWatch(user *User, id Identifier) error
	// CheckDomainWatch looks the domain up now.
	CheckDomainWatch(ctx context.Context, user *User, id Identifier) (*DomainWatch, error)
}
