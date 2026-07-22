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
	"testing"
	"time"

	"git.happydns.org/happyDomain/model"
)

//go:fix inline
func ptr[T any](v T) *T { return new(v) }

func TestDecide(t *testing.T) {
	noon := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	night := time.Date(2026, 4, 29, 3, 0, 0, 0, time.UTC)

	enabled := func() *happydns.NotificationPreference {
		return &happydns.NotificationPreference{
			Enabled:        true,
			MinStatus:      happydns.StatusWarn,
			NotifyRecovery: true,
		}
	}

	tests := []struct {
		name       string
		state      happydns.NotificationState
		pref       *happydns.NotificationPreference
		newStatus  happydns.Status
		now        time.Time
		wantAction decisionAction
		wantClear  bool
	}{
		{
			name:       "no transition",
			state:      happydns.NotificationState{LastStatus: happydns.StatusOK},
			pref:       enabled(),
			newStatus:  happydns.StatusOK,
			now:        noon,
			wantAction: actionSkip,
		},
		{
			name:       "nil preference advances state without notifying",
			state:      happydns.NotificationState{LastStatus: happydns.StatusOK},
			pref:       nil,
			newStatus:  happydns.StatusCrit,
			now:        noon,
			wantAction: actionAdvance,
			wantClear:  true,
		},
		{
			name:  "preference disabled advances",
			state: happydns.NotificationState{LastStatus: happydns.StatusOK},
			pref: &happydns.NotificationPreference{
				Enabled:   false,
				MinStatus: happydns.StatusWarn,
			},
			newStatus:  happydns.StatusCrit,
			now:        noon,
			wantAction: actionAdvance,
			wantClear:  true,
		},
		{
			name:       "below MinStatus advances",
			state:      happydns.NotificationState{LastStatus: happydns.StatusUnknown},
			pref:       enabled(),
			newStatus:  happydns.StatusOK,
			now:        noon,
			wantAction: actionAdvance,
		},
		{
			name:  "recovery suppressed when NotifyRecovery is false",
			state: happydns.NotificationState{LastStatus: happydns.StatusCrit},
			pref: &happydns.NotificationPreference{
				Enabled:        true,
				MinStatus:      happydns.StatusWarn,
				NotifyRecovery: false,
			},
			newStatus:  happydns.StatusOK,
			now:        noon,
			wantAction: actionAdvance,
			wantClear:  true,
		},
		{
			name:       "recovery notifies when NotifyRecovery is true",
			state:      happydns.NotificationState{LastStatus: happydns.StatusCrit},
			pref:       enabled(),
			newStatus:  happydns.StatusOK,
			now:        noon,
			wantAction: actionNotify,
			wantClear:  true,
		},
		{
			name:       "escalation notifies and clears ack",
			state:      happydns.NotificationState{LastStatus: happydns.StatusWarn, Acknowledged: true},
			pref:       enabled(),
			newStatus:  happydns.StatusCrit,
			now:        noon,
			wantAction: actionNotify,
			wantClear:  true,
		},
		{
			name:       "acknowledged non-recovery is suppressed",
			state:      happydns.NotificationState{LastStatus: happydns.StatusCrit, Acknowledged: true},
			pref:       enabled(),
			newStatus:  happydns.StatusWarn,
			now:        noon,
			wantAction: actionAdvance,
			wantClear:  false,
		},
		{
			name:  "quiet hours suppress alert",
			state: happydns.NotificationState{LastStatus: happydns.StatusOK},
			pref: &happydns.NotificationPreference{
				Enabled:        true,
				MinStatus:      happydns.StatusWarn,
				NotifyRecovery: true,
				QuietStart:     new(22),
				QuietEnd:       new(6),
			},
			newStatus:  happydns.StatusCrit,
			now:        night,
			wantAction: actionAdvance,
			wantClear:  true,
		},
		{
			name:  "outside quiet hours notifies",
			state: happydns.NotificationState{LastStatus: happydns.StatusOK},
			pref: &happydns.NotificationPreference{
				Enabled:        true,
				MinStatus:      happydns.StatusWarn,
				NotifyRecovery: true,
				QuietStart:     new(22),
				QuietEnd:       new(6),
			},
			newStatus:  happydns.StatusCrit,
			now:        noon,
			wantAction: actionNotify,
			wantClear:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := decide(&tc.state, tc.pref, tc.newStatus, tc.now)
			if got.Action != tc.wantAction {
				t.Errorf("Action: got %v (%s), want %v", got.Action, got.Reason, tc.wantAction)
			}
			if got.ClearAck != tc.wantClear {
				t.Errorf("ClearAck: got %v, want %v", got.ClearAck, tc.wantClear)
			}
		})
	}
}

func TestIsQuietHour(t *testing.T) {
	at := func(h int) time.Time {
		return time.Date(2026, 4, 29, h, 30, 0, 0, time.UTC)
	}
	tests := []struct {
		name   string
		start  *int
		end    *int
		hour   int
		expect bool
	}{
		{"no window", nil, nil, 3, false},
		{"inside non-wrap (9-17)", new(9), new(17), 12, true},
		{"outside non-wrap (9-17)", new(9), new(17), 18, false},
		{"end-exclusive (9-17 at 17)", new(9), new(17), 17, false},
		{"wrap inside before midnight (22-6)", new(22), new(6), 23, true},
		{"wrap inside after midnight (22-6)", new(22), new(6), 3, true},
		{"wrap outside (22-6)", new(22), new(6), 12, false},
		{"wrap end-exclusive (22-6 at 6)", new(22), new(6), 6, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pref := &happydns.NotificationPreference{QuietStart: tc.start, QuietEnd: tc.end}
			if got := isQuietHour(pref, at(tc.hour)); got != tc.expect {
				t.Errorf("got %v, want %v", got, tc.expect)
			}
		})
	}
}

func TestIsQuietHourTimezone(t *testing.T) {
	// 02:30 UTC == 12:30 Asia/Tokyo (UTC+9), so a 9-17 quiet window in Tokyo should fire while UTC says off-hours.
	now := time.Date(2026, 4, 29, 2, 30, 0, 0, time.UTC)
	pref := &happydns.NotificationPreference{
		QuietStart: new(9),
		QuietEnd:   new(17),
		Timezone:   "Asia/Tokyo",
	}
	if !isQuietHour(pref, now) {
		t.Fatalf("expected quiet hour in Asia/Tokyo at local 11:30, got false")
	}
	pref.Timezone = ""
	if isQuietHour(pref, now) {
		t.Fatalf("expected non-quiet in UTC at 02:30, got true")
	}
	// Invalid TZ falls back to UTC.
	pref.Timezone = "Not/AReal_Zone"
	if isQuietHour(pref, now) {
		t.Fatalf("expected fallback to UTC for invalid timezone, got quiet hour")
	}
}
