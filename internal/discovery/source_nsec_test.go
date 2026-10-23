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
	"log"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
)

// testDNSSEC tunes how a testZone denies existence.
type testDNSSEC struct {
	// nsec signs the zone with an NSEC chain.
	nsec bool
	// nsec3 denies existence with NSEC3 records instead.
	nsec3 bool
	// compactDenial answers every NSEC question about x with a record
	// from x to \000.x, as servers signing on the fly do (RFC 9824).
	compactDenial bool
	// whiteLies makes the NSEC record of every name point to a made-up
	// next name.
	whiteLies bool
	// refuseNSECQuestions answers NOTIMP to questions of type NSEC.
	refuseNSECQuestions bool
	// next overrides the next name of the NSEC record of an owner.
	next map[string]string
}

// canonicalCompare orders names as RFC 4034 section 6.1 does, for the
// plain names of the tests.
func canonicalCompare(a, b string) int {
	la, lb := dns.SplitDomainName(a), dns.SplitDomainName(b)
	for i := 1; i <= len(la) && i <= len(lb); i++ {
		x := strings.ReplaceAll(strings.ToLower(la[len(la)-i]), `\000`, "\x00")
		y := strings.ReplaceAll(strings.ToLower(lb[len(lb)-i]), `\000`, "\x00")
		if c := strings.Compare(x, y); c != 0 {
			return c
		}
	}
	return len(la) - len(lb)
}

// nsecOwners returns the names of the zone owning an NSEC record, in
// canonical order: those holding records, but none below a delegation.
func (z *testZone) nsecOwners() (owners []string) {
	for _, rr := range z.rrs {
		name := rr.Header().Name
		if slices.Contains(owners, name) || slices.ContainsFunc(z.cuts, func(cut string) bool { return name != cut && dns.IsSubDomain(cut, name) }) {
			continue
		}
		owners = append(owners, name)
	}
	slices.SortFunc(owners, canonicalCompare)
	return
}

func (z *testZone) nsecFor(owner string) *dns.NSEC {
	types := []uint16{dns.TypeNSEC, dns.TypeRRSIG}
	for _, rr := range z.at(owner) {
		if !slices.Contains(types, rr.Header().Rrtype) {
			types = append(types, rr.Header().Rrtype)
		}
	}
	slices.Sort(types)

	owners := z.nsecOwners()
	next := z.origin
	if i := slices.Index(owners, owner); i >= 0 && i+1 < len(owners) {
		next = owners[i+1]
	}
	switch {
	case z.dnssec.next[owner] != "":
		next = z.dnssec.next[owner]
	case z.dnssec.whiteLies:
		next = "fake." + owner
	}

	return &dns.NSEC{
		Hdr:        dns.RR_Header{Name: owner, Rrtype: dns.TypeNSEC, Class: dns.ClassINET, Ttl: 300},
		NextDomain: next,
		TypeBitMap: types,
	}
}

// answerNSEC answers questions of type NSEC, as a signed zone does, and
// tells whether it did.
func (z *testZone) answerNSEC(m *dns.Msg, name string, qtype uint16) bool {
	if qtype != dns.TypeNSEC || !z.dnssec.nsec {
		return false
	}
	// At and below a cut, the referral comes first, as from a real server.
	if slices.ContainsFunc(z.cuts, func(cut string) bool { return dns.IsSubDomain(cut, name) }) {
		return false
	}

	switch {
	case z.dnssec.refuseNSECQuestions:
		m.Rcode = dns.RcodeNotImplemented
	case z.dnssec.compactDenial:
		nsec := z.nsecFor(name)
		nsec.Hdr.Name = name
		nsec.NextDomain = "\\000." + name
		m.Authoritative = true
		m.Answer = []dns.RR{nsec}
	case slices.Contains(z.nsecOwners(), name):
		m.Authoritative = true
		m.Answer = []dns.RR{z.nsecFor(name)}
	default:
		return false
	}

	return true
}

// proveDenial adds to a negative answer the NSEC or NSEC3 records proving
// it.
func (z *testZone) proveDenial(m *dns.Msg, name string) {
	if !m.Authoritative || len(m.Answer) > 0 {
		return
	}

	if z.dnssec.nsec3 {
		rr, _ := dns.NewRR(fmt.Sprintf("0p9mhaveqvm6t7vbl5lop2u3t2rp3tom.%s 300 IN NSEC3 1 0 0 - 2t7b4g4vsa5smi47k61mv5bv1a22bojr A RRSIG", z.origin))
		m.Ns = append(m.Ns, rr)
		return
	}
	if !z.dnssec.nsec {
		return
	}

	owners := z.nsecOwners()
	if slices.Contains(owners, name) {
		m.Ns = append(m.Ns, z.nsecFor(name))
		return
	}

	// The record covering name is owned by the last name before it.
	covering := owners[len(owners)-1]
	for _, owner := range owners {
		if canonicalCompare(owner, name) < 0 {
			covering = owner
		}
	}
	m.Ns = append(m.Ns, z.nsecFor(covering))
}

