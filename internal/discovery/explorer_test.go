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
	"math"
	"net"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/time/rate"

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

// testZone is a minimal authoritative server for one zone, also standing in
// for the recursive resolver: it answers the same either way.
type testZone struct {
	origin string
	rrs    []dns.RR
	// cuts are the delegation points of the zone.
	cuts []string
	// allowAXFR lets anyone transfer the zone.
	allowAXFR bool
	// servfail are the names failing every question, but for their NSEC.
	servfail map[string]bool

	// questions counts the questions answered.
	questions atomic.Int32

	// dnssec tunes how the zone denies existence, see source_nsec_test.go.
	dnssec testDNSSEC

	mu sync.Mutex
	// asked are the names questions were asked about.
	asked map[string]bool
}

func newTestZone(t *testing.T, origin string, records ...string) *testZone {
	t.Helper()

	z := &testZone{origin: origin}
	for _, s := range records {
		rr, err := dns.NewRR(s)
		if err != nil {
			t.Fatalf("bad test record %q: %v", s, err)
		}
		rr.Header().Name = dns.CanonicalName(rr.Header().Name)
		z.rrs = append(z.rrs, rr)

		if rr.Header().Rrtype == dns.TypeNS && rr.Header().Name != origin {
			z.cuts = append(z.cuts, rr.Header().Name)
		}
	}
	return z
}

func (z *testZone) at(name string) (ret []dns.RR) {
	for _, rr := range z.rrs {
		if rr.Header().Name == name {
			ret = append(ret, rr)
		}
	}
	return
}

func (z *testZone) ServeDNS(w dns.ResponseWriter, req *dns.Msg) {
	z.questions.Add(1)

	m := new(dns.Msg)
	m.SetReply(req)
	defer w.WriteMsg(m)

	name := dns.CanonicalName(req.Question[0].Name)
	qtype := req.Question[0].Qtype

	if !dns.IsSubDomain(z.origin, name) {
		m.Rcode = dns.RcodeRefused
		return
	}

	z.mu.Lock()
	if z.asked == nil {
		z.asked = map[string]bool{}
	}
	z.asked[name] = true
	z.mu.Unlock()

	// Deferred after the deferred w.WriteMsg, so it runs before it and the
	// reply carries the denial proofs.
	defer z.proveDenial(m, name)
	if z.answerNSEC(m, name, qtype) {
		return
	}

	if z.servfail[name] {
		m.Rcode = dns.RcodeServerFailure
		return
	}

	if qtype == dns.TypeAXFR {
		if !z.allowAXFR || name != z.origin {
			m.Rcode = dns.RcodeRefused
			return
		}

		soa := z.at(z.origin)[0]
		m.Answer = append(m.Answer, soa)
		for _, rr := range z.rrs {
			if rr != soa {
				m.Answer = append(m.Answer, rr)
			}
		}
		m.Answer = append(m.Answer, soa)
		return
	}

	for _, cut := range z.cuts {
		if dns.IsSubDomain(cut, name) {
			m.Ns = z.at(cut)
			for _, rr := range m.Ns {
				if ns, ok := rr.(*dns.NS); ok {
					m.Extra = append(m.Extra, z.at(ns.Ns)...)
				}
			}
			// The referral to an unsigned child proves there is no DS.
			if z.dnssec.nsec && !slices.ContainsFunc(m.Ns, func(rr dns.RR) bool { return rr.Header().Rrtype == dns.TypeDS }) {
				m.Ns = append(m.Ns, z.nsecFor(cut))
			}
			return
		}
	}

	m.Authoritative = true

	rrs := z.at(name)
	if len(rrs) == 0 {
		hasChild := slices.ContainsFunc(z.rrs, func(rr dns.RR) bool { return dns.IsSubDomain(name, rr.Header().Name) })
		wildcard := z.at("*." + z.origin)
		switch {
		case hasChild:
			return
		case len(wildcard) > 0:
			for _, rr := range wildcard {
				rr = dns.Copy(rr)
				rr.Header().Name = name
				rrs = append(rrs, rr)
			}
		default:
			m.Rcode = dns.RcodeNameError
			return
		}
	}

	for _, rr := range rrs {
		if rr.Header().Rrtype == dns.TypeCNAME && qtype != dns.TypeCNAME {
			m.Answer = []dns.RR{rr}
			return
		}
		if rr.Header().Rrtype == qtype {
			m.Answer = append(m.Answer, rr)
		}
	}
}

