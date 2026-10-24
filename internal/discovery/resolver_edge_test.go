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
	"time"

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
	return Options{Guard: guard, Resolver: addr, authPort: port}
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

func TestResolveTCPFallback(t *testing.T) {
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

	records, _, err := Resolve(context.Background(), "example.com.", []string{"big"}, nil, loopbackOptions(t, addr))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	assertHas(t, recordStrings(records), "big.example.com. 300 IN TXT \"too big for UDP\"")
	if tcp.Load() == 0 {
		t.Error("a truncated answer was not asked again over TCP")
	}
}

func TestResolveServerFailover(t *testing.T) {
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

	records, _, err := Resolve(context.Background(), "example.com.", []string{"www"}, nil, loopbackOptions(t, addr))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	assertHas(t, recordStrings(records), "www.example.com. 300 IN A 192.0.2.1")
	if failed.Load() == 0 {
		t.Error("the failing name server was never asked: the test proves nothing")
	}
}

func TestResolveErrors(t *testing.T) {
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
		"resolver refused by the guard": {
			opts: func(t *testing.T, opts Options) Options {
				opts.Resolver = "10.0.0.53"
				return opts
			},
			want: "unable to use the resolver",
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

			_, _, err := Resolve(context.Background(), "example.com.", nil, nil, opts)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Resolve = %v; want an error about %q", err, tc.want)
			}
		})
	}
}

func TestResolveMixedNameServers(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns0.example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns0.example.com. 300 IN A 10.0.0.53",
		"ns1.example.com. 300 IN A 127.0.0.1",
	)

	records, _, err := Resolve(context.Background(), "example.com.", nil, nil, loopbackOptions(t, serveLocal(t, zone)))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	assertHas(t, recordStrings(records), "example.com. 3600 IN NS ns1.example.com.")
}

func TestResolveNothingAnswered(t *testing.T) {
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

	_, _, err := Resolve(context.Background(), "example.com.", nil, nil, loopbackOptions(t, addr))
	if err == nil || !strings.Contains(err.Error(), "REFUSED") {
		t.Errorf("Resolve = %v; want the refusal of the servers", err)
	}
}

func TestResolveBudgetKeepsApex(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
		"example.com. 3600 IN NS ns1.example.com.",
		"example.com. 300 IN A 192.0.2.1",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"www.example.com. 300 IN A 192.0.2.2",
	)
	opts := loopbackOptions(t, serveLocal(t, zone))
	opts.MaxQueries = len(apexTypes)

	records, _, err := Resolve(context.Background(), "example.com.", []string{"www"}, nil, opts)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	got := recordStrings(records)

	// The apex is asked about first, the budget runs out before the rest.
	assertHas(t, got, "example.com. 300 IN A 192.0.2.1")
	assertHasNoName(t, got, "www.example.com.")
}

func TestResolveMaxDepth(t *testing.T) {
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

	records, _, err := Resolve(context.Background(), "example.com.", []string{"a"}, nil, opts)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
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

func TestResolveServerFailoverDemotes(t *testing.T) {
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
	if _, _, err := Resolve(context.Background(), "example.com.", known, nil, opts); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Once it failed, the others are asked first: each of its failures
	// costs a round trip, or a timeout, to every question.
	if n := failed.Load(); n != 1 {
		t.Errorf("the failing name server was asked %d times; want it asked once, then left last", n)
	}
}

func TestServersByHealthRotates(t *testing.T) {
	e := &explorer{
		servers:  []string{"a", "b", "c"},
		failures: map[string]*atomic.Int32{"a": {}, "b": {}, "c": {}},
	}

	// The questions are spread over the healthy servers, each paced on
	// its own.
	var firsts []string
	for range 3 {
		firsts = append(firsts, e.serversByHealth()[0])
	}
	slices.Sort(firsts)
	if !slices.Equal(firsts, []string{"a", "b", "c"}) {
		t.Errorf("successive questions start with %v; want each server in turn", firsts)
	}

	// A failing server stays last, whatever the turn.
	e.failures["b"].Store(1)
	for range 3 {
		if servers := e.serversByHealth(); servers[2] != "b" {
			t.Errorf("servers = %v; want the failing one last", servers)
		}
	}
}

func TestResolvePacePerServer(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns0.example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns0.example.com. 300 IN A 127.0.0.2",
		"ns1.example.com. 300 IN A 127.0.0.1",
	)
	var asked0, asked1 atomic.Int32
	counting := func(count *atomic.Int32) dns.Handler {
		return handlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
			count.Add(1)
			zone.ServeDNS(w, req)
		})
	}
	addr := serveLocal(t, counting(&asked1), alsoOn{"127.0.0.2", counting(&asked0)})

	opts := loopbackOptions(t, addr)
	opts.QueriesPerSecond = 50
	opts.Concurrency = 4

	var names staticSource
	for i := range 120 {
		names = append(names, fmt.Sprintf("absent%d.example.com.", i))
	}

	start := time.Now()
	if _, _, err := Resolve(context.Background(), "example.com.", nil, []NameSource{names}, opts); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	elapsed := time.Since(start)

	n0, n1 := asked0.Load(), asked1.Load()
	if total := n0 + n1; n0 < total/3 || n1 < total/3 {
		t.Errorf("the name servers were asked %d and %d questions; want them shared", n0, n1)
	}
	// Over 120 questions at 50 a second in all take at least 2.3 seconds:
	// each server asked at its own pace takes about half of it.
	if elapsed > 2*time.Second {
		t.Errorf("exploration took %s; want each server paced on its own", elapsed)
	}
}