// nsecTarget serves z and returns a Target asking it, within budget
// questions when budget is positive.
func nsecTarget(t *testing.T, z *testZone, budget int) Target {
	t.Helper()

	servers := z.serve(t).Resolvers
	client := &dns.Client{Timeout: time.Second}

	var asked atomic.Int64
	return Target{
		Domain: z.origin,
		Query: func(ctx context.Context, name string, qtype uint16) (*dns.Msg, error) {
			if budget > 0 && asked.Add(1) > int64(budget) {
				return nil, errBudget
			}

			r, err := helpers.Question(ctx, client, servers, name, qtype, helpers.QueryOptions{})
			if err == nil && r.Rcode != dns.RcodeSuccess && r.Rcode != dns.RcodeNameError {
				return nil, fmt.Errorf("answered %s", dns.RcodeToString[r.Rcode])
			}
			return r, err
		},
	}
}

func nsecTestZone(t *testing.T) *testZone {
	z := newTestZone(t, "example.com.",
		"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
		"example.com. 3600 IN NS ns1.example.com.",
		"example.com. 300 IN A 192.0.2.1",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"_dmarc.example.com. 300 IN TXT \"v=DMARC1; p=reject\"",
		"host.example.com. 300 IN A 192.0.2.30",
		"host.example.com. 300 IN SSHFP 1 1 123456789abcdef67890123456789abcdef67890",
		"www.example.com. 300 IN CNAME host.example.com.",
	)
	z.dnssec.nsec = true
	return z
}

// walked returns the names, with their types, of a walk.
func walked(names []Name) (ret []string) {
	for _, name := range names {
		var types []string
		for _, rrtype := range name.Types {
			types = append(types, dns.TypeToString[rrtype])
		}
		ret = append(ret, name.FQDN+" "+strings.Join(types, ","))
	}
	return
}

func TestNSECFactory(t *testing.T) {
	f := factory(t, "nsec")

	if _, ok := f.New(settingsFrom(t, `{}`)).(NSECSource); !ok {
		t.Error("the NSEC walk is disabled by default")
	}
	if source := f.New(settingsFrom(t, `{"disable_nsec_walk":true}`)); source != nil {
		t.Errorf("disable_nsec_walk gives %v; want no source", source)
	}
}

func TestNSECWalk(t *testing.T) {
	want := []string{
		"example.com. A,NS,SOA",
		"_dmarc.example.com. TXT",
		"host.example.com. A,SSHFP",
		"ns1.example.com. A",
		"www.example.com. CNAME",
	}

	for _, refuse := range []bool{false, true} {
		t.Run(fmt.Sprintf("refuseNSECQuestions=%v", refuse), func(t *testing.T) {
			z := nsecTestZone(t)
			z.dnssec.refuseNSECQuestions = refuse

			names, complete, err := NSECSource{}.DiscoverAll(context.Background(), nsecTarget(t, z, 0))
			if err != nil {
				t.Fatalf("DiscoverAll: %v", err)
			}
			if !complete {
				t.Error("the walk of a whole chain is not complete")
			}
			if got := walked(names); !slices.Equal(got, want) {
				t.Errorf("walk = %q; want %q", got, want)
			}
		})
	}
}

func TestNSECWalkDelegation(t *testing.T) {
	z := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"sub.example.com. 300 IN NS ns.sub.example.com.",
		"ns.sub.example.com. 300 IN A 192.0.2.53",
		"zzz.example.com. 300 IN A 192.0.2.2",
	)
	z.dnssec.nsec = true

	names, complete, err := NSECSource{}.DiscoverAll(context.Background(), nsecTarget(t, z, 0))
	if err != nil || !complete {
		t.Fatalf("DiscoverAll = %v, %v; want a complete walk", complete, err)
	}

	want := []string{"example.com. NS", "ns1.example.com. A", "sub.example.com. NS", "zzz.example.com. A"}
	if got := walked(names); !slices.Equal(got, want) {
		t.Errorf("walk = %q; want %q, nothing below the delegation", got, want)
	}
}