// serve starts z on a local port, over UDP and TCP, and returns options
// pointing every role of the exploration at it.
func (z *testZone) serve(t *testing.T) Options {
	t.Helper()

	addr := serveLocal(t, z)

	guard, err := netguard.New("test", "-test-allowed-target", []string{"127.0.0.0/8"})
	if err != nil {
		t.Fatalf("guard: %v", err)
	}

	_, port, _ := net.SplitHostPort(addr)
	return Options{
		Guard:     guard,
		Resolvers: []string{addr},
		// The local server has no rate to protect.
		QueriesPerSecond: -1,
		authPort:         port,
	}
}

// staticSource yields fixed names.
type staticSource []string

func (staticSource) ID() string { return "static" }

func (s staticSource) Discover(ctx context.Context, t Target) (names []Name, err error) {
	for _, name := range s {
		names = append(names, Name{FQDN: name, Sources: []string{"static"}})
	}
	return
}

// wasAsked tells whether a question was asked about name.
func (z *testZone) wasAsked(name string) bool {
	z.mu.Lock()
	defer z.mu.Unlock()
	return z.asked[name]
}

func recordStrings(records []happydns.Record) (ret []string) {
	for _, rr := range records {
		ret = append(ret, strings.Join(strings.Fields(rr.String()), " "))
	}
	return
}

func assertHas(t *testing.T, got []string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("missing record %q in:\n%s", w, strings.Join(got, "\n"))
		}
	}
}

func assertHasNoName(t *testing.T, got []string, names ...string) {
	t.Helper()
	for _, rr := range got {
		for _, name := range names {
			if strings.HasPrefix(rr, name+" ") {
				t.Errorf("unexpected record %q", rr)
			}
		}
	}
}

func TestExplore(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
		"example.com. 3600 IN NS ns1.example.com.",
		"example.com. 300 IN A 192.0.2.1",
		"example.com. 300 IN MX 10 mail.example.com.",
		"example.com. 300 IN TXT \"v=spf1 mx -all\"",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"mail.example.com. 300 IN A 192.0.2.10",
		"www.example.com. 300 IN CNAME web.example.com.",
		"web.example.com. 300 IN A 192.0.2.20",
		"_dmarc.example.com. 300 IN TXT \"v=DMARC1; p=reject\"",
		"hidden.example.com. 300 IN A 192.0.2.30",
		"sub.example.com. 300 IN NS ns.other.test.",
		"host.sub.example.com. 300 IN A 192.0.2.40",
	)
	opts := zone.serve(t)

	records, _, err := Explore(context.Background(), "Example.com", []string{"_dmarc", "host.sub"}, []NameSource{staticSource{"www.example.com."}}, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	got := recordStrings(records)

	assertHas(t, got,
		// The apex.
		"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
		"example.com. 3600 IN NS ns1.example.com.",
		"example.com. 300 IN A 192.0.2.1",
		"example.com. 300 IN MX 10 mail.example.com.",
		"example.com. 300 IN TXT \"v=spf1 mx -all\"",
		// Targets of the apex records.
		"ns1.example.com. 300 IN A 127.0.0.1",
		"mail.example.com. 300 IN A 192.0.2.10",
		// A known name.
		"_dmarc.example.com. 300 IN TXT \"v=DMARC1; p=reject\"",
		// A name from a source, an alias, and its target.
		"www.example.com. 300 IN CNAME web.example.com.",
		"web.example.com. 300 IN A 192.0.2.20",
		// The delegation a known name lies below.
		"sub.example.com. 300 IN NS ns.other.test.",
	)

	// Nothing points to hidden, and the child zone is not ours.
	assertHasNoName(t, got, "hidden.example.com.", "host.sub.example.com.")
}

