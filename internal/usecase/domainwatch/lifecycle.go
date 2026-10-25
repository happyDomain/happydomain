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

package domainwatch

import (
	"time"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/pkg/domaininfo"
)

const day = 24 * time.Hour

const (
	// expiringWithin is how close to its expiration date a registered
	// domain is said to be expiring.
	expiringWithin = 30 * day

	// stableInterval is how often a registered domain is looked up: its
	// expiration date does not come closer by surprise.
	stableInterval = 7 * day
	// closeInterval is how often a domain past its expiration date,
	// available or auto-renewed by the registry is looked up: every step
	// until the release lasts days.
	closeInterval = 12 * time.Hour
	// afterExpiration is how long after the expiration date of a domain it
	// is looked up again, to see whether it was renewed.
	afterExpiration = 6 * time.Hour

	// errorRetry is how long after a failed lookup the next one happens,
	// and firstErrorRetry when the domain was never looked up successfully.
	errorRetry      = 12 * time.Hour
	firstErrorRetry = time.Hour

	// redemptionLength and pendingDeleteLength are the usual length of
	// these periods for gTLDs (ICANN Expired Registration Recovery Policy).
	redemptionLength    = 30 * day
	pendingDeleteLength = 5 * day

	// maxHistory is the number of phase changes kept per watch.
	maxHistory = 20
)

// phaseOf is domaininfo.PhaseOf, telling apart the registered domains
// expiring within expiringWithin.
func phaseOf(info *domaininfo.DomainInfo, err error, now time.Time) (phase happydns.DomainWatchPhase, ok bool) {
	p, ok := domaininfo.PhaseOf(info, err, now)
	if !ok {
		return "", false
	}
	if p == domaininfo.PhaseRegistered && info.ExpirationDate != nil && info.ExpirationDate.Sub(now) <= expiringWithin {
		return happydns.DomainWatchExpiring, true
	}
	return happydns.DomainWatchPhase(p), true
}

// apply records the lookup of the domain of w, made at now, into w, and
// schedules its next lookup, delayed by jitter so that lookups spread over
// time. It returns the previous phase, and whether the change is worth
// telling the user: the first phase found is not.
func apply(w *happydns.DomainWatch, info *domaininfo.DomainInfo, err error, now time.Time, jitter time.Duration) (from happydns.DomainWatchPhase, changed bool) {
	w.LastCheck = &now

	phase, ok := phaseOf(info, err, now)
	if !ok {
		// A failed lookup tells nothing about the domain: keep what is
		// known, and try again later.
		if err != nil {
			w.LastError = err.Error()
		} else {
			w.LastError = "no registration information returned"
		}
		retry := errorRetry
		if w.Phase == happydns.DomainWatchUnknown || w.Phase == "" {
			retry = firstErrorRetry
		}
		w.NextCheck = now.Add(retry + jitter)
		return "", false
	}

	w.LastError = ""
	if phase == happydns.DomainWatchAvailable {
		w.ExpirationDate, w.Registrar, w.Status = nil, "", nil
	} else {
		w.ExpirationDate, w.Registrar, w.Status = info.ExpirationDate, info.Registrar, info.Status
	}

	from = w.Phase
	if from != phase {
		w.Phase = phase
		w.PhaseSince = &now
		if from != happydns.DomainWatchUnknown && from != "" {
			changed = true
			w.History = append(w.History, happydns.DomainWatchEvent{At: now, From: from, To: phase})
			if len(w.History) > maxHistory {
				w.History = w.History[len(w.History)-maxHistory:]
			}
		}
	}

	w.EstimatedRelease = estimateRelease(w)
	w.NextCheck = nextCheck(w, now).Add(jitter)

	return from, changed
}

// estimateRelease tells the latest date the registry should release the
// domain at. When the domain was already in its phase when first seen, the
// actual release comes earlier.
func estimateRelease(w *happydns.DomainWatch) *time.Time {
	if w.PhaseSince == nil {
		return nil
	}

	var t time.Time
	switch w.Phase {
	case happydns.DomainWatchRedemption:
		t = w.PhaseSince.Add(redemptionLength + pendingDeleteLength)
	case happydns.DomainWatchPendingDelete:
		t = w.PhaseSince.Add(pendingDeleteLength)
	default:
		return nil
	}
	return &t
}

// nextCheck tells when to look the domain of w up again.
func nextCheck(w *happydns.DomainWatch, now time.Time) time.Time {
	switch w.Phase {
	case happydns.DomainWatchRegistered, happydns.DomainWatchExpiring:
		if isAutoRenewPending(w) {
			return now.Add(closeInterval)
		}
	default:
		return now.Add(closeInterval)
	}

	next := now.Add(stableInterval)
	if w.ExpirationDate == nil {
		return next
	}

	// Do not miss the moments that matter: when the domain starts expiring
	// and right after it expired.
	for _, t := range []time.Time{
		w.ExpirationDate.Add(-expiringWithin),
		w.ExpirationDate.Add(afterExpiration),
	} {
		if t.After(now) && t.Before(next) {
			next = t
		}
	}
	return next
}

// isAutoRenewPending tells whether the registry renewed the domain of w on
// its own, at its expiration date, and waits for the registrar to confirm
// the renewal or to delete the domain.
func isAutoRenewPending(w *happydns.DomainWatch) bool {
	return domaininfo.HasStatus(w.Status, "autoRenewPeriod")
}
