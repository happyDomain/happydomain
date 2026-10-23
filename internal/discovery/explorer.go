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
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/miekg/dns"
	"golang.org/x/time/rate"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

// Record types asked for, after the shape of the name, when its source did
// not tell which types exist there. The first type also tells apart a name
// that does not exist, an alias, a delegation and a name made up by a
// wildcard, so the remaining ones are only asked for names worth it.
var (
	apexTypes       = []uint16{dns.TypeSOA, dns.TypeNS, dns.TypeA, dns.TypeAAAA, dns.TypeMX, dns.TypeTXT, dns.TypeCAA, dns.TypeHTTPS, dns.TypeDNAME}
	hostTypes       = []uint16{dns.TypeA, dns.TypeAAAA, dns.TypeMX, dns.TypeTXT, dns.TypeCAA, dns.TypeHTTPS}
	underscoreTypes = []uint16{dns.TypeTXT, dns.TypeSRV, dns.TypeTLSA}
)

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
	return e.findAuthoritative(ctx)
}

// Explore explores domain and returns the records its authoritative servers
// answer for the names yielded by sources, the apex and the known
// subdomains, as absolute records.
//
// The exploration is best effort: questions that fail, or that the budget
// or the timeout leave unasked, are logged and skipped. An error is only
// returned when nothing could be learned about the domain.
func Explore(ctx context.Context, domain string, known []string, sources []NameSource, opts Options) ([]happydns.Record, error) {
	opts = opts.withDefaults()
	domain = dns.CanonicalName(domain)

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	e := &explorer{
		domain:  domain,
		opts:    opts,
		udp:     &dns.Client{Timeout: opts.QueryTimeout},
		seen:    map[string]bool{},
		records: map[string]dns.RR{},
	}

	var err error
	e.servers, err = e.findAuthoritative(ctx)
	if err != nil {
		return nil, err
	}
	e.failures = map[string]*atomic.Int32{}
	e.pace = map[string]*rate.Limiter{}
	for _, server := range e.servers {
		e.failures[server] = new(atomic.Int32)
		if opts.QueriesPerSecond > 0 {
			// A burst of one question per worker lets them all start at
			// once, the pace only holding them back afterwards.
			e.pace[server] = rate.NewLimiter(rate.Limit(opts.QueriesPerSecond), opts.Concurrency)
		} else {
			e.pace[server] = rate.NewLimiter(rate.Inf, 0)
		}
	}

	target := Target{
		Domain:        domain,
		Authoritative: e.servers,
		Known:         known,
	}

	// The apex comes first, then the known names: should the budget run
	// out, the names already in the zone are the ones not to lose. A
	// wildcard never covers the apex, so it is explored before the probe.
	names := e.exploreAll(ctx, e.unseen([]Name{{FQDN: domain, Sources: []string{"apex"}}}))

	names = append(names, e.probeWildcard(ctx)...)

	names = append(names, e.collectNames(ctx, target, sources)...)

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

	// udp asks the questions, again over TCP when the answer is truncated.
	udp *dns.Client

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

// collectNames asks the name sources, the known names first, for the
// names of the zone.
func (e *explorer) collectNames(ctx context.Context, target Target, sources []NameSource) (names []Name) {
	for _, source := range append([]NameSource{KnownSource{}}, sources...) {
		found, err := source.Discover(ctx, target)
		if err != nil {
			log.Printf("discovery: %s: source %s: %s", e.domain, source.ID(), err)
		}
		names = append(names, found...)
	}
	return
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

	real := name.FQDN == e.domain

	// The records of the name are kept once all its answers are in: a
	// later one may tell the name lies in a zone below ours.
	var found []dns.RR
	defer func() {
		for _, rr := range found {
			e.addRecord(rr)
		}
	}()

	for _, qtype := range types {
		r, err := e.query(ctx, name.FQDN, qtype)
		if err != nil {
			e.fail(err)
			if errors.Is(err, errBudget) || errors.Is(err, errOutOfTime) || ctx.Err() != nil {
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
			// What the name answered before belongs to the child zone.
			found = nil
			if cut != name.FQDN {
				return []Name{{FQDN: cut, Sources: []string{"target"}}}
			}
			return e.exploreDelegation(ctx, cut, r)
		}

		// Below a DNAME, no name exists: the server makes up an alias for
		// any. Only the DNAME is ours, and the name holding it.
		if dname := e.dnameAbove(r, name.FQDN); dname != nil {
			found, targets = nil, nil
			e.addRecord(dname)
			return []Name{{FQDN: dns.CanonicalName(dname.Header().Name), Sources: []string{"target"}}}
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

			found = append(found, rr)
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

// exploreDelegation keeps the delegation at cut, which r tells of, and
// returns the names of the zone its name servers are.
func (e *explorer) exploreDelegation(ctx context.Context, cut string, r *dns.Msg) (targets []Name) {
	// The same server serving the child zone answers with the
	// authority of the child, not a referral: the name servers
	// of the delegation are asked for.
	if r.Authoritative {
		ns, err := e.query(ctx, cut, dns.TypeNS)
		if err != nil {
			e.fail(err)
			return
		}
		r = &dns.Msg{Ns: append(ns.Answer, ns.Ns...), Extra: ns.Extra}
	}

	for _, rr := range r.Ns {
		if dns.CanonicalName(rr.Header().Name) == cut && rr.Header().Rrtype == dns.TypeNS {
			e.addRecord(rr)

			// A name server out of the delegation is a name of
			// the zone, to ask about as any other: only those
			// below the cut are known from the glue.
			for _, target := range targetsOf(rr) {
				if !dns.IsSubDomain(cut, dns.CanonicalName(target)) {
					targets = append(targets, Name{FQDN: target, Sources: []string{"target"}})
				}
			}
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
	if !errors.Is(err, errBudget) && !errors.Is(err, errOutOfTime) {
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