func TestExploreDelegationGlue(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"sub.example.com. 300 IN NS ns.sub.example.com.",
		"sub.example.com. 300 IN NS ns.elsewhere.example.com.",
		"ns.sub.example.com. 300 IN A 192.0.2.53",
		"ns.sub.example.com. 300 IN AAAA 2001:db8::53",
		"ns.sub.example.com. 300 IN TXT \"not glue\"",
		"ns.elsewhere.example.com. 300 IN A 192.0.2.54",
	)
	// The referrals carry a stale address of the name server out of the
	// delegation, to tell whether it was taken from them.
	var askedGlue atomic.Bool
	opts := loopbackOptions(t, serveLocal(t, handlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		if dns.CanonicalName(req.Question[0].Name) == "ns.sub.example.com." {
			askedGlue.Store(true)
		}
		zone.ServeDNS(rewritingWriter{w, func(m *dns.Msg) {
			for i, rr := range m.Extra {
				if a, ok := rr.(*dns.A); ok && a.Hdr.Name == "ns.elsewhere.example.com." {
					stale := *a
					stale.A = net.IPv4(198, 51, 100, 54)
					m.Extra[i] = &stale
				}
			}
		}}, req)
	})))
	opts.QueriesPerSecond = -1

	records, _, err := Explore(context.Background(), "example.com.", []string{"sub"}, nil, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	got := recordStrings(records)

	assertHas(t, got,
		"sub.example.com. 300 IN NS ns.sub.example.com.",
		"ns.sub.example.com. 300 IN A 192.0.2.53",
		"ns.sub.example.com. 300 IN AAAA 2001:db8::53",
	)
	if slices.Contains(got, "ns.sub.example.com. 300 IN TXT \"not glue\"") {
		t.Error("Explore kept a record of the child zone that is no glue")
	}
	// The glue is only known from the referral: asking about it would
	// only get the referral again.
	if askedGlue.Load() {
		t.Error("Explore asked about a name server below the delegation")
	}

	// The name server out of the delegation is no glue but a name of the
	// zone: its records come from asking about it, not from the referral.
	assertHas(t, got, "ns.elsewhere.example.com. 300 IN A 192.0.2.54")
	if slices.Contains(got, "ns.elsewhere.example.com. 300 IN A 198.51.100.54") {
		t.Error("Explore kept the address of a name server out of the delegation from the referral")
	}
}

func TestExploreChildOnSameServer(t *testing.T) {
	parent := newTestZone(t, "example.com.",
		"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"sub.example.com. 300 IN NS ns1.example.com.",
	)
	child := newTestZone(t, "sub.example.com.",
		"sub.example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
		"sub.example.com. 300 IN NS ns1.example.com.",
		"sub.example.com. 300 IN A 192.0.2.77",
		"www.sub.example.com. 300 IN A 192.0.2.78",
	)
	// The name server serves both zones: it answers about the child with
	// the authority of the child, its denials carrying the child's SOA.
	opts := loopbackOptions(t, serveLocal(t, handlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		if !dns.IsSubDomain(child.origin, dns.CanonicalName(req.Question[0].Name)) {
			parent.ServeDNS(w, req)
			return
		}
		child.ServeDNS(rewritingWriter{w, func(m *dns.Msg) {
			if m.Authoritative && len(m.Answer) == 0 {
				m.Ns = append(m.Ns, child.at(child.origin)[0])
			}
		}}, req)
	})))

	records, _, err := Explore(context.Background(), "example.com.", []string{"sub", "www.sub"}, nil, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	got := recordStrings(records)

	assertHas(t, got, "sub.example.com. 300 IN NS ns1.example.com.")
	if slices.Contains(got, "sub.example.com. 300 IN A 192.0.2.77") {
		t.Error("Explore kept a record of the child zone at the delegation")
	}
	assertHasNoName(t, got, "www.sub.example.com.")
}

// rewritingWriter rewrites the replies with f before sending them.
type rewritingWriter struct {
	dns.ResponseWriter
	f func(*dns.Msg)
}

func (w rewritingWriter) WriteMsg(m *dns.Msg) error {
	w.f(m)
	return w.ResponseWriter.WriteMsg(m)
}

func TestExploreWildcard(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"*.example.com. 300 IN A 192.0.2.99",
		"real.example.com. 300 IN A 192.0.2.1",
	)
	opts := zone.serve(t)

	records, _, err := Explore(context.Background(), "example.com.", []string{"real", "ghost"}, nil, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	got := recordStrings(records)

	assertHas(t, got,
		"*.example.com. 300 IN A 192.0.2.99",
		"real.example.com. 300 IN A 192.0.2.1",
	)
	assertHasNoName(t, got, "ghost.example.com.")
}

