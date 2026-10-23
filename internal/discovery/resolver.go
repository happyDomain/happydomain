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
	"cmp"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"maps"
	"net"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/time/rate"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

// Default values of Options.
//
// The budget and the timeout leave room to ask about the whole
// WordlistSource, nearly one question per name, at the default pace.
const (
	DefaultConcurrency      = 16
	DefaultMaxQueries       = 7000
	DefaultMaxDepth         = 3
	DefaultQueriesPerSecond = 100
	DefaultTimeout          = 90 * time.Second
	DefaultQueryTimeout     = 3 * time.Second
)

// Record types asked for, after the shape of the name, when its source did
// not tell which types exist there. The first type also tells apart a name
// that does not exist, an alias, a delegation and a name made up by a
// wildcard, so the remaining ones are only asked for names worth it.
var (
	apexTypes       = []uint16{dns.TypeSOA, dns.TypeNS, dns.TypeA, dns.TypeAAAA, dns.TypeMX, dns.TypeTXT, dns.TypeCAA, dns.TypeHTTPS}
	hostTypes       = []uint16{dns.TypeA, dns.TypeAAAA, dns.TypeMX, dns.TypeTXT, dns.TypeCAA, dns.TypeHTTPS}
	underscoreTypes = []uint16{dns.TypeTXT, dns.TypeSRV, dns.TypeTLSA}
)

// wildcardProbeTypes are asked for at a random name, to learn what a
// wildcard, if any, answers.
var wildcardProbeTypes = []uint16{dns.TypeA, dns.TypeAAAA, dns.TypeMX, dns.TypeTXT}

// errBudget is returned once Options.MaxQueries questions have been asked.
var errBudget = errors.New("query budget exhausted")

// Options tunes an exploration. Zero values take the defaults.
type Options struct {
	// Guard vets every DNS server the exploration talks to. Their
	// addresses come from DNS answers, controlled by whoever runs the
	// domain: without the guard, they would make happyDomain query its own
	// network. A nil Guard refuses every non-public destination.
	Guard *netguard.Guard

	// Resolver is the recursive resolver ("host" or "host:port") asked to
	// find the authoritative servers. Empty uses the one of
	// /etc/resolv.conf, chosen by the operator and therefore not vetted.
	Resolver string

	// Concurrency is the number of names explored at once.
	Concurrency int

	// MaxQueries caps the number of questions asked to the authoritative
	// servers.
	MaxQueries int

	// QueriesPerSecond caps the pace of the questions asked to each
	// authoritative server, which may limit the rate of its answers (RRL)
	// to whoever asks too fast. A negative value lifts the cap.
	QueriesPerSecond int

	// MaxDepth caps how many times in-zone targets of answers are
	// followed, from the names the sources yielded.
	MaxDepth int

	// Timeout bounds the whole exploration.
	Timeout time.Duration

	// QueryTimeout bounds each question.
	QueryTimeout time.Duration

	// authPort is the port of the authoritative servers. Tests set it, as
	// they cannot listen on 53.
	authPort string
}

func (o Options) withDefaults() Options {
	if o.Concurrency <= 0 {
		o.Concurrency = DefaultConcurrency
	}
	if o.MaxQueries <= 0 {
		o.MaxQueries = DefaultMaxQueries
	}
	if o.QueriesPerSecond == 0 {
		o.QueriesPerSecond = DefaultQueriesPerSecond
	}
	if o.MaxDepth <= 0 {
		o.MaxDepth = DefaultMaxDepth
	}
	if o.Timeout <= 0 {
		o.Timeout = DefaultTimeout
	}
	if o.QueryTimeout <= 0 {
		o.QueryTimeout = DefaultQueryTimeout
	}
	if o.authPort == "" {
		o.authPort = "53"
	}
	return o
}

// Locate returns the authoritative servers of domain the guard allows, which
// is all it takes to tell the domain exists, without exploring it.
func Locate(ctx context.Context, domain string, opts Options) ([]string, error) {
	opts = opts.withDefaults()

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	e := &explorer{
		domain: dns.CanonicalName(domain),
		opts:   opts,
		udp:    &dns.Client{Timeout: opts.QueryTimeout},
	}
	return e.locate(ctx)
}

