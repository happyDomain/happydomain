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

package notification

import (
	"time"

	"git.happydns.org/happyDomain/model"
)

type decisionAction int

const (
	actionSkip decisionAction = iota
	actionAdvance
	actionNotify
)

// Reason is for logging only — callers must not branch on its text.
type decision struct {
	Action       decisionAction
	Reason       string
	IsRecovery   bool
	IsEscalation bool
	ClearAck     bool
}

// Pure predicate. nil pref means "no preference" — suppress notify, still advance state. now is injected for quiet-hour tests.
func decide(state *happydns.NotificationState, pref *happydns.NotificationPreference, newStatus happydns.Status, now time.Time) decision {
	oldStatus := state.LastStatus

	if oldStatus == newStatus {
		return decision{Action: actionSkip, Reason: "no transition"}
	}

	isRecovery := newStatus < happydns.StatusWarn && oldStatus >= happydns.StatusWarn
	isEscalation := newStatus > oldStatus && newStatus >= happydns.StatusWarn
	clearAck := isRecovery || isEscalation

	d := decision{IsRecovery: isRecovery, IsEscalation: isEscalation, ClearAck: clearAck}

	if pref == nil {
		d.Action = actionAdvance
		d.Reason = "no preference configured"
		return d
	}
	if !pref.Enabled {
		d.Action = actionAdvance
		d.Reason = "preference disabled"
		return d
	}
	if !isRecovery && newStatus < pref.MinStatus {
		d.Action = actionAdvance
		d.Reason = "below MinStatus threshold"
		return d
	}
	if isRecovery && !pref.NotifyRecovery {
		d.Action = actionAdvance
		d.Reason = "recovery suppressed by preference"
		return d
	}
	// Active ack means user already knows; recoveries skip this check.
	if state.Acknowledged && !clearAck && !isRecovery {
		d.Action = actionAdvance
		d.Reason = "acknowledged"
		return d
	}
	if isQuietHour(pref, now) {
		d.Action = actionAdvance
		d.Reason = "quiet hours"
		return d
	}

	d.Action = actionNotify
	d.Reason = "notify"
	return d
}

func isQuietHour(pref *happydns.NotificationPreference, now time.Time) bool {
	if pref.QuietStart == nil || pref.QuietEnd == nil {
		return false
	}
	loc := time.UTC
	if pref.Timezone != "" {
		// Validated at write time; on a stale/invalid value we silently fall back to UTC rather than firing during what the user thinks are quiet hours.
		if l, err := time.LoadLocation(pref.Timezone); err == nil {
			loc = l
		}
	}
	hour := now.In(loc).Hour()
	start := *pref.QuietStart
	end := *pref.QuietEnd
	if start <= end {
		return hour >= start && hour < end
	}
	// Wraps midnight, e.g. 22:00 - 06:00.
	return hour >= start || hour < end
}