func TestExploreWildcardOwnAnswers(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"*.example.com. 300 IN TXT \"wildcard\"",
		"real.example.com. 300 IN A 192.0.2.1",
		"real.example.com. 300 IN TXT \"wildcard\"",
	)
	opts := zone.serve(t)

	records, _, err := Explore(context.Background(), "example.com.", []string{"real"}, nil, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}

	// Once the name is known to exist, a record like the wildcard is its own.
	assertHas(t, recordStrings(records), "real.example.com. 300 IN TXT \"wildcard\"")
}

func TestExploreWildcardCost(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"*.example.com. 300 IN A 192.0.2.99",
	)
	opts := zone.serve(t)

	if _, _, err := Explore(context.Background(), "example.com.", nil, nil, opts); err != nil {
		t.Fatalf("Explore: %v", err)
	}
	base := zone.questions.Swap(0)

	ghosts := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	if _, _, err := Explore(context.Background(), "example.com.", ghosts, nil, opts); err != nil {
		t.Fatalf("Explore: %v", err)
	}
	if cost := zone.questions.Load() - base; cost != int32(len(ghosts)) {
		t.Errorf("%d names the wildcard makes up cost %d questions; want one each", len(ghosts), cost)
	}
}

func TestExploreWildcardProbeLost(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"*.example.com. 300 IN A 192.0.2.99",
	)
	// The first probe of the wildcard is lost on the way.
	var lost atomic.Bool
	opts := loopbackOptions(t, serveLocal(t, handlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		if strings.HasPrefix(req.Question[0].Name, "hd-probe-") && lost.CompareAndSwap(false, true) {
			return
		}
		zone.ServeDNS(w, req)
	})))
	opts.QueryTimeout = 200 * time.Millisecond

	records, _, err := Explore(context.Background(), "example.com.", []string{"ghost"}, nil, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	if !lost.Load() {
		t.Fatal("no probe was lost: the test proves nothing")
	}
	assertHasNoName(t, recordStrings(records), "ghost.example.com.")
}

func TestExploreWildcardSigned(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"*.example.com. 300 IN A 192.0.2.99",
		"real.example.com. 300 IN A 192.0.2.1",
	)
	// The zone is signed: each answer carries its signature, the same for
	// every name the wildcard makes up.
	opts := loopbackOptions(t, serveLocal(t, handlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		zone.ServeDNS(rewritingWriter{w, func(m *dns.Msg) {
			if len(m.Answer) == 0 {
				return
			}
			hdr := m.Answer[0].Header()
			signer := hdr.Name
			if len(zone.at(signer)) == 0 {
				signer = "*.example.com."
			}
			m.Answer = append(m.Answer, &dns.RRSIG{
				Hdr:         dns.RR_Header{Name: hdr.Name, Rrtype: dns.TypeRRSIG, Class: dns.ClassINET, Ttl: hdr.Ttl},
				TypeCovered: hdr.Rrtype,
				Algorithm:   dns.ECDSAP256SHA256,
				Labels:      uint8(dns.CountLabel(strings.TrimPrefix(signer, "*."))),
				OrigTtl:     hdr.Ttl,
				Expiration:  2000000000,
				Inception:   1700000000,
				KeyTag:      12345,
				SignerName:  "example.com.",
				Signature:   "c2lnbmF0dXJl",
			})
		}}, req)
	})))

	records, _, err := Explore(context.Background(), "example.com.", []string{"real", "ghost"}, nil, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	got := recordStrings(records)

	assertHas(t, got,
		"*.example.com. 300 IN A 192.0.2.99",
		"real.example.com. 300 IN A 192.0.2.1",
	)
	assertHasNoName(t, got, "ghost.example.com.")
}

func TestExploreBudget(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
	)
	opts := zone.serve(t)
	opts.MaxQueries = 1

	records, _, err := Explore(context.Background(), "example.com.", nil, nil, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	if got := recordStrings(records); len(got) != 1 || !strings.Contains(got[0], "SOA") {
		t.Errorf("Explore with a budget of one question = %v; want the apex SOA only", got)
	}
}

