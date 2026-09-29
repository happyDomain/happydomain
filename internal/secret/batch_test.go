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

package secret

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

type batchObject struct{ Name string }

func batchName(o *batchObject) string { return "thing " + o.Name }

func batchEntries() *entriesIterator[batchObject] {
	return newEntriesIterator(
		iterEntry[batchObject]{key: "k-a", item: &batchObject{"a"}},
		iterEntry[batchObject]{key: "k-b", item: &batchObject{"b"}},
		iterEntry[batchObject]{key: "k-bad", err: errors.New("invalid character")},
		iterEntry[batchObject]{key: "k-c", item: &batchObject{"c"}},
		iterEntry[batchObject]{key: "k-d", item: &batchObject{"d"}},
		iterEntry[batchObject]{key: "k-e", item: &batchObject{"e"}},
	)
}

func containsAll(t *testing.T, what string, got []string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !slices.ContainsFunc(got, func(g string) bool { return strings.Contains(g, w) }) {
			t.Errorf("%s = %q, want an entry naming %q", what, got, w)
		}
	}
}

// One object that does not decode, fails or is skipped never stops the
// others: running it again is how a reseal resumes.
func TestResealAllReportsEveryObject(t *testing.T) {
	it := batchEntries()

	var seen []string
	report, err := ResealAll("thing", it, batchName, func(o *batchObject) (bool, error) {
		if !it.closed {
			t.Error("reseal started while the listing was still open")
		}
		seen = append(seen, o.Name)
		switch o.Name {
		case "a":
			return true, nil
		case "c":
			return false, errors.New("does not open")
		case "d":
			return false, fmt.Errorf("stored record: %w", happydns.ErrChangedMeanwhile)
		case "e":
			return false, fmt.Errorf("%w: someone", ErrUnknownOwner)
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("ResealAll: %v", err)
	}

	if !slices.Equal(seen, []string{"a", "b", "c", "d", "e"}) {
		t.Errorf("resealed %v, want every decodable object once", seen)
	}
	if report.ObjectType != "thing" || report.Processed != 6 || report.Changed != 1 || report.Skipped != 2 || report.Failed != 2 {
		t.Errorf("report = %+v, want 6 processed, 1 changed, 2 skipped, 2 failed", report)
	}
	if len(report.Errors) != 4 {
		t.Errorf("errors = %q, want one per skipped or failed object", report.Errors)
	}
	containsAll(t, "errors", report.Errors, "k-bad", "thing c", "thing d", "thing e")
}

func TestResealAllStopsWhenListingFails(t *testing.T) {
	it := newEntriesIterator(iterEntry[batchObject]{key: "k-a", item: &batchObject{"a"}})
	it.listErr = errors.New("storage unavailable")

	_, err := ResealAll("thing", it, batchName, func(*batchObject) (bool, error) {
		t.Error("reseal ran on an incomplete listing")
		return false, nil
	})
	if err == nil {
		t.Error("a failed listing was not reported")
	}
	if !it.closed {
		t.Error("listing not closed")
	}
}

func TestResealAllCapsTheErrorsItReports(t *testing.T) {
	var entries []iterEntry[batchObject]
	for i := range 3 * maxProblems {
		entries = append(entries, iterEntry[batchObject]{key: fmt.Sprint(i), item: &batchObject{fmt.Sprint(i)}})
	}

	report, err := ResealAll("thing", newEntriesIterator(entries...), batchName, func(*batchObject) (bool, error) {
		return false, errors.New("broken")
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Failed != 3*maxProblems {
		t.Errorf("failed = %d, want every object counted", report.Failed)
	}
	if len(report.Errors) != maxProblems+1 || !strings.Contains(report.Errors[maxProblems], fmt.Sprint(2*maxProblems)) {
		t.Errorf("errors = %d entries ending with %q, want %d and a count of the others", len(report.Errors), report.Errors[len(report.Errors)-1], maxProblems+1)
	}
}

// A report on damaged data is when the administrator needs it most: nothing
// but a failed listing stops it.
func TestInspectAllCountsWhatItCannotRead(t *testing.T) {
	counts, err := InspectAll(batchEntries(), batchName, func(o *batchObject, c *Counts) error {
		if o.Name == "c" {
			return errors.New("unknown provider type")
		}
		c.Clear++
		return nil
	})
	if err != nil {
		t.Fatalf("InspectAll: %v", err)
	}
	if counts.Clear != 4 || counts.Undecodable != 2 {
		t.Errorf("counts = %+v, want 4 clear and 2 undecodable", counts)
	}
	containsAll(t, "problems", counts.Problems, "k-bad", "thing c")

	it := batchEntries()
	it.listErr = errors.New("storage unavailable")
	if _, err := InspectAll(it, batchName, func(*batchObject, *Counts) error { return nil }); err == nil {
		t.Error("a failed listing was not reported")
	}
	if !it.closed {
		t.Error("listing not closed")
	}
}
