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
	"fmt"
	"net"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/netguard"
)

// alsoOn serves a handler at another loopback IP, on the port of the
// first server.
type alsoOn struct {
	ip string
	h  dns.Handler
}

// serveLocal serves h over UDP and TCP on 127.0.0.1, each of also on the
// same port at its own IP, and returns the address of the first.
func serveLocal(t *testing.T, h dns.Handler, also ...alsoOn) string {
	t.Helper()

	ips := []string{"127.0.0.1"}
	handlers := []dns.Handler{h}
	for _, a := range also {
		ips = append(ips, a.ip)
		handlers = append(handlers, a.h)
	}

	// The port is picked by the first UDP listen, the others bound to it
	// after: a test running in parallel may take it in between.
	var pcs []net.PacketConn
	var lns []net.Listener
	for attempt := 1; ; attempt++ {
		var err error
		pcs, lns, err = listenSamePort(ips)
		if err == nil {
			break
		}
		if attempt == 10 || !errors.Is(err, syscall.EADDRINUSE) {
			t.Fatalf("listen: %v", err)
		}
	}

	for i, h := range handlers {
		for _, srv := range []*dns.Server{{PacketConn: pcs[i], Handler: h}, {Listener: lns[i], Handler: h}} {
			started := make(chan struct{})
			srv.NotifyStartedFunc = func() { close(started) }
			go srv.ActivateAndServe()
			<-started
			t.Cleanup(func() { srv.Shutdown() })
		}
	}

	return pcs[0].LocalAddr().String()
}

// listenSamePort opens a UDP and a TCP listener at each of ips, all on
// one free port, closing them all when one fails.
func listenSamePort(ips []string) (pcs []net.PacketConn, lns []net.Listener, err error) {
	defer func() {
		if err != nil {
			for _, pc := range pcs {
				pc.Close()
			}
			for _, ln := range lns {
				ln.Close()
			}
		}
	}()

	port := "0"
	for _, ip := range ips {
		pc, err := net.ListenPacket("udp", net.JoinHostPort(ip, port))
		if err != nil {
			return pcs, lns, err
		}
		pcs = append(pcs, pc)
		_, port, _ = net.SplitHostPort(pc.LocalAddr().String())

		ln, err := net.Listen("tcp", net.JoinHostPort(ip, port))
		if err != nil {
			return pcs, lns, err
		}
		lns = append(lns, ln)
	}
	return pcs, lns, nil
}

// loopbackOptions points every role of the exploration at addr, the guard
// allowing loopback only.
func loopbackOptions(t *testing.T, addr string) Options {
	t.Helper()

	guard, err := netguard.New("test", "-test-allowed-target", []string{"127.0.0.0/8"})
	if err != nil {
		t.Fatalf("guard: %v", err)
	}

	_, port, _ := net.SplitHostPort(addr)
	return Options{Guard: guard, Resolvers: []string{addr}, authPort: port}
}

// handlerFunc adapts a function to dns.Handler.
type handlerFunc func(w dns.ResponseWriter, req *dns.Msg)

func (f handlerFunc) ServeDNS(w dns.ResponseWriter, req *dns.Msg) { f(w, req) }

// answerWith replies to every question with rcode, counting them.
func answerWith(rcode int, count *atomic.Int32) dns.Handler {
	return handlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		count.Add(1)
		m := new(dns.Msg)
		m.SetRcode(req, rcode)
		w.WriteMsg(m)
	})
}

func TestExploreTCPFallback(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"big.example.com. 300 IN TXT \"too big for UDP\"",
	)
	var tcp atomic.Int32
	addr := serveLocal(t, handlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		if req.Question[0].Name != "big.example.com." {
			zone.ServeDNS(w, req)
			return
		}
		if w.RemoteAddr().Network() == "udp" {
			m := new(dns.Msg)
			m.SetReply(req)
			m.Truncated = true
			w.WriteMsg(m)
			return
		}
		tcp.Add(1)
		zone.ServeDNS(w, req)
	}))

	records, err := Explore(context.Background(), "example.com.", []string{"big"}, nil, loopbackOptions(t, addr))
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}

	assertHas(t, recordStrings(records), "big.example.com. 300 IN TXT \"too big for UDP\"")
	if tcp.Load() == 0 {
		t.Error("a truncated answer was not asked again over TCP")
	}
}