func TestExploreRefusesPrivateNameServers(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 10.0.0.53",
	)
	// The guard allows the resolver, on loopback, but not the private
	// address the name server resolves to.
	opts := zone.serve(t)

	_, _, err := Explore(context.Background(), "example.com.", nil, nil, opts)
	if err == nil {
		t.Fatal("Explore reached a name server on loopback without the guard allowing it")
	}
}

func TestLocate(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"www.example.com. 300 IN A 192.0.2.20",
	)
	opts := zone.serve(t)

	servers, err := Locate(context.Background(), "Example.com", opts)
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if len(servers) != 1 || servers[0] != net.JoinHostPort("127.0.0.1", opts.authPort) {
		t.Errorf("servers = %v; want the one name server", servers)
	}

	// Finding the name servers is all it takes to tell the domain exists:
	// their names, then the addresses of the one name server.
	if n := zone.questions.Load(); n != 3 {
		t.Errorf("Locate asked %d questions; want 3, not exploring the zone", n)
	}
}

func TestLocateNoNameServer(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
	)
	opts := zone.serve(t)

	if _, err := Locate(context.Background(), "example.com.", opts); err == nil {
		t.Error("Locate found name servers where there are none")
	}
}

func TestExplorePace(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
	)
	opts := zone.serve(t)
	opts.QueriesPerSecond = 50
	opts.Concurrency = 4

	var names staticSource
	for i := range 60 {
		names = append(names, fmt.Sprintf("absent%d.example.com.", i))
	}

	start := time.Now()
	if _, _, err := Explore(context.Background(), "example.com.", nil, []NameSource{names}, opts); err != nil {
		t.Fatalf("Explore: %v", err)
	}

	// 60 names and the apex are more than 60 questions: past the burst
	// of 4, they take at least (60-4)/50 seconds.
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Errorf("exploration took %s; want the questions paced at 50 per second", elapsed)
	}
}

func TestQueriesPerSecondDefault(t *testing.T) {
	if got := (Options{}).withDefaults().QueriesPerSecond; got != DefaultQueriesPerSecond {
		t.Errorf("QueriesPerSecond defaults to %d; want %d", got, DefaultQueriesPerSecond)
	}
	if got := (Options{QueriesPerSecond: -1}).withDefaults().QueriesPerSecond; got != -1 {
		t.Errorf("QueriesPerSecond -1 becomes %d; want the cap lifted", got)
	}
}

func TestQueryOutOfTime(t *testing.T) {
	var asked atomic.Int32
	addr := serveLocal(t, answerWith(dns.RcodeSuccess, &asked))

	opts := loopbackOptions(t, addr).withDefaults()
	e := &explorer{
		domain:   "example.com.",
		opts:     opts,
		udp:      &dns.Client{Timeout: opts.QueryTimeout},
		servers:  []string{addr},
		failures: map[string]*atomic.Int32{addr: new(atomic.Int32)},
		// Its only question already asked, the next one is a second away.
		pace: map[string]*rate.Limiter{addr: rate.NewLimiter(1, 1)},
	}
	e.pace[addr].Allow()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := e.query(ctx, "www.example.com.", dns.TypeA); !errors.Is(err, errOutOfTime) {
		t.Errorf("query = %v; want %v", err, errOutOfTime)
	}
	if asked.Load() != 0 {
		t.Error("the server was asked a question the pace left no time for")
	}
	// A question the pace leaves no time for is not a server failing.
	if n := e.failures[addr].Load(); n != 0 {
		t.Errorf("the server failed %d questions; want none", n)
	}
}

// progressSource yields names, recording how far the import was when
// asked for them.
type progressSource struct {
	names []string
	seen  *happydns.ImportProgressSnapshot
}

func (progressSource) ID() string { return "progress" }

func (s progressSource) Discover(ctx context.Context, t Target) (names []Name, err error) {
	*s.seen = happydns.ImportProgressFrom(ctx).Snapshot()
	return staticSource(s.names).Discover(ctx, t)
}