// locate finds the authoritative servers, through the recursive resolver.
func (e *explorer) locate(ctx context.Context) ([]string, error) {
	recursive, err := e.recursiveResolver(ctx)
	if err != nil {
		return nil, err
	}

	return e.findAuthoritative(ctx, recursive)
}

// Resolve explores domain and returns the records its authoritative servers
// answer for the names yielded by sources, the apex and the known
// subdomains, as absolute records.
//
// The exploration is best effort: questions that fail, or that the budget
// or the timeout leave unasked, are logged and skipped. An error is only
// returned when nothing could be learned about the domain.
func Resolve(ctx context.Context, domain string, known []string, sources []NameSource, opts Options) ([]happydns.Record, error) {
	opts = opts.withDefaults()
	domain = dns.CanonicalName(domain)

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	e := &explorer{
		domain:  domain,
		opts:    opts,
		udp:     &dns.Client{Timeout: opts.QueryTimeout},
		tcp:     &dns.Client{Net: "tcp", Timeout: opts.QueryTimeout},
		seen:    map[string]bool{},
		records: map[string]dns.RR{},
	}

	var err error
	e.servers, err = e.locate(ctx)
	if err != nil {
		return nil, err
	}
	e.failures = map[string]*atomic.Int32{}
	e.pace = map[string]*rate.Limiter{}
	for _, server := range e.servers {
		e.failures[server] = new(atomic.Int32)
		e.pace[server] = rate.NewLimiter(rate.Inf, 0)
		if opts.QueriesPerSecond > 0 {
			e.pace[server] = rate.NewLimiter(rate.Limit(opts.QueriesPerSecond), opts.Concurrency)
		}
	}

	target := Target{
		Domain:        domain,
		Authoritative: e.servers,
		Known:         known,
		Query:         e.query,
	}

	// A source handing the whole zone spares the exploration.
	for _, source := range sources {
		rs, ok := source.(RecordSource)
		if !ok {
			continue
		}

		rrs, err := rs.Records(ctx, target)
		if err != nil {
			log.Printf("discovery: %s: source %s: %s", domain, source.ID(), err)
		}
		if len(rrs) > 0 {
			for _, rr := range rrs {
				// The server of the zone is not trusted for other zones.
				if dns.IsSubDomain(domain, dns.CanonicalName(rr.Header().Name)) {
					e.addRecord(rr)
				}
			}
			return e.result(), nil
		}
	}

	// The apex comes first, then the known names: should the budget run
	// out, the names already in the zone are the ones not to lose. A
	// wildcard never covers the apex, so it is explored before the probe.
	names := e.exploreAll(ctx, e.unseen([]Name{{FQDN: domain, Sources: []string{"apex"}}}))

	e.probeWildcard(ctx)

	for _, source := range append([]NameSource{KnownSource{}}, sources...) {
		var (
			found    []Name
			complete bool
			err      error
		)
		if es, ok := source.(ExhaustiveSource); ok {
			found, complete, err = es.DiscoverAll(ctx, target)
		} else {
			found, err = source.Discover(ctx, target)
		}
		if err != nil {
			log.Printf("discovery: %s: source %s: %s", domain, source.ID(), err)
		}
		names = append(names, found...)

		// Every name is known: the remaining sources would only guess.
		if complete {
			break
		}
	}

	names = append(names, e.exploreApexTypes(ctx, names)...)

	round := e.unseen(names)
	for depth := 0; len(round) > 0 && depth <= opts.MaxDepth && ctx.Err() == nil; depth++ {
		round = e.unseen(e.exploreAll(ctx, round))
	}

	if len(e.records) == 0 {
		if e.lastErr != nil {
			return nil, fmt.Errorf("unable to learn anything about %s: %w", domain, e.lastErr)
		}
		return nil, fmt.Errorf("the authoritative servers of %s answered no record", domain)
	}

	if ctx.Err() != nil || e.queries.Load() > int64(opts.MaxQueries) {
		log.Printf("discovery: %s: exploration cut short after %d questions", domain, e.queries.Load())
	}

	return e.result(), nil
}

