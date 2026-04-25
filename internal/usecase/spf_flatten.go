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

package usecase

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

const (
	spfMaxLookups       = 10
	spfMaxVoidLookups   = 2
	spfMaxDepth         = 12
	spfPerLookupTimeout = 2 * time.Second
	spfTotalDeadline    = 10 * time.Second
)

// spfTermKind identifies the lookup-consuming mechanisms tracked by RFC 7208
// §4.6.4.
type spfTermKind int

const (
	spfTermNone spfTermKind = iota
	spfTermInclude
	spfTermRedirect
	spfTermA
	spfTermMX
	spfTermPTR
	spfTermExists
)

func (k spfTermKind) consumesLookup() bool {
	return k != spfTermNone
}

// parsedSPFTerm describes a single SPF directive or modifier term, after
// stripping the qualifier and splitting around ":" / "=" / "/".
type parsedSPFTerm struct {
	raw       string
	kind      spfTermKind
	value     string
	isAll     bool
	mechanism string
}

func parseSPFTerm(raw string) parsedSPFTerm {
	s := raw
	if len(s) > 0 && (s[0] == '+' || s[0] == '-' || s[0] == '~' || s[0] == '?') {
		s = s[1:]
	}

	eqIdx := strings.IndexByte(s, '=')
	colonIdx := strings.IndexByte(s, ':')
	slashIdx := strings.IndexByte(s, '/')

	isModifier := eqIdx != -1 && (colonIdx == -1 || eqIdx < colonIdx) && (slashIdx == -1 || eqIdx < slashIdx)

	name := s
	value := ""
	switch {
	case isModifier:
		name = s[:eqIdx]
		value = s[eqIdx+1:]
	case colonIdx != -1:
		name = s[:colonIdx]
		value = s[colonIdx+1:]
	case slashIdx != -1:
		name = s[:slashIdx]
		value = "" // a/24 — no domain, just the cidr
	}

	name = strings.ToLower(name)
	pt := parsedSPFTerm{raw: raw, value: value, mechanism: name}
	switch {
	case isModifier && name == "redirect":
		pt.kind = spfTermRedirect
	case !isModifier && name == "include":
		pt.kind = spfTermInclude
	case !isModifier && name == "a":
		pt.kind = spfTermA
	case !isModifier && name == "mx":
		pt.kind = spfTermMX
	case !isModifier && name == "exists":
		pt.kind = spfTermExists
	case !isModifier && name == "ptr":
		pt.kind = spfTermPTR
	case !isModifier && name == "all":
		pt.isAll = true
	}
	return pt
}

// flattenContext threads the global limits (lookups, voids, deadline) through
// the recursion.
type flattenContext struct {
	resolver string

	deadline time.Time
	lookups  int
	voids    int

	visited map[string]struct{}
}

func (fc *flattenContext) overBudget() bool {
	return fc.lookups > spfMaxLookups
}

func (fc *flattenContext) overVoidBudget() bool {
	return fc.voids > spfMaxVoidLookups
}

func (fc *flattenContext) deadlineExceeded() bool {
	return time.Now().After(fc.deadline)
}

// queryTXT issues a TXT query and returns the raw payload joined into a
// single string per record. The second return value is true when the lookup
// "voids" — i.e. NXDOMAIN, NoData, or no SPF record found.
func queryTXT(client dns.Client, resolver, name string) ([]string, bool, error) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(name), dns.TypeTXT)
	m.RecursionDesired = true
	m.SetEdns0(4096, true)

	r, _, err := client.Exchange(m, resolver)
	if err != nil {
		return nil, false, err
	}
	if r == nil {
		return nil, true, nil
	}
	switch r.Rcode {
	case dns.RcodeNameError:
		return nil, true, nil
	case dns.RcodeSuccess:
		// fallthrough
	default:
		return nil, false, fmt.Errorf("resolver returned %s", dns.RcodeToString[r.Rcode])
	}

	var out []string
	for _, ans := range r.Answer {
		if txt, ok := ans.(*dns.TXT); ok {
			out = append(out, strings.Join(txt.Txt, ""))
		}
	}
	return out, len(out) == 0, nil
}

func pickSPFRecord(records []string) string {
	for _, rec := range records {
		if strings.HasPrefix(strings.ToLower(rec), "v=spf1") {
			return rec
		}
	}
	return ""
}