func TestExploreReportsProgress(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"example.com. 300 IN A 192.0.2.1",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"www.example.com. 300 IN CNAME web.example.com.",
		"web.example.com. 300 IN A 192.0.2.20",
	)
	opts := zone.serve(t)

	progress := happydns.NewImportProgress()
	ctx := happydns.WithImportProgress(context.Background(), progress)

	var seen happydns.ImportProgressSnapshot
	records, _, err := Explore(ctx, "example.com.", nil, []NameSource{progressSource{[]string{"www.example.com.", "absent.example.com."}, &seen}}, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}

	if seen.Step != happydns.ImportStepCollecting || seen.Source != "progress" {
		t.Errorf("the source was asked at step %q, source %q; want %q, \"progress\"", seen.Step, seen.Source, happydns.ImportStepCollecting)
	}
	if seen.Done == 0 {
		t.Error("the apex was not counted as explored before the sources were asked")
	}
	bands := planBands(false)
	if float64(seen.Percent) < bands.collecting.from || float64(seen.Percent) > bands.collecting.to {
		t.Errorf("the source was asked at %d%%; want within the collection, from %v%% to %v%%", seen.Percent, bands.collecting.from, bands.collecting.to)
	}

	s := progress.Snapshot()
	if s.Step != happydns.ImportStepFollowing {
		t.Errorf("the exploration ended at step %q; want %q, as www has a target to follow", s.Step, happydns.ImportStepFollowing)
	}
	if s.Total == 0 || s.Done != s.Total {
		t.Errorf("the exploration ended at %d names of %d; want all of them, and some", s.Done, s.Total)
	}
	if float64(s.Percent) < bands.exploring.to || s.Percent > happydns.ImportRetrievedPercent {
		t.Errorf("the exploration ended at %d%%; want past the exploration, at %v%%, and up to %d%%", s.Percent, bands.exploring.to, happydns.ImportRetrievedPercent)
	}
	if s.Records != int64(len(records)) {
		t.Errorf("the progress counts %d records; want the %d found", s.Records, len(records))
	}
	if s.Queries == 0 {
		t.Error("the progress counts no question")
	}
}

func TestReportCountsWhileExploring(t *testing.T) {
	defer func(interval time.Duration) { progressInterval = interval }(progressInterval)
	progressInterval = 10 * time.Millisecond

	rr, _ := dns.NewRR("www.example.com. 300 IN A 192.0.2.1")
	e := &explorer{
		opts:     Options{MaxQueries: 10},
		records:  map[string]dns.RR{},
		progress: happydns.NewImportProgress(),
	}
	stop := e.reportCounts()
	defer stop()

	e.addRecord(rr)
	e.queries.Add(15)

	deadline := time.Now().Add(time.Second)
	for s := e.progress.Snapshot(); s.Records != 1 || s.Queries != 10; s = e.progress.Snapshot() {
		if time.Now().After(deadline) {
			t.Fatalf("the progress counts %d records and %d questions; want 1 and the 10 of the budget, before the end", s.Records, s.Queries)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestPlanBands(t *testing.T) {
	for _, transfer := range []bool{true, false} {
		b := planBands(transfer)
		stages := []struct {
			name string
			band band
		}{
			{"locating", b.locating},
			{"transfer", b.transfer},
			{"apex", b.apex},
			{"collecting", b.collecting},
			{"exploring", b.exploring},
			{"following", b.following},
		}

		// The stages follow each other, from 0% to the end of the
		// retrieval.
		var at float64
		for _, stage := range stages {
			if math.Abs(stage.band.from-at) > 1e-9 || stage.band.to < stage.band.from {
				t.Errorf("transfer %v: the %s spans %v%% to %v%%; want from %v%%", transfer, stage.name, stage.band.from, stage.band.to, at)
			}
			at = stage.band.to
		}
		if math.Abs(at-happydns.ImportRetrievedPercent) > 1e-9 {
			t.Errorf("transfer %v: the stages end at %v%%; want %d%%", transfer, at, happydns.ImportRetrievedPercent)
		}
	}

	with, without := planBands(true), planBands(false)
	if w := without.transfer.to - without.transfer.from; w != 0 {
		t.Errorf("without a source handing the whole zone, the transfer spans %v%%; want none", w)
	}
	if without.exploring.to-without.exploring.from <= with.exploring.to-with.exploring.from {
		t.Error("the share of the transfer left out was not handed to the other stages")
	}
}