// explorer holds the state of one exploration.
type explorer struct {
	domain string
	opts   Options

	udp *dns.Client
	tcp *dns.Client

	// servers are the "ip:port" of the authoritative servers.
	servers []string
	// failures counts, by server, the questions it failed in a row: the
	// servers failing are asked last, as each of their failures costs a
	// round trip, or a timeout, to every question.
	failures map[string]*atomic.Int32

	queries atomic.Int64

	// next is the turn of the questions, spreading them over the servers.
	next atomic.Uint32

	// pace spaces out the questions asked to each authoritative server.
	pace map[string]*rate.Limiter

	mu sync.Mutex
	// seen are the names already explored or scheduled.
	seen map[string]bool
	// records are the records found, by their text form.
	records map[string]dns.RR
	// wildcard are the answers of the wildcard by type, nil without
	// wildcard.
	wildcard map[uint16]string
	lastErr  error
}

// recursiveResolver returns the "ip:port" of the recursive resolver to use.
func (e *explorer) recursiveResolver(ctx context.Context) (string, error) {
	if e.opts.Resolver != "" {
		resolver, err := e.opts.Guard.ResolveAddrPort(ctx, e.opts.Resolver, 53)
		if err != nil {
			return "", fmt.Errorf("unable to use the resolver %q: %w", e.opts.Resolver, err)
		}
		return resolver, nil
	}

	conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return "", fmt.Errorf("unable to read the system resolver: %w", err)
	}
	if len(conf.Servers) == 0 {
		return "", errors.New("no resolver in /etc/resolv.conf")
	}
	return net.JoinHostPort(conf.Servers[0], conf.Port), nil
}

// findAuthoritative asks the recursive resolver for the name servers of the
// domain and returns those of their addresses the guard allows.
func (e *explorer) findAuthoritative(ctx context.Context, recursive string) ([]string, error) {
	r, err := helpers.Question(ctx, e.udp, recursive, e.domain, dns.TypeNS, helpers.QueryOptions{Recursive: true})
	if err != nil {
		return nil, fmt.Errorf("unable to find the name servers of %s: %w", e.domain, err)
	}

	var nsNames []string
	for _, rr := range r.Answer {
		if ns, ok := rr.(*dns.NS); ok && dns.CanonicalName(ns.Hdr.Name) == e.domain {
			nsNames = append(nsNames, ns.Ns)
		}
	}
	if len(nsNames) == 0 {
		return nil, fmt.Errorf("%s has no name server: is it a registered domain?", e.domain)
	}

	var servers []string
	for _, nsName := range nsNames {
		for _, qtype := range []uint16{dns.TypeA, dns.TypeAAAA} {
			r, err := helpers.Question(ctx, e.udp, recursive, nsName, qtype, helpers.QueryOptions{Recursive: true})
			if err != nil {
				continue
			}

			for _, rr := range r.Answer {
				var ip net.IP
				switch rr := rr.(type) {
				case *dns.A:
					ip = rr.A
				case *dns.AAAA:
					ip = rr.AAAA
				default:
					continue
				}

				addr, ok := netip.AddrFromSlice(ip)
				if !ok || !e.opts.Guard.AllowsAddr(addr) {
					log.Printf("discovery: %s: name server %s at %s is not an allowed destination", e.domain, nsName, ip)
					continue
				}

				server := net.JoinHostPort(addr.Unmap().String(), e.opts.authPort)
				if !slices.Contains(servers, server) {
					servers = append(servers, server)
				}
			}
		}
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("no name server of %s can be reached", e.domain)
	}

	return servers, nil
}