func TestExploreServerFailover(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns0.example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns0.example.com. 300 IN A 127.0.0.2",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"www.example.com. 300 IN A 192.0.2.1",
	)
	// The first name server fails every question.
	var failed atomic.Int32
	addr := serveLocal(t, zone, alsoOn{"127.0.0.2", answerWith(dns.RcodeServerFailure, &failed)})

	records, err := Explore(context.Background(), "example.com.", []string{"www"}, nil, loopbackOptions(t, addr))
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}

	assertHas(t, recordStrings(records), "www.example.com. 300 IN A 192.0.2.1")
	if failed.Load() == 0 {
		t.Error("the failing name server was never asked: the test proves nothing")
	}
}

func TestExploreLameServer(t *testing.T) {
	upward, _ := dns.NewRR("com. 172800 IN NS a.gtld-servers.net.")
	for _, tt := range []struct {
		name   string
		answer func(m *dns.Msg)
	}{
		{"upward referral", func(m *dns.Msg) { m.Ns = []dns.RR{upward} }},
		{"empty answer", func(m *dns.Msg) {}},
		{"name error", func(m *dns.Msg) { m.Rcode = dns.RcodeNameError }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			zone := newTestZone(t, "example.com.",
				"example.com. 3600 IN NS ns0.example.com.",
				"example.com. 3600 IN NS ns1.example.com.",
				"ns0.example.com. 300 IN A 127.0.0.2",
				"ns1.example.com. 300 IN A 127.0.0.1",
			)
			var known []string
			for i := range 6 {
				name := fmt.Sprintf("host%d", i)
				known = append(known, name)
				zone.rrs = append(zone.rrs, &dns.A{Hdr: dns.RR_Header{Name: name + ".example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300}, A: net.IPv4(192, 0, 2, byte(i))})
			}
			// The first name server no longer serves the zone, and
			// answers without authority.
			var asked atomic.Int32
			addr := serveLocal(t, zone, alsoOn{"127.0.0.2", handlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
				asked.Add(1)
				m := new(dns.Msg)
				m.SetReply(req)
				tt.answer(m)
				w.WriteMsg(m)
			})})

			records, err := Explore(context.Background(), "example.com.", known, nil, loopbackOptions(t, addr))
			if err != nil {
				t.Fatalf("Explore: %v", err)
			}
			if asked.Load() == 0 {
				t.Fatal("the lame name server was never asked: the test proves nothing")
			}
			got := recordStrings(records)
			for i, name := range known {
				assertHas(t, got, fmt.Sprintf("%s.example.com. 300 IN A 192.0.2.%d", name, i))
			}
		})
	}
}

func TestExploreErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		records []string
		opts    func(t *testing.T, opts Options) Options
		want    string
	}{
		"no name server": {
			records: []string{"example.com. 300 IN A 192.0.2.1"},
			want:    "no name server",
		},
		"name servers all refused by the guard": {
			records: []string{
				"example.com. 3600 IN NS ns1.example.com.",
				"example.com. 3600 IN NS ns2.example.com.",
				"ns1.example.com. 300 IN A 10.0.0.53",
				"ns2.example.com. 300 IN AAAA ::ffff:10.0.0.53",
			},
			want: "can be reached",
		},
		"canceled": {
			records: []string{
				"example.com. 3600 IN NS ns1.example.com.",
				"ns1.example.com. 300 IN A 127.0.0.1",
			},
			opts: func(t *testing.T, opts Options) Options {
				opts.Timeout = 1
				return opts
			},
			want: "name servers",
		},
	} {
		t.Run(name, func(t *testing.T) {
			zone := newTestZone(t, "example.com.", tc.records...)
			opts := loopbackOptions(t, serveLocal(t, zone))
			if tc.opts != nil {
				opts = tc.opts(t, opts)
			}

			_, err := Explore(context.Background(), "example.com.", nil, nil, opts)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Explore = %v; want an error about %q", err, tc.want)
			}
		})
	}
}

