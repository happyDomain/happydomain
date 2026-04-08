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

	// 20 executions in the first 20 minutes, all inside 0..1 day window.
	var execs []*happydns.Execution
	for i := 0; i < 20; i++ {
		execs = append(execs, mkExec(fmt.Sprintf("e%d", i), time.Duration(i)*time.Minute, now))
	}

	keep, drop := p.Decide(execs, now)
	if len(drop) != 0 {
		t.Fatalf("expected no drops in <1d window, got %d", len(drop))
	}
	if len(keep) != 20 {
		t.Fatalf("expected 20 keeps, got %d", len(keep))
	}
}

func TestDecide_HourlyBucket(t *testing.T) {
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	p := DefaultRetentionPolicy(365)

	// 6 executions in the same hour ~3 days ago (inside hourly window).
	var execs []*happydns.Execution
	base := 3*24*time.Hour + 30*time.Minute
	for i := 0; i < 6; i++ {
		execs = append(execs, mkExec(fmt.Sprintf("e%d", i), base+time.Duration(i)*time.Minute, now))
	}

	keep, drop := p.Decide(execs, now)
	if len(keep) != p.PerHourKept {
		t.Fatalf("expected %d keeps in hourly bucket, got %d", p.PerHourKept, len(keep))
	}
	if len(drop) != 6-p.PerHourKept {
		t.Fatalf("expected %d drops, got %d", 6-p.PerHourKept, len(drop))
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

func TestDecide_SmallRetentionCollapseTiers(t *testing.T) {
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	p := DefaultRetentionPolicy(3)

	// With retentionDays=3, tiers collapse:
	//   FullDetailDays=1, HourlyBucketDays=3, DailyBucketDays=3,
	//   WeeklyBucketDays=3 - only full-detail and hourly tiers are reachable.

	var execs []*happydns.Execution
	// 3 executions inside full-detail window (< 1 day).
	for i := 0; i < 3; i++ {
		execs = append(execs, mkExec(fmt.Sprintf("recent%d", i), time.Duration(i)*time.Minute, now))
	}
	// 4 executions in the same hour, ~2 days ago (hourly tier).
	base := 2*24*time.Hour + 30*time.Minute
	for i := 0; i < 4; i++ {
		execs = append(execs, mkExec(fmt.Sprintf("hourly%d", i), base+time.Duration(i)*time.Minute, now))
	}
	// 1 execution beyond retention (5 days ago).
	execs = append(execs, mkExec("expired", 5*24*time.Hour, now))

	keep, drop := p.Decide(execs, now)
	// 3 full-detail + 1 hourly kept + 3 hourly dropped + 1 expired dropped
	if len(keep) != 3+p.PerHourKept {
		t.Fatalf("expected %d keeps, got %d", 3+p.PerHourKept, len(keep))
	}
	if len(drop) != 4-p.PerHourKept+1 {
		t.Fatalf("expected %d drops, got %d", 4-p.PerHourKept+1, len(drop))
	}
}

func TestDecide_BoundaryFullDetailToHourly(t *testing.T) {
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	p := DefaultRetentionPolicy(365)

	// Execution exactly at the full-detail boundary (age == exactly 1 day).
	// !t.Before(fullCutoff) is true when t == fullCutoff, so this lands in full-detail.
	exactBoundary := mkExec("boundary", 24*time.Hour, now)
	// Execution 1 second past the boundary (age == 1 day + 1s) lands in hourly.
	pastBoundary := mkExec("past", 24*time.Hour+time.Second, now)

	keep, drop := p.Decide([]*happydns.Execution{exactBoundary, pastBoundary}, now)
	// Both should be kept (one as full-detail, one as hourly).
	if len(keep) != 2 {
		t.Fatalf("expected 2 keeps, got %d (keep=%v, drop=%v)", len(keep), keep, drop)
	}
	if len(drop) != 0 {
		t.Fatalf("expected 0 drops, got %d", len(drop))
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