// query asks the authoritative servers, one after the other, until one of
// them gives an answer that settles the question.
func (e *explorer) query(ctx context.Context, name string, qtype uint16) (*dns.Msg, error) {
	if e.queries.Add(1) > int64(e.opts.MaxQueries) {
		return nil, errBudget
	}

	var lastErr error
	for _, server := range e.serversByHealth() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		r, err := e.ask(ctx, e.udp, server, name, qtype)
		if err == nil && r.Truncated {
			r, err = e.ask(ctx, e.tcp, server, name, qtype)
		}
		if err == nil && r.Rcode != dns.RcodeSuccess && r.Rcode != dns.RcodeNameError {
			err = fmt.Errorf("%s answered %s", server, dns.RcodeToString[r.Rcode])
		}
		if err != nil {
			// Running out of time or budget is not the server's fault.
			if ctx.Err() == nil {
				e.failures[server].Add(1)
			}
			lastErr = err
			continue
		}

		e.failures[server].Store(0)
		return r, nil
	}

	return nil, fmt.Errorf("no authoritative server answered %s %s: %w", name, dns.TypeToString[qtype], lastErr)
}

// serversByHealth returns the servers, those failing the fewest questions
// in a row first. Successive questions start with each of the servers
// equally healthy in turn: each server is paced on its own, so sharing the
// questions among them makes the exploration as much faster.
func (e *explorer) serversByHealth() []string {
	turn := int(e.next.Add(1)-1) % len(e.servers)
	servers := append(slices.Clone(e.servers[turn:]), e.servers[:turn]...)
	slices.SortStableFunc(servers, func(a, b string) int {
		return cmp.Compare(e.failures[a].Load(), e.failures[b].Load())
	})
	return servers
}

// ask sends one question to an authoritative server, at the pace allowed.
func (e *explorer) ask(ctx context.Context, client *dns.Client, server, name string, qtype uint16) (*dns.Msg, error) {
	if err := e.pace[server].Wait(ctx); err != nil {
		return nil, err
	}

	return helpers.Question(ctx, client, server, name, qtype, helpers.QueryOptions{})
}

// probeWildcard asks about a random name, which only a wildcard can answer
// for, and remembers the answers to tell them apart from real names.
func (e *explorer) probeWildcard(ctx context.Context) {
	label := make([]byte, 8)
	rand.Read(label)
	probe := "hd-probe-" + hex.EncodeToString(label) + "." + e.domain

	for _, qtype := range wildcardProbeTypes {
		r, err := e.query(ctx, probe, qtype)
		if err != nil || r.Rcode == dns.RcodeNameError {
			return
		}
		// Even answering none of the types probed, the wildcard exists.
		if e.wildcard == nil {
			e.wildcard = map[uint16]string{}
		}

		for _, rr := range answersAt(r, probe) {
			if rr.Header().Rrtype != qtype && rr.Header().Rrtype != dns.TypeCNAME {
				continue
			}

			wildcardRR := dns.Copy(rr)
			wildcardRR.Header().Name = "*." + e.domain
			e.addRecord(wildcardRR)
		}

		for rrtype, rdata := range rdataByType(answersAt(r, probe)) {
			e.wildcard[rrtype] = rdata
		}
	}
}

// unseen returns the names not explored yet, merging duplicates, and marks
// them as seen.
func (e *explorer) unseen(names []Name) (ret []Name) {
	e.mu.Lock()
	defer e.mu.Unlock()

	index := map[string]int{}
	for _, name := range names {
		fqdn := dns.CanonicalName(name.FQDN)
		if !dns.IsSubDomain(e.domain, fqdn) {
			continue
		}

		if i, ok := index[fqdn]; ok {
			ret[i].Sources = append(ret[i].Sources, name.Sources...)
			// A source knowing the types present at the name beats
			// asking for the default ones.
			if len(ret[i].Types) == 0 {
				ret[i].Types = name.Types
			}
			continue
		}
		if e.seen[fqdn] {
			continue
		}

		e.seen[fqdn] = true
		name.FQDN = fqdn
		index[fqdn] = len(ret)
		ret = append(ret, name)
	}

	return
}