func TestExploreCanceledDuringExploration(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"a.example.com. 300 IN A 192.0.2.1",
		"b.example.com. 300 IN A 192.0.2.2",
		"c.example.com. 300 IN A 192.0.2.3",
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var explored atomic.Int32
	opts := loopbackOptions(t, serveLocal(t, handlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		if strings.Count(req.Question[0].Name, ".") == 3 && req.Question[0].Name != "ns1.example.com." {
			explored.Add(1)
			cancel()
		}
		zone.ServeDNS(w, req)
	})))
	// One name at a time: the others wait for their turn when canceled.
	opts.Concurrency = 1

	// The exploration cut short is no error: its records are kept.
	if _, err := Explore(ctx, "example.com.", []string{"a", "b", "c"}, nil, opts); err != nil {
		t.Fatalf("Explore: %v", err)
	}
	if n := explored.Load(); n != 1 {
		t.Errorf("%d names were asked about; want the exploration to stop after the first", n)
	}
}

func TestExploreMixedNameServers(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns0.example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns0.example.com. 300 IN A 10.0.0.53",
		"ns1.example.com. 300 IN A 127.0.0.1",
	)

	records, err := Explore(context.Background(), "example.com.", nil, nil, loopbackOptions(t, serveLocal(t, zone)))
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	assertHas(t, recordStrings(records), "example.com. 3600 IN NS ns1.example.com.")
}

func TestExploreNothingAnswered(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
	)
	// The server answers as a resolver, but refuses as an authority.
	addr := serveLocal(t, handlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		if req.RecursionDesired {
			zone.ServeDNS(w, req)
			return
		}
		m := new(dns.Msg)
		m.SetRcode(req, dns.RcodeRefused)
		w.WriteMsg(m)
	}))

	_, err := Explore(context.Background(), "example.com.", nil, nil, loopbackOptions(t, addr))
	if err == nil || !strings.Contains(err.Error(), "REFUSED") {
		t.Errorf("Explore = %v; want the refusal of the servers", err)
	}
}

func TestExploreBudgetKeepsApex(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
		"example.com. 3600 IN NS ns1.example.com.",
		"example.com. 300 IN A 192.0.2.1",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"www.example.com. 300 IN A 192.0.2.2",
	)
	opts := loopbackOptions(t, serveLocal(t, zone))
	opts.MaxQueries = len(apexTypes)

	records, err := Explore(context.Background(), "example.com.", []string{"www"}, nil, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	got := recordStrings(records)

	// The apex is asked about first, the budget runs out before the rest.
	assertHas(t, got, "example.com. 300 IN A 192.0.2.1")
	assertHasNoName(t, got, "www.example.com.")
}

func TestExploreMaxDepth(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"a.example.com. 300 IN CNAME b.example.com.",
		"b.example.com. 300 IN CNAME c.example.com.",
		"c.example.com. 300 IN CNAME d.example.com.",
		"d.example.com. 300 IN A 192.0.2.1",
	)
	opts := loopbackOptions(t, serveLocal(t, zone))
	opts.MaxDepth = 1

	records, err := Explore(context.Background(), "example.com.", []string{"a"}, nil, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	got := recordStrings(records)

	assertHas(t, got,
		"a.example.com. 300 IN CNAME b.example.com.",
		"b.example.com. 300 IN CNAME c.example.com.",
	)
	assertHasNoName(t, got, "c.example.com.", "d.example.com.")
}

