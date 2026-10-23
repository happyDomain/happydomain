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

package discovery

import (
	"context"
	"net"
	"testing"
	"time"
)

func axfrTestZone(t *testing.T) *testZone {
	return newTestZone(t, "example.com.",
		"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"hidden.example.com. 300 IN A 192.0.2.30",
	)
}

func TestAXFRFactory(t *testing.T) {
	f := factory(t, "axfr")

	if _, ok := f.New(settingsFrom(t, `{}`)).(AXFRSource); !ok {
		t.Error("the zone transfer is disabled by default")
	}
	if source := f.New(settingsFrom(t, `{"disable_axfr":true}`)); source != nil {
		t.Errorf("disable_axfr gives %v; want no source", source)
	}
}

func TestResolveAXFR(t *testing.T) {
	zone := axfrTestZone(t)
	zone.allowAXFR = true
	opts := zone.serve(t)

	records, err := Resolve(context.Background(), "example.com.", nil, []NameSource{AXFRSource{}}, opts)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	got := recordStrings(records)

	// A name nothing points to, only a transfer reveals.
	assertHas(t, got, "hidden.example.com. 300 IN A 192.0.2.30")
	if len(got) != 4 {
		t.Errorf("Resolve returned %d records; want the 4 of the zone, the SOA once:\n%v", len(got), got)
	}
}

func TestResolveAXFRRefused(t *testing.T) {
	zone := axfrTestZone(t)
	opts := zone.serve(t)

	records, err := Resolve(context.Background(), "example.com.", nil, []NameSource{AXFRSource{}}, opts)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	got := recordStrings(records)

	// The exploration takes over.
	assertHas(t, got,
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
	)
	assertHasNoName(t, got, "hidden.example.com.")
}

func TestResolveAXFROutOfZone(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"www.example.net. 300 IN A 192.0.2.40",
		"notexample.com. 300 IN A 192.0.2.41",
	)
	zone.allowAXFR = true
	opts := zone.serve(t)

	records, err := Resolve(context.Background(), "example.com.", nil, []NameSource{AXFRSource{}}, opts)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	got := recordStrings(records)

	assertHas(t, got, "ns1.example.com. 300 IN A 127.0.0.1")
	assertHasNoName(t, got, "www.example.net.")
	assertHasNoName(t, got, "notexample.com.")
}

func TestAXFRRecordsCap(t *testing.T) {
	zone := axfrTestZone(t)
	zone.allowAXFR = true
	opts := zone.serve(t)

	defer func(max int) { maxAXFRRecords = max }(maxAXFRRecords)
	maxAXFRRecords = 2

	if _, err := transfer(context.Background(), "example.com.", net.JoinHostPort("127.0.0.1", opts.authPort)); err == nil {
		t.Error("a transfer beyond the cap succeeded")
	}
}

func TestAXFRBoundByContext(t *testing.T) {
	// A server accepting the transfer, then never answering.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			defer conn.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	if _, err := transfer(ctx, "example.com.", ln.Addr().String()); err == nil {
		t.Error("a transfer never answered succeeded")
	}
	if elapsed := time.Since(start); elapsed > axfrTimeout/2 {
		t.Errorf("the transfer took %s; want it cut at the end of the context", elapsed)
	}
}
