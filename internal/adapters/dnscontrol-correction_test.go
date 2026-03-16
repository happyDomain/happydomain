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

package adapter_test

import (
	"net"
	"testing"

	"github.com/miekg/dns"

	adapter "git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

func makeA(name string, ip string) happydns.Record {
	return &dns.A{
		Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
		A:   net.ParseIP(ip),
	}
}

func makeMX(name string, pref uint16, mx string) happydns.Record {
	return &dns.MX{
		Hdr:        dns.RR_Header{Name: name, Rrtype: dns.TypeMX, Class: dns.ClassINET, Ttl: 300},
		Preference: pref,
		Mx:         mx,
	}
}

func TestBuildTargetRecords_AllSelected(t *testing.T) {
	providerRecords := []happydns.Record{
		makeA("example.com.", "1.2.3.4"),
	}

	newRecord := makeA("example.com.", "5.6.7.8")
	corrections := []*happydns.Correction{
		{
			Id:         happydns.Identifier([]byte("add-1")),
			Kind:       happydns.CorrectionKindAddition,
			NewRecords: []happydns.Record{newRecord},
		},
	}

	selectedIDs := []happydns.Identifier{
		happydns.Identifier([]byte("add-1")),
	}

	result := adapter.BuildTargetRecords(providerRecords, corrections, selectedIDs)
	if len(result) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result))
	}
}

func TestBuildTargetRecords_NoneSelected(t *testing.T) {
	providerRecords := []happydns.Record{
		makeA("example.com.", "1.2.3.4"),
	}

	corrections := []*happydns.Correction{
		{
			Id:         happydns.Identifier([]byte("add-1")),
			Kind:       happydns.CorrectionKindAddition,
			NewRecords: []happydns.Record{makeA("example.com.", "5.6.7.8")},
		},
	}

	result := adapter.BuildTargetRecords(providerRecords, corrections, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result))
	}
	if result[0].String() != providerRecords[0].String() {
		t.Errorf("expected unchanged provider record, got %s", result[0].String())
	}
}

func TestBuildTargetRecords_Deletion(t *testing.T) {
	providerRecords := []happydns.Record{
		makeA("example.com.", "1.2.3.4"),
		makeA("example.com.", "5.6.7.8"),
	}

	corrections := []*happydns.Correction{
		{
			Id:         happydns.Identifier([]byte("del-1")),
			Kind:       happydns.CorrectionKindDeletion,
			OldRecords: []happydns.Record{makeA("example.com.", "1.2.3.4")},
		},
	}

	selectedIDs := []happydns.Identifier{
		happydns.Identifier([]byte("del-1")),
	}

	result := adapter.BuildTargetRecords(providerRecords, corrections, selectedIDs)
	if len(result) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result))
	}
	if result[0].String() != providerRecords[1].String() {
		t.Errorf("expected remaining record %s, got %s", providerRecords[1].String(), result[0].String())
	}
}

func TestBuildTargetRecords_Update(t *testing.T) {
	oldRecord := makeA("example.com.", "1.2.3.4")
	newRecord := makeA("example.com.", "9.8.7.6")

	providerRecords := []happydns.Record{oldRecord}

	corrections := []*happydns.Correction{
		{
			Id:         happydns.Identifier([]byte("upd-1")),
			Kind:       happydns.CorrectionKindUpdate,
			OldRecords: []happydns.Record{oldRecord},
			NewRecords: []happydns.Record{newRecord},
		},
	}

	selectedIDs := []happydns.Identifier{
		happydns.Identifier([]byte("upd-1")),
	}

	result := adapter.BuildTargetRecords(providerRecords, corrections, selectedIDs)
	if len(result) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result))
	}
	if result[0].String() != newRecord.String() {
		t.Errorf("expected updated record %s, got %s", newRecord.String(), result[0].String())
	}
}

