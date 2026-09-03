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
	"time"

	"github.com/libdns/libdns"

	"git.happydns.org/happyDomain/model"
)

// sampleLibdnsRR builds the libdns.RR a provider would hand back for the given
// type and rdata.
func sampleLibdnsRR(rrType, data string) libdns.RR {
	return libdns.RR{Name: "sample", TTL: 300 * time.Second, Type: rrType, Data: data}
}

// TestLibdnsRoundTrip_EveryType runs every record type through the libdns
// conversion and back.
//
// Unlike the DNSControl one, this boundary is made of text: the rdata leaves as
// the tail of the miekg zone file line (extractRdata) and comes back parsed
// from a line rebuilt around it. A type whose presentation format the parser
// reads differently from the way it writes it does not survive the trip.
func TestLibdnsRoundTrip_EveryType(t *testing.T) {
	for _, name := range sampleNames {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rr := parseSample(t, name)

			out := happyDNSRecordToLibdnsRR(rr, rrTypesOrigin)

			if got := out.Name; got != "sample" {
				t.Errorf("libdns name is %q, expected \"sample\"", got)
			}
			if got := out.Type; got != name {
				t.Errorf("libdns type is %q, expected %q", got, name)
			}
			if got, expected := out.TTL, 300*time.Second; got != expected {
				t.Errorf("libdns TTL is %v, expected %v", got, expected)
			}
			if out.Data == "" {
				t.Fatalf("libdns rdata is empty, the record was %q", rr.String())
			}

			back, err := libdnsToHappyDNSRecord(out, rrTypesOrigin)
			if err != nil {
				t.Fatalf("converting %q back: %v", out.Data, err)
			}

			if got, expected := back.String(), rr.String(); got != expected {
				t.Errorf("the round trip altered the record: got %q, expected %q", got, expected)
			}
		})
	}
}

// TestLibdnsImport_EveryType reads a zone holding one record of each type from
// a provider, and asks for the corrections towards that very same zone. A
// record the import spells differently from the way the diff engine hands it
// back shows up as a correction: the provider would then be asked, at every
// publication, to apply a change that changes nothing.
func TestLibdnsImport_EveryType(t *testing.T) {
	for _, name := range sampleNames {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rr := parseSample(t, name)

			mock := &mockLibdnsProvider{records: []libdns.Record{
				sampleLibdnsRR(name, rrSamples[name]),
			}}

			adapter, err := NewLibdnsProviderAdapter(&mockLibdnsConfig{provider: mock})
			if err != nil {
				t.Fatalf("NewLibdnsProviderAdapter: %v", err)
			}

			imported, err := adapter.GetZoneRecords("example.com")
			if err != nil {
				t.Fatalf("GetZoneRecords: %v", err)
			}
			if len(imported) != 1 {
				t.Fatalf("imported %d records, expected 1", len(imported))
			}
			if got, expected := imported[0].String(), rr.String(); got != expected {
				t.Errorf("the import altered the record: got %q, expected %q", got, expected)
			}

			_, nb, err := adapter.GetZoneCorrections("example.com", []happydns.Record{rr})
			if err != nil {
				t.Fatalf("GetZoneCorrections: %v", err)
			}
			if nb != 0 {
				t.Errorf("got %d corrections against the zone the provider already holds, expected none", nb)
			}
		})
	}
}

// TestLibdnsRecordKey_EveryType checks the key a record read from a provider
// gets is the one it gets again once the diff engine has handed it back. Those
// two are what resolveOriginalRecords matches to keep the concrete record of
// the provider, and its identifier, on a deletion.
func TestLibdnsRecordKey_EveryType(t *testing.T) {
	for _, name := range sampleNames {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rr := parseSample(t, name)

			// As read from a provider.
			imported, err := libdnsToHappyDNSRecord(
				sampleLibdnsRR(name, rrSamples[name]),
				rrTypesOrigin,
			)
			if err != nil {
				t.Fatalf("importing: %v", err)
			}

			// As handed back by the diff engine, which rebuilds the record from
			// the RecordConfig it made of it.
			rcs, err := DNSControlRRtoRC([]happydns.Record{rr}, rrTypesOrigin)
			if err != nil {
				t.Fatalf("DNSControlRRtoRC: %v", err)
			}
			fromDiff, err := recordFromRecordConfig(rcs[0])
			if err != nil {
				t.Fatalf("recordFromRecordConfig: %v", err)
			}

			if got, expected := libdnsRecordKey(fromDiff, rrTypesOrigin), libdnsRecordKey(imported, rrTypesOrigin); got != expected {
				t.Errorf("the record read from the provider and the one given back by the diff have different keys:\n  provider %q\n      diff %q", expected, got)
			}
		})
	}
}

// TestLibdnsRecordKey_ProviderSpellings covers the spellings a provider is free
// to use for an rdata miekg/dns writes otherwise: the key must not depend on
// them. This is what a deletion matching the record of the provider rests on.
func TestLibdnsRecordKey_ProviderSpellings(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rrType   string
		data     string
		expected string // the same record, spelled the way miekg/dns writes it
	}{
		{"relative MX target", "MX", "10 mail", "10 mail.example.com."},
		{"relative CNAME target", "CNAME", "target", "target.example.com."},
		{"relative SRV target", "SRV", "10 20 5060 sip", "10 20 5060 sip.example.com."},
		{"unquoted TXT", "TXT", "hello world", `"hello world"`},
		{"TXT in several strings", "TXT", `"part one " "part two"`, `"part one part two"`},
		{"unpadded IPv6", "AAAA", "2001:0db8:0000::1", "2001:db8::1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			asProviderSpellsIt, err := libdnsToHappyDNSRecord(
				sampleLibdnsRR(tc.rrType, tc.data),
				rrTypesOrigin,
			)
			if err != nil {
				t.Fatalf("importing %q: %v", tc.data, err)
			}

			asMiekgWritesIt, err := libdnsToHappyDNSRecord(
				sampleLibdnsRR(tc.rrType, tc.expected),
				rrTypesOrigin,
			)
			if err != nil {
				t.Fatalf("importing %q: %v", tc.expected, err)
			}

			got := libdnsRecordKey(asProviderSpellsIt, rrTypesOrigin)
			if expected := libdnsRecordKey(asMiekgWritesIt, rrTypesOrigin); got != expected {
				t.Errorf("%q and %q are the same record but have different keys:\n  got %q\n  expected %q", tc.data, tc.expected, got, expected)
			}
		})
	}
}