// exploreAll explores names concurrently and returns the in-zone targets
// their answers point to.
func (e *explorer) exploreAll(ctx context.Context, names []Name) []Name {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		targets []Name
		sem     = make(chan struct{}, e.opts.Concurrency)
	)

	for _, name := range names {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return targets
		}

		wg.Add(1)
		go func(name Name) {
			defer wg.Done()
			defer func() { <-sem }()

			found := e.explore(ctx, name)

			mu.Lock()
			targets = append(targets, found...)
			mu.Unlock()
		}(name)
	}

	wg.Wait()
	return targets
}

// explore asks about one name and returns the in-zone targets of the
// records found there.
func (e *explorer) explore(ctx context.Context, name Name) (targets []Name) {
	types := name.Types
	if len(types) == 0 {
		types = e.typesFor(name.FQDN)
	}

	// A name whose source knows its types exists.
	real := name.FQDN == e.domain || len(name.Types) > 0
	for _, qtype := range types {
		r, err := e.query(ctx, name.FQDN, qtype)
		if err != nil {
			e.fail(err)
			if errors.Is(err, errBudget) || ctx.Err() != nil {
				return
			}
			continue
		}

		if r.Rcode == dns.RcodeNameError {
			return
		}

		// A referral: the name is, or is below, a delegation point. The
		// child zone is not ours to explore, only the delegation is.
		if cut := e.delegationIn(r); cut != "" {
			if cut != name.FQDN {
				return []Name{{FQDN: cut, Sources: []string{"target"}}}
			}

			for _, rr := range r.Ns {
				if dns.CanonicalName(rr.Header().Name) == cut && rr.Header().Rrtype == dns.TypeNS {
					e.addRecord(rr)
				}
			}
			for _, rr := range r.Extra {
				// Glue lives in the parent zone, ours.
				if rrtype := rr.Header().Rrtype; (rrtype == dns.TypeA || rrtype == dns.TypeAAAA) && dns.IsSubDomain(cut, dns.CanonicalName(rr.Header().Name)) {
					e.addRecord(rr)
				}
			}
			return
		}

		// The first answer tells a name the wildcard makes up from one that
		// exists, whose answers are then all its own, even those matching
		// the wildcard. A real name answering its first question like the
		// wildcard is lost: telling it apart would cost every name of a
		// list all the types, on a zone holding a wildcard.
		answers := answersAt(r, name.FQDN)
		if !real {
			if e.matchesWildcard(qtype, answers) {
				return
			}
			real = true
		}

		isAlias := false
		for _, rr := range answers {
			rrtype := rr.Header().Rrtype
			if rrtype != qtype && rrtype != dns.TypeCNAME {
				continue
			}

			e.addRecord(rr)
			for _, target := range targetsOf(rr) {
				targets = append(targets, Name{FQDN: target, Sources: []string{"target"}})
			}
			isAlias = isAlias || rrtype == dns.TypeCNAME
		}

		// No other record may live beside a CNAME.
		if isAlias {
			return
		}
	}

	return
}

// exploreApexTypes asks, at the apex, for the types names tell exist there
// beyond apexTypes. The apex is explored before any source is asked, with
// the default types only: rarer ones, such as SSHFP, would otherwise be
// missed. It returns the in-zone targets of the records found.
func (e *explorer) exploreApexTypes(ctx context.Context, names []Name) []Name {
	var extra []uint16
	for _, name := range names {
		if dns.CanonicalName(name.FQDN) != e.domain {
			continue
		}
		for _, rrtype := range name.Types {
			if !slices.Contains(apexTypes, rrtype) && !slices.Contains(extra, rrtype) {
				extra = append(extra, rrtype)
			}
		}
	}

	if len(extra) == 0 {
		return nil
	}
	return e.explore(ctx, Name{FQDN: e.domain, Types: extra})
}

// typesFor returns the record types to ask for at name.
func (e *explorer) typesFor(name string) []uint16 {
	if name == e.domain {
		return apexTypes
	}

	for _, label := range dns.SplitDomainName(strings.TrimSuffix(name, e.domain)) {
		if strings.HasPrefix(label, "_") {
			return underscoreTypes
		}
	}

	return hostTypes
}

