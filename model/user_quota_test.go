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

package happydns_test

import (
	"encoding/json"
	"testing"
	"time"

	"git.happydns.org/happyDomain/model"
)

func TestUserQuotaZeroValues(t *testing.T) {
	q := happydns.UserQuota{}

	if q.MaxChecksPerDay != 0 {
		t.Errorf("zero UserQuota should have MaxChecksPerDay 0, got %d", q.MaxChecksPerDay)
	}
	if q.RetentionDays != 0 {
		t.Errorf("zero UserQuota should have RetentionDays 0, got %d", q.RetentionDays)
	}
	if q.InactivityPauseDays != 0 {
		t.Errorf("zero UserQuota should have InactivityPauseDays 0, got %d", q.InactivityPauseDays)
	}
	if q.SchedulingPaused {
		t.Error("zero UserQuota should have SchedulingPaused false")
	}
	if !q.UpdatedAt.IsZero() {
		t.Error("zero UserQuota should have zero UpdatedAt")
	}
}

func TestUserQuotaJSON_RoundTrip(t *testing.T) {
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)

	original := happydns.UserQuota{
		MaxChecksPerDay:     100,
		RetentionDays:       30,
		InactivityPauseDays: 14,
		SchedulingPaused:    true,
		UpdatedAt:           now,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal UserQuota: %v", err)
	}

	var decoded happydns.UserQuota
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal UserQuota: %v", err)
	}

	if decoded.MaxChecksPerDay != original.MaxChecksPerDay {
		t.Errorf("MaxChecksPerDay = %d; want %d", decoded.MaxChecksPerDay, original.MaxChecksPerDay)
	}
	if decoded.RetentionDays != original.RetentionDays {
		t.Errorf("RetentionDays = %d; want %d", decoded.RetentionDays, original.RetentionDays)
	}
	if decoded.InactivityPauseDays != original.InactivityPauseDays {
		t.Errorf("InactivityPauseDays = %d; want %d", decoded.InactivityPauseDays, original.InactivityPauseDays)
	}
	if decoded.SchedulingPaused != original.SchedulingPaused {
		t.Errorf("SchedulingPaused = %v; want %v", decoded.SchedulingPaused, original.SchedulingPaused)
	}
	if !decoded.UpdatedAt.Equal(original.UpdatedAt) {
		t.Errorf("UpdatedAt = %v; want %v", decoded.UpdatedAt, original.UpdatedAt)
	}
}

func TestUserQuotaJSON_OmitEmpty(t *testing.T) {
	q := happydns.UserQuota{}

	data, err := json.Marshal(q)
	if err != nil {
		t.Fatalf("failed to marshal zero UserQuota: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	for _, field := range []string{"max_checks_per_day", "retention_days", "inactivity_pause_days", "scheduling_paused"} {
		if _, ok := m[field]; ok {
			t.Errorf("zero-value field %q should be omitted from JSON, but was present", field)
		}
	}
}

func TestUserQuotaJSON_PartialDecode(t *testing.T) {
	raw := `{"retention_days": 7, "scheduling_paused": true}`

	var q happydns.UserQuota
	if err := json.Unmarshal([]byte(raw), &q); err != nil {
		t.Fatalf("failed to unmarshal partial JSON: %v", err)
	}

	if q.RetentionDays != 7 {
		t.Errorf("RetentionDays = %d; want 7", q.RetentionDays)
	}
	if !q.SchedulingPaused {
		t.Error("SchedulingPaused should be true")
	}
	if q.MaxChecksPerDay != 0 {
		t.Errorf("MaxChecksPerDay should default to 0, got %d", q.MaxChecksPerDay)
	}
	if q.InactivityPauseDays != 0 {
		t.Errorf("InactivityPauseDays should default to 0, got %d", q.InactivityPauseDays)
	}
}

func TestUserQuotaJSON_NegativeInactivityPauseDays(t *testing.T) {
	q := happydns.UserQuota{InactivityPauseDays: -1}

	data, err := json.Marshal(q)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded happydns.UserQuota
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.InactivityPauseDays != -1 {
		t.Errorf("InactivityPauseDays = %d; want -1", decoded.InactivityPauseDays)
	}
}

func TestUserWithQuotaJSON_RoundTrip(t *testing.T) {
	user := happydns.User{
		Id:    happydns.Identifier{0x01, 0x02},
		Email: "test@example.com",
		Quota: happydns.UserQuota{
			MaxChecksPerDay:  50,
			RetentionDays:    90,
			SchedulingPaused: false,
		},
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("failed to marshal User with Quota: %v", err)
	}

	var decoded happydns.User
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal User with Quota: %v", err)
	}

	if decoded.Quota.MaxChecksPerDay != 50 {
		t.Errorf("Quota.MaxChecksPerDay = %d; want 50", decoded.Quota.MaxChecksPerDay)
	}
	if decoded.Quota.RetentionDays != 90 {
		t.Errorf("Quota.RetentionDays = %d; want 90", decoded.Quota.RetentionDays)
	}
}

func TestUserWithEmptyQuotaJSON(t *testing.T) {
	raw := `{"id":"AQID","email":"test@example.com","created_at":"0001-01-01T00:00:00Z","last_seen":"0001-01-01T00:00:00Z","settings":{}}`

	var user happydns.User
	if err := json.Unmarshal([]byte(raw), &user); err != nil {
		t.Fatalf("failed to unmarshal User without quota field: %v", err)
	}

	if user.Quota.MaxChecksPerDay != 0 {
		t.Errorf("missing quota should default MaxChecksPerDay to 0, got %d", user.Quota.MaxChecksPerDay)
	}
	if user.Quota.SchedulingPaused {
		t.Error("missing quota should default SchedulingPaused to false")
	}
}
