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

package adapter

import (
	"testing"

	"git.happydns.org/happyDomain/model"
)

// TestDNSControlRRtoRC_EveryType runs every record type through the conversion
// to DNSControl and back, the way a zone published to a provider goes:
// happyDomain record -> RecordConfig -> ToRR() -> RecordConfig. The second
// conversion is the one prepare_changes does on the records read back, and the
// one the rtype wrappers (DS, RP, …) used to fail.
func TestDNSControlRRtoRC_EveryType(t *testing.T) {
	for _, name := range sampleNames {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rr := parseSample(t, name)

			rcs, err := DNSControlRRtoRC([]happydns.Record{rr}, rrTypesOrigin)
			if err != nil {
				t.Fatalf("DNSControlRRtoRC: %v", err)
			}
			if len(rcs) != 1 {
				t.Fatalf("got %d RecordConfig, expected 1", len(rcs))
			}

			if got := rcs[0].Type; got != name {
				t.Errorf("RecordConfig type is %q, expected %q", got, name)
			}
			if got := rcs[0].GetLabelFQDN(); got != "sample.example.com" {
				t.Errorf("RecordConfig label is %q, expected \"sample.example.com\"", got)
			}
			if got := rcs[0].TTL; got != 300 {
				t.Errorf("RecordConfig TTL is %d, expected 300", got)
			}

			back := rcs[0].ToRR()
			if back == nil {
				t.Fatalf("RecordConfig.ToRR() gave no record back")
			}
			if got, expected := back.String(), rr.String(); got != expected {
				t.Fatalf("the conversion altered the record: got %q, expected %q", got, expected)
			}

			// The record DNSControl hands back must be convertible again: this
			// is the round trip prepare_changes makes.
			rcs, err = DNSControlRRtoRC([]happydns.Record{back}, rrTypesOrigin)
			if err != nil {
				t.Fatalf("DNSControlRRtoRC on the record given back by ToRR(): %v", err)
			}
			if got, expected := rcs[0].ToRR().String(), rr.String(); got != expected {
				t.Errorf("the round trip altered the record: got %q, expected %q", got, expected)
			}
		})
	}
}

// TestDNSControlDiffByRecord_EveryType checks the diff engine, which every
// publication goes through, handles each type: a zone holding the record
// against an empty one gives exactly one addition, and the record it carries is
// the one we started from.
func TestDNSControlDiffByRecord_EveryType(t *testing.T) {
	for _, name := range sampleNames {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rr := parseSample(t, name)

			corrections, nb, err := DNSControlDiffByRecord(nil, []happydns.Record{rr}, rrTypesOrigin)
			if err != nil {
				t.Fatalf("DNSControlDiffByRecord: %v", err)
			}
			if nb != 1 {
				t.Fatalf("got %d corrections, expected 1", nb)
			}
			if corrections[0].Kind != happydns.CorrectionKindAddition {
				t.Errorf("correction kind is %v, expected an addition", corrections[0].Kind)
			}
			if len(corrections[0].NewRecords) != 1 {
				t.Fatalf("the correction carries %d records, expected 1", len(corrections[0].NewRecords))
			}
			if got, expected := corrections[0].NewRecords[0].String(), rr.String(); got != expected {
				t.Errorf("the diff altered the record: got %q, expected %q", got, expected)
			}

			// The same zone on both sides gives nothing to do.
			_, nb, err = DNSControlDiffByRecord([]happydns.Record{rr}, []happydns.Record{rr}, rrTypesOrigin)
			if err != nil {
				t.Fatalf("DNSControlDiffByRecord on two identical zones: %v", err)
			}
			if nb != 0 {
				t.Errorf("got %d corrections between two identical zones, expected none", nb)
			}
		})
	}
}
