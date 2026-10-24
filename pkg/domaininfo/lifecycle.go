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

package domaininfo

import (
	"errors"
	"time"

	"git.happydns.org/happyDomain/pkg/domaininfo/types"
)

// Phase is the stage of its registration life cycle a domain name is in, as
// the registry tells it.
//
// A domain that is not renewed goes, for gTLDs, through a grace period
// (usually up to 45 days, during which its owner can still renew it as
// usual), a redemption period (30 days, during which its owner can only
// restore it, for a fee), then a pending delete period (5 days, nothing can be
// done anymore), at the end of which the registry releases it. ccTLDs follow
// their own rules, often a subset of these stages.
type Phase string

const (
	// PhaseRegistered is a domain held by someone, not expired.
	PhaseRegistered Phase = "registered"
	// PhaseGrace is a domain whose expiration date passed. Its owner can
	// still renew it. A domain the registry auto-renewed (autoRenewPeriod
	// status) has a new expiration date: it stays registered.
	PhaseGrace Phase = "grace"
	// PhaseRedemption is a domain deleted by its registrar that its owner
	// can still restore.
	PhaseRedemption Phase = "redemption"
	// PhasePendingDelete is a domain about to be released by the registry.
	PhasePendingDelete Phase = "pending_delete"
	// PhaseAvailable is a domain the registry does not know: it can be
	// registered.
	PhaseAvailable Phase = "available"
)

// PhaseOf tells the phase of a domain from the result of its lookup (info
// and err, as returned by GetDomainInfo). ok is false when the lookup failed
// for another reason than the domain not being registered: nothing can be
// told about the domain then.
func PhaseOf(info *DomainInfo, err error, now time.Time) (phase Phase, ok bool) {
	if err != nil {
		if errors.Is(err, types.ErrDomainDoesNotExist) {
			return PhaseAvailable, true
		}
		return "", false
	}
	if info == nil {
		return "", false
	}

	switch {
	case HasStatus(info.Status, "pendingDelete"):
		return PhasePendingDelete, true
	case HasStatus(info.Status, "redemptionPeriod"), HasStatus(info.Status, "pendingRestore"):
		return PhaseRedemption, true
	case info.ExpirationDate != nil && info.ExpirationDate.Before(now):
		return PhaseGrace, true
	}
	return PhaseRegistered, true
}