func TestBuildTargetRecords_PartialSelection(t *testing.T) {
	providerRecords := []happydns.Record{
		makeA("example.com.", "1.2.3.4"),
	}

	corrections := []*happydns.Correction{
		{
			Id:         happydns.Identifier([]byte("add-1")),
			Kind:       happydns.CorrectionKindAddition,
			NewRecords: []happydns.Record{makeA("example.com.", "5.6.7.8")},
		},
		{
			Id:         happydns.Identifier([]byte("add-2")),
			Kind:       happydns.CorrectionKindAddition,
			NewRecords: []happydns.Record{makeMX("example.com.", 10, "mail.example.com.")},
		},
	}

	// Only select the first correction.
	selectedIDs := []happydns.Identifier{
		happydns.Identifier([]byte("add-1")),
	}

	result := adapter.BuildTargetRecords(providerRecords, corrections, selectedIDs)
	if len(result) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result))
	}
}

func TestBuildTargetRecords_MixedOperations(t *testing.T) {
	providerRecords := []happydns.Record{
		makeA("example.com.", "1.2.3.4"),
		makeA("example.com.", "10.0.0.1"),
	}

	corrections := []*happydns.Correction{
		{
			Id:         happydns.Identifier([]byte("del-1")),
			Kind:       happydns.CorrectionKindDeletion,
			OldRecords: []happydns.Record{makeA("example.com.", "10.0.0.1")},
		},
		{
			Id:         happydns.Identifier([]byte("add-1")),
			Kind:       happydns.CorrectionKindAddition,
			NewRecords: []happydns.Record{makeA("example.com.", "5.6.7.8")},
		},
	}

	selectedIDs := []happydns.Identifier{
		happydns.Identifier([]byte("del-1")),
		happydns.Identifier([]byte("add-1")),
	}

	result := adapter.BuildTargetRecords(providerRecords, corrections, selectedIDs)
	if len(result) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result))
	}

	// Should have 1.2.3.4 and 5.6.7.8 (10.0.0.1 deleted, 5.6.7.8 added)
	found := map[string]bool{}
	for _, r := range result {
		found[r.String()] = true
	}
	if !found[makeA("example.com.", "1.2.3.4").String()] {
		t.Error("expected record 1.2.3.4 to remain")
	}
	if !found[makeA("example.com.", "5.6.7.8").String()] {
		t.Error("expected record 5.6.7.8 to be added")
	}
}

func TestDNSControlDiffByRecord_EnrichedFields(t *testing.T) {
	oldRecords := []happydns.Record{
		makeA("example.com.", "1.2.3.4"),
	}

	newRecords := []happydns.Record{
		makeA("example.com.", "1.2.3.4"),
		makeA("example.com.", "5.6.7.8"),
	}

	corrections, nbDiffs, err := adapter.DNSControlDiffByRecord(oldRecords, newRecords, "example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if nbDiffs == 0 {
		t.Fatal("expected at least 1 diff")
	}

	if len(corrections) == 0 {
		t.Fatal("expected at least 1 correction")
	}

	for _, c := range corrections {
		if len(c.Id) == 0 {
			t.Error("expected correction to have an ID")
		}

		switch c.Kind {
		case happydns.CorrectionKindAddition:
			if len(c.NewRecords) == 0 {
				t.Error("addition correction should have NewRecords")
			}
		case happydns.CorrectionKindDeletion:
			if len(c.OldRecords) == 0 {
				t.Error("deletion correction should have OldRecords")
			}
		case happydns.CorrectionKindUpdate:
			if len(c.OldRecords) == 0 || len(c.NewRecords) == 0 {
				t.Error("update correction should have both OldRecords and NewRecords")
			}
		}
	}
}

func TestDNSControlDiffByRecord_NoChanges(t *testing.T) {
	records := []happydns.Record{
		makeA("example.com.", "1.2.3.4"),
	}

	corrections, _, err := adapter.DNSControlDiffByRecord(records, records, "example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(corrections) != 0 {
		t.Errorf("expected 0 corrections for identical zones, got %d", len(corrections))
	}
}
