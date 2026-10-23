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
	"errors"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
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

func TestExploreAXFR(t *testing.T) {
	zone := axfrTestZone(t)
	zone.allowAXFR = true
	opts := zone.serve(t)

	records, err := Explore(context.Background(), "example.com.", nil, []NameSource{AXFRSource{}}, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	got := recordStrings(records)

	// A name nothing points to, only a transfer reveals.
	assertHas(t, got, "hidden.example.com. 300 IN A 192.0.2.30")
	if len(got) != 4 {
		t.Errorf("Explore returned %d records; want the 4 of the zone, the SOA once:\n%v", len(got), got)
	}
	// The 3 questions locating the servers, then the transfer.
	if n := zone.questions.Load(); n != 4 {
		t.Errorf("Explore asked %d questions; want 4, the transfer sparing the exploration", n)
	}
}

func TestExploreAXFRRefused(t *testing.T) {
	zone := axfrTestZone(t)
	opts := zone.serve(t)

	records, err := Explore(context.Background(), "example.com.", nil, []NameSource{AXFRSource{}}, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	got := recordStrings(records)

	// The exploration takes over.
	assertHas(t, got,
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
	)
	assertHasNoName(t, got, "hidden.example.com.")
}

func TestExploreAXFROutOfZone(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"www.example.net. 300 IN A 192.0.2.40",
		"notexample.com. 300 IN A 192.0.2.41",
	)
	zone.allowAXFR = true
	opts := zone.serve(t)

	records, err := Explore(context.Background(), "example.com.", nil, []NameSource{AXFRSource{}}, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	got := recordStrings(records)

	assertHas(t, got, "ns1.example.com. 300 IN A 127.0.0.1")
	assertHasNoName(t, got, "www.example.net.")
	assertHasNoName(t, got, "notexample.com.")
}

// outOfZoneSource hands a zone holding no record of the domain asked.
type outOfZoneSource struct{}

func (outOfZoneSource) ID() string { return "out-of-zone" }

func (outOfZoneSource) Discover(context.Context, Target) ([]Name, error) { return nil, nil }

func (outOfZoneSource) Records(context.Context, Target) ([]dns.RR, error) {
	rr, err := dns.NewRR("www.example.net. 300 IN A 192.0.2.40")
	return []dns.RR{rr}, err
}

func TestExploreAXFRNothingInZone(t *testing.T) {
	zone := axfrTestZone(t)
	opts := zone.serve(t)

	records, err := Explore(context.Background(), "example.com.", nil, []NameSource{outOfZoneSource{}}, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}

	// The exploration takes over.
	assertHas(t, recordStrings(records),
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
	)
}

func TestAXFRRecordsTriesEachServer(t *testing.T) {
	zone := axfrTestZone(t)
	zone.allowAXFR = true
	opts := zone.serve(t)

	// A server refusing the connection, before the one allowing the transfer.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	closed := ln.Addr().String()
	ln.Close()

	rrs, err := (AXFRSource{}).Records(context.Background(), Target{
		Domain:        "example.com.",
		Authoritative: []string{closed, net.JoinHostPort("127.0.0.1", opts.authPort)},
	})
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	var got []string
	for _, rr := range rrs {
		got = append(got, rr.String())
	}
	assertHas(t, got, "hidden.example.com.\t300\tIN\tA\t192.0.2.30")
}

func TestAXFRRecordsCanceled(t *testing.T) {
	zone := axfrTestZone(t)
	zone.allowAXFR = true
	opts := zone.serve(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rrs, err := (AXFRSource{}).Records(ctx, Target{
		Domain:        "example.com.",
		Authoritative: []string{net.JoinHostPort("127.0.0.1", opts.authPort)},
	})
	if !errors.Is(err, context.Canceled) || rrs != nil {
		t.Errorf("Records = %d records, %v; want %v", len(rrs), err, context.Canceled)
	}
	if n := zone.questions.Load(); n != 0 {
		t.Errorf("%d transfers were asked once canceled; want none", n)
	}
}

func TestAXFRMidTransferError(t *testing.T) {
	// A server opening the transfer, then hanging up.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		raw, err := ln.Accept()
		if err != nil {
			return
		}
		conn := &dns.Conn{Conn: raw}
		defer conn.Close()
		req, err := conn.ReadMsg()
		if err != nil {
			return
		}
		soa, _ := dns.NewRR("example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300")
		a, _ := dns.NewRR("www.example.com. 300 IN A 192.0.2.1")
		m := new(dns.Msg)
		m.SetReply(req)
		m.Answer = []dns.RR{soa, a}
		conn.WriteMsg(m)
	}()

	rrs, err := transfer(context.Background(), "example.com.", ln.Addr().String())
	if err == nil {
		t.Error("a transfer cut short succeeded")
	}
	if rrs != nil {
		t.Errorf("a transfer cut short handed %d records", len(rrs))
	}
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

func TestAXFRCapCountsKeptRecords(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"example.com. 300 IN NSEC ns1.example.com. NS SOA RRSIG NSEC",
		"example.com. 300 IN RRSIG NS 13 2 3600 20300101000000 20200101000000 12345 example.com. dGVzdA==",
		"www.example.org. 300 IN A 192.0.2.1",
	)
	zone.allowAXFR = true
	opts := zone.serve(t)

	// The SOA twice, the NS and the A: the signatures and the record of
	// another zone are dropped, and do not count.
	defer func(max int) { maxAXFRRecords = max }(maxAXFRRecords)
	maxAXFRRecords = 4

	rrs, err := transfer(context.Background(), "example.com.", net.JoinHostPort("127.0.0.1", opts.authPort))
	if err != nil {
		t.Fatalf("transfer: %v", err)
	}
	for _, rr := range rrs {
		if rr.Header().Rrtype == dns.TypeNSEC || rr.Header().Rrtype == dns.TypeRRSIG || !dns.IsSubDomain("example.com.", rr.Header().Name) {
			t.Errorf("transfer kept %s", rr)
		}
	}
	if len(rrs) != 4 {
		t.Errorf("transfer kept %d records, want 4", len(rrs))
	}
}

func TestAXFRBytesCap(t *testing.T) {
	zone := axfrTestZone(t)
	zone.allowAXFR = true
	opts := zone.serve(t)

	defer func(max int) { maxAXFRBytes = max }(maxAXFRBytes)
	maxAXFRBytes = 100

	if _, err := transfer(context.Background(), "example.com.", net.JoinHostPort("127.0.0.1", opts.authPort)); err == nil {
		t.Error("a transfer beyond the size cap succeeded")
	}
}

// dripServer opens a transfer, then sends a record every interval, never
// ending it.
func dripServer(t *testing.T, interval time.Duration) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		raw, err := ln.Accept()
		if err != nil {
			return
		}
		conn := &dns.Conn{Conn: raw}
		defer conn.Close()
		req, err := conn.ReadMsg()
		if err != nil {
			return
		}
		soa, _ := dns.NewRR("example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300")
		a, _ := dns.NewRR("www.example.com. 300 IN A 192.0.2.1")
		m := new(dns.Msg)
		m.SetReply(req)
		m.Answer = []dns.RR{soa}
		for {
			if conn.WriteMsg(m) != nil {
				return
			}
			m.Answer = []dns.RR{a}
			time.Sleep(interval)
		}
	}()
	return ln.Addr().String()
}

func TestAXFRRecordsBudget(t *testing.T) {
	defer func(budget time.Duration) { axfrBudget = budget }(axfrBudget)
	axfrBudget = 200 * time.Millisecond

	start := time.Now()
	rrs, err := (AXFRSource{}).Records(context.Background(), Target{
		Domain:        "example.com.",
		Authoritative: []string{dripServer(t, 20*time.Millisecond)},
	})
	if !errors.Is(err, context.DeadlineExceeded) || rrs != nil {
		t.Errorf("Records = %d records, %v; want %v", len(rrs), err, context.DeadlineExceeded)
	}
	if elapsed := time.Since(start); elapsed > axfrTimeout/2 {
		t.Errorf("the transfer took %s; want it cut at the end of its budget", elapsed)
	}
}

func TestAXFRRecordsCanceledMidTransfer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	time.AfterFunc(100*time.Millisecond, cancel)

	rrs, err := (AXFRSource{}).Records(ctx, Target{
		Domain:        "example.com.",
		Authoritative: []string{dripServer(t, 20*time.Millisecond)},
	})
	if !errors.Is(err, context.Canceled) || rrs != nil {
		t.Errorf("Records = %d records, %v; want %v", len(rrs), err, context.Canceled)
	}
}