func TestNSECWalkSecureDelegation(t *testing.T) {
	z := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"sub.example.com. 300 IN NS ns.sub.example.com.",
		"sub.example.com. 300 IN DS 12345 13 2 0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
		"ns.sub.example.com. 300 IN A 192.0.2.53",
		"zzz.example.com. 300 IN A 192.0.2.2",
	)
	z.dnssec.nsec = true

	// The referral to a signed child carries its DS, and no NSEC.
	names, complete, err := NSECSource{}.DiscoverAll(context.Background(), nsecTarget(t, z, 0))
	if err != nil || !complete {
		t.Fatalf("DiscoverAll = %v, %v; want a complete walk", complete, err)
	}

	want := []string{"example.com. NS", "ns1.example.com. A", "sub.example.com. NS,DS", "zzz.example.com. A"}
	if got := walked(names); !slices.Equal(got, want) {
		t.Errorf("walk = %q; want %q, nothing below the delegation", got, want)
	}

	// Below the delegation, the referral is the same: not worth asking.
	if z.wasAsked("\\000.sub.example.com.") {
		t.Error("the walk asked below a delegation after its referral")
	}
}

func TestNSECWalkLongLabelDelegation(t *testing.T) {
	long := strings.Repeat("a", 63) + ".example.com."
	z := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		long+" 300 IN NS ns.sub.example.com.",
		long+" 300 IN DS 12345 13 2 0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
		"zzz.example.com. 300 IN A 192.0.2.2",
	)
	z.cuts = []string{long}
	z.dnssec.nsec = true

	// No zero octet fits after a label of 63 octets.
	names, complete, err := NSECSource{}.DiscoverAll(context.Background(), nsecTarget(t, z, 0))
	if err != nil || !complete {
		t.Fatalf("DiscoverAll = %v, %v; want a complete walk", complete, err)
	}

	want := []string{"example.com. NS", long + " NS,DS", "ns1.example.com. A", "zzz.example.com. A"}
	if got := walked(names); !slices.Equal(got, want) {
		t.Errorf("walk = %q; want %q", got, want)
	}
}

func TestNSECWalkChildOnSameServer(t *testing.T) {
	for _, signed := range []bool{true, false} {
		t.Run(fmt.Sprintf("signed=%v", signed), func(t *testing.T) {
			parent := newTestZone(t, "example.com.",
				"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
				"example.com. 3600 IN NS ns1.example.com.",
				"ns1.example.com. 300 IN A 127.0.0.1",
				"sub.example.com. 300 IN NS ns1.example.com.",
				"zzz.example.com. 300 IN A 192.0.2.2",
			)
			parent.cuts = []string{"sub.example.com."}
			parent.dnssec.nsec = true
			child := newTestZone(t, "sub.example.com.",
				"sub.example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
				"sub.example.com. 300 IN NS ns1.example.com.",
				"sub.example.com. 300 IN A 192.0.2.77",
				"www.sub.example.com. 300 IN A 192.0.2.78",
			)
			child.dnssec.nsec = signed

			// The name server serves both zones: it answers about the
			// child with the authority, and the NSEC chain, of the child.
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
			opts.QueriesPerSecond = -1

			records, err := Explore(context.Background(), "example.com.", nil, []NameSource{NSECSource{}}, opts)
			if err != nil {
				t.Fatalf("Explore: %v", err)
			}
			got := recordStrings(records)

			assertHas(t, got, "sub.example.com. 300 IN NS ns1.example.com.", "zzz.example.com. 300 IN A 192.0.2.2")
			if slices.Contains(got, "sub.example.com. 300 IN A 192.0.2.77") {
				t.Error("Explore kept a record of the child zone at the delegation")
			}
			assertHasNoName(t, got, "www.sub.example.com.")
			if child.wasAsked("\\000.sub.example.com.") {
				t.Error("the walk asked below a delegation after the child zone answered")
			}
		})
	}
}

func TestNSECWalkNSEC3AsksOnce(t *testing.T) {
	z := nsecTestZone(t)
	z.dnssec = testDNSSEC{nsec3: true}

	if _, _, err := (NSECSource{}).DiscoverAll(context.Background(), nsecTarget(t, z, 0)); err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}
	if n := z.questions.Load(); n != 1 {
		t.Errorf("the walk asked %d questions of a zone signed with NSEC3; want 1", n)
	}
}

