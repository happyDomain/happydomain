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
	"errors"
	"fmt"
	"log"
	"net"
	"net/netip"
	"slices"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
)

// errBudget is returned once Options.MaxQueries questions have been asked.
var errBudget = errors.New("query budget exhausted")

// findAuthoritative asks the recursive resolver for the name servers of the
// domain and returns those of their addresses the guard allows.
func (e *explorer) findAuthoritative(ctx context.Context) ([]string, error) {
	r, err := helpers.Question(ctx, e.udp, e.opts.Resolvers, e.domain, dns.TypeNS, helpers.QueryOptions{Recursive: true})
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
			r, err := helpers.Question(ctx, e.udp, e.opts.Resolvers, nsName, qtype, helpers.QueryOptions{Recursive: true})
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

		r, err := helpers.Question(ctx, e.udp, []string{server}, name, qtype, helpers.QueryOptions{})
		if err == nil && r.Rcode != dns.RcodeSuccess && r.Rcode != dns.RcodeNameError {
			err = fmt.Errorf("%s answered %s", server, dns.RcodeToString[r.Rcode])
		}
		// A server no longer serving the zone may still answer, without
		// authority: its empty answers and denials would hide the records
		// of the name.
		if err == nil && !r.Authoritative && e.delegationIn(r) == "" {
			err = fmt.Errorf("%s is not authoritative for %s", server, name)
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
// in a row first.
func (e *explorer) serversByHealth() []string {
	servers := slices.Clone(e.servers)
	slices.SortStableFunc(servers, func(a, b string) int {
		return cmp.Compare(e.failures[a].Load(), e.failures[b].Load())
	})
	return servers
}
