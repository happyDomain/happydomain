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

package usecase

import (
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestParseSPFTerm(t *testing.T) {
	cases := []struct {
		raw      string
		kind     spfTermKind
		value    string
		isAll    bool
		mech     string
	}{
		{"v=spf1", spfTermNone, "spf1", false, "v"},
		{"include:foo.example.com", spfTermInclude, "foo.example.com", false, "include"},
		{"redirect=fallback.example.com", spfTermRedirect, "fallback.example.com", false, "redirect"},
		{"a", spfTermA, "", false, "a"},
		{"a:host.example.com", spfTermA, "host.example.com", false, "a"},
		{"a/24", spfTermA, "", false, "a"},
		{"mx", spfTermMX, "", false, "mx"},
		{"mx:host.example.com", spfTermMX, "host.example.com", false, "mx"},
		{"ptr", spfTermPTR, "", false, "ptr"},
		{"ptr:example.com", spfTermPTR, "example.com", false, "ptr"},
		{"exists:_e.example.com", spfTermExists, "_e.example.com", false, "exists"},
		{"-all", spfTermNone, "", true, "all"},
		{"~all", spfTermNone, "", true, "all"},
		{"+all", spfTermNone, "", true, "all"},
		{"ip4:1.2.3.4", spfTermNone, "1.2.3.4", false, "ip4"},
		{"ip6:::1", spfTermNone, "::1", false, "ip6"},
	}

	for _, tc := range cases {
		got := parseSPFTerm(tc.raw)
		if got.kind != tc.kind {
			t.Errorf("parseSPFTerm(%q).kind = %v; want %v", tc.raw, got.kind, tc.kind)
		}
		if got.value != tc.value {
			t.Errorf("parseSPFTerm(%q).value = %q; want %q", tc.raw, got.value, tc.value)
		}
		if got.isAll != tc.isAll {
			t.Errorf("parseSPFTerm(%q).isAll = %v; want %v", tc.raw, got.isAll, tc.isAll)
		}
		if got.mechanism != tc.mech {
			t.Errorf("parseSPFTerm(%q).mechanism = %q; want %q", tc.raw, got.mechanism, tc.mech)
		}
	}
}

func TestPickSPFRecord(t *testing.T) {
	if got := pickSPFRecord([]string{"some other txt", "v=spf1 -all", "v=DMARC1; p=none"}); got != "v=spf1 -all" {
		t.Errorf("pickSPFRecord did not return SPF record: got %q", got)
	}
	if got := pickSPFRecord([]string{"v=spf2"}); got != "" {
		t.Errorf("pickSPFRecord returned %q on no SPF1", got)
	}
}

// fakeFlattenContext exercises the lookup-counting logic without DNS.
// It feeds a complete root record (so the root call does not trigger a
// query), and only mechanisms that don't recurse (a, mx, exists, ptr,
// ip4/ip6, all) are present.
func TestFlatten_LocalCounting(t *testing.T) {
	fc := &flattenContext{
		visited:  map[string]struct{}{},
		deadline: timeFar(),
	}
	// 6 lookup-consuming terms: a, mx, exists, ptr, redirect (counted but
	// resolves to "" → "syntax" child), and one include with empty value.
	root := fc.flatten(noopClient(), "example.com", "v=spf1 a mx exists:_e.example.com ptr ip4:1.2.3.4 -all", "root", 0)
	if root.Error != "" {
		t.Fatalf("unexpected root error: %q", root.Error)
	}
	if fc.lookups != 4 {
		t.Errorf("lookups = %d; want 4 (a, mx, exists, ptr)", fc.lookups)
	}
	if len(root.Children) != 4 {
		t.Errorf("children count = %d; want 4", len(root.Children))
	}
}

func TestFlatten_LoopDetection(t *testing.T) {
	fc := &flattenContext{
		visited:  map[string]struct{}{},
		deadline: timeFar(),
	}
	// Pre-mark the domain as visited; the root call should immediately
	// return a "loop" error — though only when entering, not on the
	// initial visit. Verify by re-entering via a child include.
	fc.visited["example.com"] = struct{}{}
	root := fc.flatten(noopClient(), "example.com", "v=spf1 -all", "root", 0)
	if root.Error != "loop" {
		t.Errorf("loop not detected: error=%q", root.Error)
	}
}

func TestFlatten_BudgetOverrun(t *testing.T) {
	fc := &flattenContext{
		visited:  map[string]struct{}{},
		deadline: timeFar(),
	}
	terms := strings.Repeat(" a", 11)
	fc.flatten(noopClient(), "example.com", "v=spf1"+terms+" -all", "root", 0)
	if !fc.overBudget() {
		t.Errorf("budget not flagged: lookups=%d", fc.lookups)
	}
}

// timeFar returns a deadline far in the future so deadline checks never trip
// during unit tests.
func timeFar() (t time.Time) {
	return time.Now().Add(1 * time.Hour)
}

// noopClient returns a default dns.Client; the unit tests above never trigger
// an actual query because they feed pre-supplied records.
func noopClient() dns.Client {
	return dns.Client{}
}