func (fc *flattenContext) flatten(client dns.Client, domain, record, mechanism string, depth int) *SPFNodeState {
	node := &SPFNodeState{
		Domain:      domain,
		Mechanism:   mechanism,
		LookupsHere: 0,
	}

	if depth > spfMaxDepth {
		node.Error = "depth"
		return node
	}
	key := strings.ToLower(domain)
	if _, seen := fc.visited[key]; seen {
		node.Error = "loop"
		return node
	}
	fc.visited[key] = struct{}{}
	defer delete(fc.visited, key)

	// Resolve the record when the caller did not pre-supply it.
	if record == "" {
		if fc.deadlineExceeded() {
			node.Error = "timeout"
			return node
		}
		records, void, err := queryTXT(client, fc.resolver, domain)
		if err != nil {
			node.Error = "resolver"
			return node
		}
		record = pickSPFRecord(records)
		if record == "" {
			fc.voids++
			if void {
				node.Error = "nxdomain"
			} else {
				node.Error = "no-spf"
			}
			return node
		}
	}
	node.Record = record

	fields := strings.Fields(record)
	if len(fields) == 0 {
		node.Error = "syntax"
		return node
	}

	for _, raw := range fields[1:] {
		term := parseSPFTerm(raw)
		if !term.kind.consumesLookup() {
			continue
		}
		fc.lookups++
		node.LookupsHere = 1

		if fc.overBudget() {
			node.Children = append(node.Children, &SPFNodeState{
				Domain:      term.value,
				Mechanism:   raw,
				LookupsHere: 1,
				Error:       "budget",
			})
			return node
		}
		if fc.deadlineExceeded() {
			node.Children = append(node.Children, &SPFNodeState{
				Domain:      term.value,
				Mechanism:   raw,
				LookupsHere: 1,
				Error:       "timeout",
			})
			return node
		}

		switch term.kind {
		case spfTermInclude, spfTermRedirect:
			target := term.value
			if target == "" {
				node.Children = append(node.Children, &SPFNodeState{
					Domain:      "",
					Mechanism:   raw,
					LookupsHere: 1,
					Error:       "syntax",
				})
				continue
			}
			child := fc.flatten(client, target, "", raw, depth+1)
			node.Children = append(node.Children, child)
		default:
			// a, mx, ptr, exists: count as a single lookup. We don't recurse
			// into the secondary records (mx hostnames, etc.) — they would
			// only count toward the budget if they triggered void responses
			// or contained nested SPF lookups, which is out of scope here.
			child := &SPFNodeState{
				Domain:      term.value,
				Mechanism:   raw,
				LookupsHere: 1,
			}
			node.Children = append(node.Children, child)
		}
	}

	return node
}

// SPFNodeState mirrors happydns.SPFNode while we build it; we then convert
// it to the public type for serialization. Keeping a private intermediate
// avoids leaking pointers to mutable internal state.
type SPFNodeState struct {
	Domain      string
	Mechanism   string
	Record      string
	LookupsHere int
	Error       string
	Children    []*SPFNodeState
}

func (s *SPFNodeState) export() *happydns.SPFNode {
	if s == nil {
		return nil
	}
	out := &happydns.SPFNode{
		Domain:      s.Domain,
		Mechanism:   s.Mechanism,
		Record:      s.Record,
		LookupsHere: s.LookupsHere,
		Error:       s.Error,
	}
	for _, c := range s.Children {
		out.Children = append(out.Children, c.export())
	}
	return out
}

func (ru *resolverUsecase) FlattenSPF(req happydns.SPFFlattenRequest) (*happydns.SPFFlattenResponse, error) {
	if req.Domain == "" {
		return nil, happydns.ValidationError{Msg: "domain is required"}
	}

	resolver, err := ru.pickResolver(req.Resolver, req.Custom)
	if err != nil {
		return nil, err
	}

	client := dns.Client{Timeout: spfPerLookupTimeout}

	fc := &flattenContext{
		resolver: resolver,
		deadline: time.Now().Add(spfTotalDeadline),
		visited:  map[string]struct{}{},
	}

	root := fc.flatten(client, req.Domain, req.Record, "root", 0)

	resp := &happydns.SPFFlattenResponse{
		Record:       root.Record,
		LookupCount:  fc.lookups,
		VoidLookups:  fc.voids,
		Exceeded:     fc.overBudget(),
		VoidExceeded: fc.overVoidBudget(),
		Truncated:    fc.overBudget() || fc.deadlineExceeded(),
		Tree:         root.export(),
	}
	return resp, nil
}

// pickResolver mirrors the logic used by ResolveQuestion to pick a resolver
// address out of {"local", "custom", explicit}. Errors are wrapped with the
// usecase's error types.
func (ru *resolverUsecase) pickResolver(name, custom string) (string, error) {
	resolver := name
	switch resolver {
	case "":
		// Default to a public, well-known resolver when the caller did not
		// specify one. Use Cloudflare's 1.1.1.1 as a sane default.
		resolver = "1.1.1.1"
	case "custom":
		if custom == "" {
			return "", happydns.ValidationError{Msg: "custom resolver address required"}
		}
		resolver = custom
	case "local":
		cConf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
		if err != nil {
			return "", happydns.InternalError{
				Err:         fmt.Errorf("unable to load ClientConfigFromFile: %s", err.Error()),
				UserMessage: "Sorry, we are currently unable to perform the request. Please try again later.",
			}
		}
		if len(cConf.Servers) == 0 {
			return "", happydns.InternalError{Err: errors.New("no resolver in /etc/resolv.conf")}
		}
		resolver = cConf.Servers[rand.Intn(len(cConf.Servers))]
	}

	if strings.Count(resolver, ":") > 0 && resolver[0] != '[' {
		resolver = "[" + resolver + "]"
	}
	return resolver + ":53", nil
}