func TestUnseen(t *testing.T) {
	e := &explorer{domain: "example.com.", seen: map[string]bool{"old.example.com.": true}}

	got := e.unseen([]Name{
		{FQDN: "WWW.example.com.", Sources: []string{"a"}},
		{FQDN: "www.example.com.", Sources: []string{"b"}},
		{FQDN: "old.example.com.", Sources: []string{"a"}},
		{FQDN: "www.other.test.", Sources: []string{"a"}},
		{FQDN: "notexample.com.", Sources: []string{"a"}},
	})

	if len(got) != 1 || got[0].FQDN != "www.example.com." || !slices.Equal(got[0].Sources, []string{"a", "b"}) {
		t.Errorf("unseen = %+v; want www once, from both sources", got)
	}
	if again := e.unseen([]Name{{FQDN: "www.example.com."}}); len(again) != 0 {
		t.Errorf("unseen yielded %+v twice", again)
	}
}

func TestTypesFor(t *testing.T) {
	e := &explorer{domain: "example.com."}

	for name, want := range map[string][]uint16{
		"example.com.":             apexTypes,
		"www.example.com.":         hostTypes,
		"_dmarc.example.com.":      underscoreTypes,
		"_x._tcp.www.example.com.": underscoreTypes,
	} {
		if got := e.typesFor(name); !slices.Equal(got, want) {
			t.Errorf("typesFor(%q) = %v; want %v", name, got, want)
		}
	}
}

// failingSource yields its names along with an error.
type failingSource []string

func (failingSource) ID() string { return "failing" }

func (s failingSource) Discover(ctx context.Context, t Target) ([]Name, error) {
	names, _ := staticSource(s).Discover(ctx, t)
	return names, errors.New("partly unavailable")
}

func TestExploreSourceError(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"www.example.com. 300 IN A 192.0.2.1",
	)
	opts := zone.serve(t)

	records, err := Explore(context.Background(), "example.com.", nil, []NameSource{failingSource{"www.example.com."}}, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}

	// The names found before the error are not lost.
	assertHas(t, recordStrings(records), "www.example.com. 300 IN A 192.0.2.1")
}

func TestExploreWildcardCNAME(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"*.example.com. 300 IN CNAME target.example.com.",
		"target.example.com. 300 IN A 192.0.2.99",
		"real.example.com. 300 IN A 192.0.2.1",
	)
	opts := zone.serve(t)

	records, err := Explore(context.Background(), "example.com.", []string{"real", "ghost"}, nil, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	got := recordStrings(records)

	assertHas(t, got,
		"*.example.com. 300 IN CNAME target.example.com.",
		"target.example.com. 300 IN A 192.0.2.99",
		"real.example.com. 300 IN A 192.0.2.1",
	)
	assertHasNoName(t, got, "ghost.example.com.")
}

