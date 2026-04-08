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

package checker

import (
	"fmt"
	"testing"
	"time"

	"git.happydns.org/happyDomain/model"
)

func mkExec(id string, age time.Duration, now time.Time) *happydns.Execution {
	return &happydns.Execution{
		Id:        happydns.Identifier(id),
		CheckerID: "ping",
		Target:    happydns.CheckTarget{DomainId: "example.com"},
		StartedAt: now.Add(-age),
	}
}

func TestDecide_Empty(t *testing.T) {
	p := DefaultRetentionPolicy(365)
	keep, drop := p.Decide(nil, time.Now())
	if len(keep) != 0 || len(drop) != 0 {
		t.Fatalf("expected empty results, got keep=%d drop=%d", len(keep), len(drop))
	}
}

func TestDecide_FullDetailWindow(t *testing.T) {
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	p := DefaultRetentionPolicy(365)

	var execs []*happydns.Execution
	for i := 0; i < 50; i++ {
		execs = append(execs, mkExec(fmt.Sprintf("e%d", i), time.Duration(i)*time.Hour, now))
	}

	keep, drop := p.Decide(execs, now)
	if len(drop) != 0 {
		t.Fatalf("expected no drops in <7d window, got %d", len(drop))
	}
	if len(keep) != 50 {
		t.Fatalf("expected 50 keeps, got %d", len(keep))
	}
}

func TestDecide_DailyBucket(t *testing.T) {
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	p := DefaultRetentionPolicy(365)

	// 10 executions on the same day, ~10 days ago (inside daily window).
	var execs []*happydns.Execution
	for i := 0; i < 10; i++ {
		execs = append(execs, mkExec(fmt.Sprintf("e%d", i), 10*24*time.Hour+time.Duration(i)*time.Hour, now))
	}

	keep, drop := p.Decide(execs, now)
	if len(keep) != p.PerDayKept {
		t.Fatalf("expected %d keeps in daily bucket, got %d", p.PerDayKept, len(keep))
	}
	if len(drop) != 10-p.PerDayKept {
		t.Fatalf("expected %d drops, got %d", 10-p.PerDayKept, len(drop))
	}
}

func TestDecide_WeeklyBucket(t *testing.T) {
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	p := DefaultRetentionPolicy(365)

	// 8 executions in the same ISO week, ~60 days ago (inside weekly window).
	var execs []*happydns.Execution
	base := 60 * 24 * time.Hour
	for i := 0; i < 8; i++ {
		execs = append(execs, mkExec(fmt.Sprintf("e%d", i), base+time.Duration(i)*time.Hour, now))
	}

	keep, drop := p.Decide(execs, now)
	if len(keep) != p.PerWeekKept {
		t.Fatalf("expected %d keeps in weekly bucket, got %d", p.PerWeekKept, len(keep))
	}
	if len(drop) != 8-p.PerWeekKept {
		t.Fatalf("expected %d drops, got %d", 8-p.PerWeekKept, len(drop))
	}
}

func TestDecide_MonthlyBucket(t *testing.T) {
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	p := DefaultRetentionPolicy(365)

	// 6 executions in the same calendar month, ~300 days ago (inside monthly window,
	// beyond weekly window which is 365/2 = 182 days).
	var execs []*happydns.Execution
	base := 300 * 24 * time.Hour
	for i := 0; i < 6; i++ {
		execs = append(execs, mkExec(fmt.Sprintf("e%d", i), base+time.Duration(i)*time.Hour, now))
	}

	keep, drop := p.Decide(execs, now)
	if len(keep) != p.PerMonthKept {
		t.Fatalf("expected %d keeps in monthly bucket, got %d", p.PerMonthKept, len(keep))
	}
	if len(drop) != 6-p.PerMonthKept {
		t.Fatalf("expected %d drops, got %d", 6-p.PerMonthKept, len(drop))
	}
}

func TestDecide_ZeroBucketCountsClamped(t *testing.T) {
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	p := DefaultRetentionPolicy(365)
	p.PerDayKept = 0

	// 5 executions ~10 days ago (daily bucket).
	var execs []*happydns.Execution
	for i := 0; i < 5; i++ {
		execs = append(execs, mkExec(fmt.Sprintf("e%d", i), 10*24*time.Hour+time.Duration(i)*time.Hour, now))
	}

	keep, drop := p.Decide(execs, now)
	// Clamped to 1, so exactly 1 kept.
	if len(keep) != 1 {
		t.Fatalf("expected 1 keep after clamping PerDayKept=0 to 1, got %d", len(keep))
	}
	if len(drop) != 4 {
		t.Fatalf("expected 4 drops, got %d", len(drop))
	}
}

func TestDecide_HardCutoff(t *testing.T) {
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	p := DefaultRetentionPolicy(30)

	execs := []*happydns.Execution{
		mkExec("recent", 1*24*time.Hour, now),
		mkExec("old", 100*24*time.Hour, now),
	}

	keep, drop := p.Decide(execs, now)
	if len(keep) != 1 || string(keep[0]) != "recent" {
		t.Fatalf("expected 'recent' to be kept, got %v", keep)
	}
	if len(drop) != 1 || string(drop[0]) != "old" {
		t.Fatalf("expected 'old' to be dropped, got %v", drop)
	}
}

func TestDecide_GroupedByTarget(t *testing.T) {
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	p := DefaultRetentionPolicy(365)

	// 5 executions same day, 10 days ago, two different targets.
	mk := func(id, dom string) *happydns.Execution {
		return &happydns.Execution{
			Id:        happydns.Identifier(id),
			CheckerID: "ping",
			Target:    happydns.CheckTarget{DomainId: dom},
			StartedAt: now.Add(-10 * 24 * time.Hour),
		}
	}
	var execs []*happydns.Execution
	for i := 0; i < 5; i++ {
		execs = append(execs, mk(fmt.Sprintf("a%d", i), "a.example"))
		execs = append(execs, mk(fmt.Sprintf("b%d", i), "b.example"))
	}

	keep, _ := p.Decide(execs, now)
	// PerDayKept per group => 2 * 2 groups = 4
	if len(keep) != 2*p.PerDayKept {
		t.Fatalf("expected %d keeps, got %d", 2*p.PerDayKept, len(keep))
	}
}