func TestNSECWalkLogsBreak(t *testing.T) {
	z := nsecTestZone(t)
	target := nsecTarget(t, z, 0)
	query := target.Query
	target.Query = func(ctx context.Context, name string, qtype uint16) (*dns.Msg, error) {
		if dns.IsSubDomain("host.example.com.", name) || name == "host\\000.example.com." {
			return nil, errors.New("timeout")
		}
		return query(ctx, name, qtype)
	}

	var logged strings.Builder
	log.SetOutput(&logged)
	defer log.SetOutput(os.Stderr)

	_, complete, err := NSECSource{}.DiscoverAll(context.Background(), target)
	if err != nil || complete {
		t.Fatalf("DiscoverAll = %v, %v; want it cut short, without error", complete, err)
	}
	if !strings.Contains(logged.String(), "host.example.com.") || !strings.Contains(logged.String(), "timeout") {
		t.Errorf("the walk stopped at host.example.com. without telling why; log: %q", logged.String())
	}
}

func TestNameAfter(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		ok         bool
	}{
		{"sub.example.com.", "sub\\000.example.com.", true},
		{"SUB.Example.com.", "sub\\000.example.com.", true},
		{strings.Repeat("a", 63) + ".example.com.", strings.Repeat("a", 62) + "b.example.com.", true},
		// Upper case letters sort as lower case ones: none is in between.
		{strings.Repeat("a", 62) + "@.example.com.", strings.Repeat("a", 62) + "[.example.com.", true},
		{strings.Repeat("a", 62) + "\\255.example.com.", "", false},
		// No room left in the name.
		{strings.Repeat("x", 61) + "." + strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + ".", "", false},
		{".", "", false},
	} {
		got, ok := nameAfter(tc.name)
		if got != tc.want || ok != tc.ok {
			t.Errorf("nameAfter(%q) = %q, %v; want %q, %v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

func TestNSECWalkStops(t *testing.T) {
	for _, tc := range []struct {
		name   string
		dnssec testDNSSEC
		budget int
		// outOfTime is the number of questions after which the pace leaves
		// no time for another one.
		outOfTime int
		// walked are the names the walk is expected to have confirmed.
		walked  int
		wantErr error
	}{
		{name: "unsigned"},
		{name: "NSEC3", dnssec: testDNSSEC{nsec3: true}},
		{name: "compact denial", dnssec: testDNSSEC{nsec: true, compactDenial: true}, walked: 1},
		{name: "white lies", dnssec: testDNSSEC{nsec: true, whiteLies: true}, walked: 1},
		{name: "loop", dnssec: testDNSSEC{nsec: true, next: map[string]string{"ns1.example.com.": "_dmarc.example.com."}}, walked: 4},
		{name: "out of zone", dnssec: testDNSSEC{nsec: true, next: map[string]string{"host.example.com.": "other.test."}}, walked: 3},
		{name: "budget", dnssec: testDNSSEC{nsec: true}, budget: 2, walked: 2, wantErr: errBudget},
		{name: "out of time", dnssec: testDNSSEC{nsec: true}, outOfTime: 2, walked: 2, wantErr: errOutOfTime},
	} {
		t.Run(tc.name, func(t *testing.T) {
			z := nsecTestZone(t)
			z.dnssec = tc.dnssec

			target := nsecTarget(t, z, tc.budget)
			if tc.outOfTime > 0 {
				query, asked := target.Query, 0
				target.Query = func(ctx context.Context, name string, qtype uint16) (*dns.Msg, error) {
					if asked++; asked > tc.outOfTime {
						return nil, errOutOfTime
					}
					return query(ctx, name, qtype)
				}
			}

			names, complete, err := NSECSource{}.DiscoverAll(context.Background(), target)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("DiscoverAll error = %v; want %v", err, tc.wantErr)
			}
			if complete {
				t.Error("the walk is complete; want it cut short")
			}
			if len(names) != tc.walked {
				t.Errorf("walk = %q; want %d names", walked(names), tc.walked)
			}
		})
	}
}

func TestExploreNSEC(t *testing.T) {
	z := newTestZone(t, "example.com.",
		"example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 7200 3600 1209600 300",
		"example.com. 3600 IN NS ns1.example.com.",
		"example.com. 300 IN SSHFP 1 1 123456789abcdef67890123456789abcdef67890",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"*.example.com. 300 IN A 192.0.2.99",
		"same.example.com. 300 IN A 192.0.2.99",
		"hidden.example.com. 300 IN A 192.0.2.30",
		"host.example.com. 300 IN A 192.0.2.31",
		"host.example.com. 300 IN SSHFP 1 1 123456789abcdef67890123456789abcdef67890",
		"sub.example.com. 300 IN NS ns.sub.example.com.",
		"ns.sub.example.com. 300 IN A 192.0.2.53",
	)
	z.dnssec.nsec = true
	opts := z.serve(t)

	records, err := Explore(context.Background(), "example.com.", []string{"host"}, []NameSource{NSECSource{}, staticSource{"guess.example.com."}}, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	got := recordStrings(records)

	assertHas(t, got,
		// A type the apex is not asked for by default.
		"example.com. 300 IN SSHFP 1 1 123456789ABCDEF67890123456789ABCDEF67890",
		// A name nothing points to.
		"hidden.example.com. 300 IN A 192.0.2.30",
		// A known name, asked for the types of its NSEC record.
		"host.example.com. 300 IN SSHFP 1 1 123456789ABCDEF67890123456789ABCDEF67890",
		// A real name answering like the wildcard.
		"same.example.com. 300 IN A 192.0.2.99",
		"*.example.com. 300 IN A 192.0.2.99",
		// A delegation.
		"sub.example.com. 300 IN NS ns.sub.example.com.",
	)

	if z.wasAsked("guess.example.com.") {
		t.Error("a guess was asked about although the NSEC walk found every name")
	}
}

func TestExploreNSECIncomplete(t *testing.T) {
	z := nsecTestZone(t)
	z.dnssec = testDNSSEC{nsec3: true}
	opts := z.serve(t)

	if _, err := Explore(context.Background(), "example.com.", nil, []NameSource{NSECSource{}, staticSource{"guess.example.com."}}, opts); err != nil {
		t.Fatalf("Explore: %v", err)
	}

	if !z.wasAsked("guess.example.com.") {
		t.Error("the guesses were not asked about after a walk that could not be completed")
	}
}

func TestNSECWalkWithoutQuery(t *testing.T) {
	names, complete, err := NSECSource{}.DiscoverAll(context.Background(), Target{Domain: "example.com."})
	if names != nil || complete || err != nil {
		t.Errorf("DiscoverAll without Query = %v, %v, %v; want nothing", names, complete, err)
	}
}

func TestNSECWalkCanceled(t *testing.T) {
	z := nsecTestZone(t)
	target := nsecTarget(t, z, 0)

	// The context ends after the walk has confirmed two names.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	query, asked := target.Query, 0
	target.Query = func(ctx context.Context, name string, qtype uint16) (*dns.Msg, error) {
		if asked++; asked > 2 {
			cancel()
			return nil, ctx.Err()
		}
		return query(ctx, name, qtype)
	}

	names, complete, err := NSECSource{}.DiscoverAll(ctx, target)
	if !errors.Is(err, context.Canceled) || complete {
		t.Errorf("DiscoverAll = %v, %v; want it canceled", complete, err)
	}
	if len(names) != 2 {
		t.Errorf("walk = %q; want the 2 names confirmed before", walked(names))
	}
}

func TestZoneTypes(t *testing.T) {
	if got := zoneTypes([]uint16{dns.TypeRRSIG, dns.TypeNSEC}); len(got) != 0 {
		t.Errorf("zoneTypes of an empty non-terminal = %v; want none", got)
	}
	if got := zoneTypes([]uint16{dns.TypeA, dns.TypeRRSIG, dns.TypeNSEC, dns.TypeDNSKEY, dns.TypeTXT}); !slices.Equal(got, []uint16{dns.TypeA, dns.TypeTXT}) {
		t.Errorf("zoneTypes = %v; want A and TXT", got)
	}
}

func TestExploreNSECWildcard(t *testing.T) {
	z := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"*.example.com. 300 IN A 192.0.2.99",
		"host.example.com. 300 IN A 192.0.2.1",
	)
	z.dnssec.nsec = true
	opts := z.serve(t)

	records, err := Explore(context.Background(), "example.com.", nil, []NameSource{NSECSource{}}, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}

	// The walk meets the wildcard as any other owner, its records are kept once.
	got := recordStrings(records)
	assertHas(t, got, "*.example.com. 300 IN A 192.0.2.99", "host.example.com. 300 IN A 192.0.2.1")
	if n := len(slices.DeleteFunc(slices.Clone(got), func(s string) bool { return !strings.HasPrefix(s, "*.") })); n != 1 {
		t.Errorf("%d wildcard records in:\n%s", n, strings.Join(got, "\n"))
	}
}