func TestExploreDNAME(t *testing.T) {
	for _, tt := range []struct {
		name    string
		records []string
		known   []string
		want    []string
		absent  []string
	}{
		{
			name:    "at the apex",
			records: []string{"example.com. 3600 IN DNAME other.test."},
			known:   []string{"a", "b"},
			want:    []string{"example.com. 3600 IN DNAME other.test."},
			absent:  []string{"a.example.com.", "b.example.com.", "*.example.com."},
		},
		{
			name: "below the apex",
			records: []string{
				"old.example.com. 3600 IN DNAME example.net.",
				"www.example.com. 300 IN A 192.0.2.1",
			},
			known:  []string{"x.old", "www"},
			want:   []string{"old.example.com. 3600 IN DNAME example.net.", "www.example.com. 300 IN A 192.0.2.1"},
			absent: []string{"x.old.example.com."},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			zone := newTestZone(t, "example.com.", append([]string{
				"example.com. 3600 IN SOA ns.other.test. admin.example.com. 1 7200 3600 1209600 300",
				"example.com. 3600 IN NS ns.other.test.",
			}, tt.records...)...)
			// Below a DNAME, the server makes up an alias for any name.
			opts := loopbackOptions(t, serveLocal(t, handlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
				name := dns.CanonicalName(req.Question[0].Name)
				m := new(dns.Msg)
				m.SetReply(req)
				m.Authoritative = true
				if name == "ns.other.test." {
					if req.Question[0].Qtype == dns.TypeA {
						m.Answer = []dns.RR{&dns.A{Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300}, A: net.IPv4(127, 0, 0, 1)}}
					}
					w.WriteMsg(m)
					return
				}
				for _, rr := range zone.rrs {
					if dname, ok := rr.(*dns.DNAME); ok && name != dname.Hdr.Name && dns.IsSubDomain(dname.Hdr.Name, name) {
						alias := &dns.CNAME{Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: dname.Hdr.Ttl}, Target: strings.TrimSuffix(name, dname.Hdr.Name) + dname.Target}
						m.Answer = []dns.RR{dname, alias}
						w.WriteMsg(m)
						return
					}
				}
				zone.ServeDNS(w, req)
			})))

			records, err := Explore(context.Background(), "example.com.", tt.known, nil, opts)
			if err != nil {
				t.Fatalf("Explore: %v", err)
			}
			got := recordStrings(records)
			assertHas(t, got, tt.want...)
			assertHasNoName(t, got, tt.absent...)
		})
	}
}

func TestAddRecordSkipsDNSSEC(t *testing.T) {
	e := &explorer{domain: "example.com.", records: map[string]dns.RR{}}

	for _, s := range []string{
		"WWW.example.com. 300 IN A 192.0.2.1",
		"www.example.com. 300 IN RRSIG A 13 3 300 20261101000000 20261001000000 12345 example.com. dGVzdA==",
		"www.example.com. 300 IN NSEC zzz.example.com. A RRSIG NSEC",
	} {
		rr, err := dns.NewRR(s)
		if err != nil {
			t.Fatal(err)
		}
		e.addRecord(rr)
	}

	if got := recordStrings(e.result()); !slices.Equal(got, []string{"www.example.com. 300 IN A 192.0.2.1"}) {
		t.Errorf("records = %v; want the A record only, its owner lowercased", got)
	}
}

func TestKnownSource(t *testing.T) {
	names, err := KnownSource{}.Discover(context.Background(), Target{
		Domain: "example.com.",
		Known:  []string{"www", "@", "", "Mail.Example.com.", "a.b"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, name := range names {
		got = append(got, name.FQDN)
	}
	want := []string{"www.example.com.", "example.com.", "example.com.", "mail.example.com.", "a.b.example.com."}
	if !slices.Equal(got, want) {
		t.Errorf("names = %v; want %v", got, want)
	}
}

func TestExploreServerFailoverDemotes(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns0.example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns0.example.com. 300 IN A 127.0.0.2",
		"ns1.example.com. 300 IN A 127.0.0.1",
	)
	var known []string
	for i := range 20 {
		name := fmt.Sprintf("host%d", i)
		known = append(known, name)
		zone.rrs = append(zone.rrs, &dns.A{Hdr: dns.RR_Header{Name: name + ".example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300}, A: net.IPv4(192, 0, 2, byte(i))})
	}
	// The first name server refuses every question, as one blocked from
	// the network of the instance does.
	var failed atomic.Int32
	addr := serveLocal(t, zone, alsoOn{"127.0.0.2", answerWith(dns.RcodeRefused, &failed)})

	opts := loopbackOptions(t, addr)
	opts.Concurrency = 1
	if _, err := Explore(context.Background(), "example.com.", known, nil, opts); err != nil {
		t.Fatalf("Explore: %v", err)
	}

	// Once it failed, the others are asked first: each of its failures
	// costs a round trip, or a timeout, to every question.
	if n := failed.Load(); n != 1 {
		t.Errorf("the failing name server was asked %d times; want it asked once, then left last", n)
	}
}