// delegationIn returns the delegation point r is a referral to, if any.
func (e *explorer) delegationIn(r *dns.Msg) string {
	if r.Authoritative || len(r.Answer) > 0 {
		return ""
	}

	for _, rr := range r.Ns {
		if rr.Header().Rrtype != dns.TypeNS {
			continue
		}

		owner := dns.CanonicalName(rr.Header().Name)
		if owner != e.domain && dns.IsSubDomain(e.domain, owner) {
			return owner
		}
	}

	return ""
}

// matchesWildcard tells whether answers to qtype are exactly what the
// wildcard answers, an empty answer included, in which case they tell
// nothing about the name.
func (e *explorer) matchesWildcard(qtype uint16, answers []dns.RR) bool {
	if e.wildcard == nil || !slices.Contains(wildcardProbeTypes, qtype) {
		return false
	}

	want := map[uint16]string{}
	if cname, ok := e.wildcard[dns.TypeCNAME]; ok {
		want[dns.TypeCNAME] = cname
	} else if rdata, ok := e.wildcard[qtype]; ok {
		want[qtype] = rdata
	}

	return maps.Equal(rdataByType(answers), want)
}

func (e *explorer) addRecord(rr dns.RR) {
	if helpers.IsDNSSECType(rr.Header().Rrtype) {
		return
	}

	rr = dns.Copy(rr)
	rr.Header().Name = dns.CanonicalName(rr.Header().Name)

	e.mu.Lock()
	e.records[rr.String()] = rr
	e.mu.Unlock()
}

func (e *explorer) fail(err error) {
	if !errors.Is(err, errBudget) {
		log.Printf("discovery: %s: %s", e.domain, err)
	}

	e.mu.Lock()
	e.lastErr = err
	e.mu.Unlock()
}

// result returns the records found, sorted by name then type.
func (e *explorer) result() []happydns.Record {
	keys := make([]string, 0, len(e.records))
	for key := range e.records {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int {
		ra, rb := e.records[a].Header(), e.records[b].Header()
		if c := strings.Compare(ra.Name, rb.Name); c != 0 {
			return c
		}
		if ra.Rrtype != rb.Rrtype {
			return int(ra.Rrtype) - int(rb.Rrtype)
		}
		return strings.Compare(a, b)
	})

	ret := make([]happydns.Record, len(keys))
	for i, key := range keys {
		ret[i] = e.records[key]
	}
	return ret
}

// answersAt returns the records of the answer section owned by name.
func answersAt(r *dns.Msg, name string) (ret []dns.RR) {
	for _, rr := range r.Answer {
		if dns.CanonicalName(rr.Header().Name) == name {
			ret = append(ret, rr)
		}
	}
	return
}

// rdataByType returns, for each type, the sorted rdata of rrs: a form that
// compares two RRsets regardless of their owner, TTL and order.
func rdataByType(rrs []dns.RR) map[uint16]string {
	byType := map[uint16][]string{}
	for _, rr := range rrs {
		rrtype := rr.Header().Rrtype
		byType[rrtype] = append(byType[rrtype], strings.TrimPrefix(rr.String(), rr.Header().String()))
	}

	ret := map[uint16]string{}
	for rrtype, rdata := range byType {
		slices.Sort(rdata)
		ret[rrtype] = strings.Join(rdata, "\n")
	}
	return ret
}

// targetsOf returns the names rr points to.
func targetsOf(rr dns.RR) []string {
	var target string
	switch rr := rr.(type) {
	case *dns.CNAME:
		target = rr.Target
	case *dns.MX:
		target = rr.Mx
	case *dns.NS:
		target = rr.Ns
	case *dns.SRV:
		target = rr.Target
	case *dns.HTTPS:
		target = rr.Target
	case *dns.SVCB:
		target = rr.Target
	}

	if target == "" || target == "." {
		return nil
	}
	return []string{target}
}
