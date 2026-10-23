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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"maps"
	"slices"

	"github.com/miekg/dns"
)

// wildcardProbeTypes are asked for at a random name, to learn what a
// wildcard, if any, answers.
var wildcardProbeTypes = []uint16{dns.TypeA, dns.TypeAAAA, dns.TypeMX, dns.TypeTXT}

// wildcardProbeAttempts is how many times each probe of the wildcard is
// asked before giving up.
const wildcardProbeAttempts = 3

// probeWildcard asks about a random name, which only a wildcard can answer
// for, and remembers the answers to tell them apart from real names. It
// returns the targets of the wildcard records, to explore as any other.
func (e *explorer) probeWildcard(ctx context.Context) (targets []Name) {
	label := make([]byte, 8)
	rand.Read(label)
	probe := "hd-probe-" + hex.EncodeToString(label) + "." + e.domain

	for _, qtype := range wildcardProbeTypes {
		// A lost probe would leave the wildcard unknown, and every name it
		// makes up taken for a real one: the probe is asked again.
		var (
			r   *dns.Msg
			err error
		)
		for range wildcardProbeAttempts {
			r, err = e.query(ctx, probe, qtype)
			if err == nil || errors.Is(err, errBudget) || ctx.Err() != nil {
				break
			}
		}
		if err != nil {
			e.fail(err)
			return
		}
		if r.Rcode == dns.RcodeNameError {
			return
		}
		// Below a DNAME, no name exists, a wildcard neither.
		if dname := e.dnameAbove(r, probe); dname != nil {
			e.addRecord(dname)
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
			for _, target := range targetsOf(rr) {
				targets = append(targets, Name{FQDN: target, Sources: []string{"target"}})
			}
		}

		for rrtype, rdata := range rdataByType(answersAt(r, probe)) {
			e.wildcard[rrtype] = rdata
		}
	}
	return
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
