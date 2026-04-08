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
	"sort"
	"time"

	"git.happydns.org/happyDomain/model"
)

// RetentionPolicy describes how check executions are thinned out as they age.
//
// The policy is intentionally tiered: users care about full detail for recent
// runs, but only need sparse historical samples to spot long-term trends.
//
// Default behaviour, given a RetentionDays of D:
//
//	  age window                | kept
//	  ------------------------- | ------------------------------------------
//	  0 .. 7 days               | every execution
//	  7 .. 30 days              | up to 2 executions per day per (checker,target)
//	  30 .. D/2 days            | up to 1 execution per week per (checker,target)
//	  D/2 .. D days             | up to 1 execution per month per (checker,target)
//	  > D days                  | dropped
//
// All thresholds and bucket counts are configurable so the policy can be
// tuned per-user via the admin UserQuota.
type RetentionPolicy struct {
	// RetentionDays is the hard cap on age. Executions older than this are
	// always dropped. Must be > 0.
	RetentionDays int

	// FullDetailDays: every execution kept under this age.
	FullDetailDays int
	// DailyBucketDays: between FullDetailDays and DailyBucketDays, keep
	// PerDayKept executions per UTC day per (checker,target).
	DailyBucketDays int
	PerDayKept      int
	// WeeklyBucketDays: between DailyBucketDays and WeeklyBucketDays, keep
	// PerWeekKept executions per ISO week per (checker,target).
	WeeklyBucketDays int
	PerWeekKept      int
	// Beyond WeeklyBucketDays and up to RetentionDays, keep PerMonthKept
	// executions per calendar month per (checker,target).
	PerMonthKept int
}

// DefaultRetentionPolicy returns the standard tiered policy for the given
// retention horizon.
func DefaultRetentionPolicy(retentionDays int) RetentionPolicy {
	if retentionDays <= 0 {
		retentionDays = 365
	}
	return RetentionPolicy{
		RetentionDays:    retentionDays,
		FullDetailDays:   7,
		DailyBucketDays:  30,
		PerDayKept:       2,
		WeeklyBucketDays: max(retentionDays/2, 31),
		PerWeekKept:      1,
		PerMonthKept:     1,
	}
}

// Decide partitions executions into the ones to keep and the ones to drop
// according to the policy. The function is pure: it does not touch storage.
//
// Executions are grouped by (CheckerID, Target) and ordered most-recent-first
// inside each group, so the newest execution in a bucket is the one preserved.
func (p RetentionPolicy) Decide(executions []*happydns.Execution, now time.Time) (keep, drop []happydns.Identifier) {
	if len(executions) == 0 {
		return nil, nil
	}

	// Clamp bucket counts: a zero or negative value would silently drop
	// every execution in that tier, which is almost certainly a
	// misconfiguration rather than intent.
	if p.PerHourKept < 1 {
		p.PerHourKept = 1
	}
	if p.PerDayKept < 1 {
		p.PerDayKept = 1
	}
	if p.PerWeekKept < 1 {
		p.PerWeekKept = 1
	}
	if p.PerMonthKept < 1 {
		p.PerMonthKept = 1
	}

	// Group by (checker, target).
	groups := map[string][]*happydns.Execution{}
	for _, e := range executions {
		if e == nil {
			continue
		}
		key := e.CheckerID + "|" + e.Target.String()
		groups[key] = append(groups[key], e)
	}

	hardCutoff := now.AddDate(0, 0, -p.RetentionDays)
	fullCutoff := now.AddDate(0, 0, -p.FullDetailDays)
	dailyCutoff := now.AddDate(0, 0, -p.DailyBucketDays)
	weeklyCutoff := now.AddDate(0, 0, -p.WeeklyBucketDays)

	for _, group := range groups {
		// Most recent first.
		sort.Slice(group, func(i, j int) bool {
			return group[i].StartedAt.After(group[j].StartedAt)
		})

		dayBuckets := map[string]int{}
		weekBuckets := map[string]int{}
		monthBuckets := map[string]int{}

		for _, e := range group {
			t := e.StartedAt
			switch {
			case t.Before(hardCutoff):
				drop = append(drop, e.Id)
			case !t.Before(fullCutoff):
				// 0 .. FullDetailDays - keep everything.
				keep = append(keep, e.Id)
			case !t.Before(dailyCutoff):
				k := t.UTC().Format("2006-01-02")
				if dayBuckets[k] < p.PerDayKept {
					dayBuckets[k]++
					keep = append(keep, e.Id)
				} else {
					drop = append(drop, e.Id)
				}
			case !t.Before(weeklyCutoff):
				y, w := t.UTC().ISOWeek()
				k := isoWeekKey(y, w)
				if weekBuckets[k] < p.PerWeekKept {
					weekBuckets[k]++
					keep = append(keep, e.Id)
				} else {
					drop = append(drop, e.Id)
				}
			default:
				k := t.UTC().Format("2006-01")
				if monthBuckets[k] < p.PerMonthKept {
					monthBuckets[k]++
					keep = append(keep, e.Id)
				} else {
					drop = append(drop, e.Id)
				}
			}
		}
	}

	return keep, drop
}

func isoWeekKey(year, week int) string {
	return fmt.Sprintf("%d-W%02d", year, week)
}
