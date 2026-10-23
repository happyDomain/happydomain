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
	"slices"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
)

// delegationIn returns the delegation point r tells of, if any: the one a
// referral sends to, or the apex of a zone below ours denying the name
// with its authority, the same server serving that zone too.
func (e *explorer) delegationIn(r *dns.Msg) string {
	if len(r.Answer) > 0 {
		return ""
	}

	want := uint16(dns.TypeNS)
	if r.Authoritative {
		want = dns.TypeSOA
	}
	for _, rr := range r.Ns {
		if rr.Header().Rrtype != want {
			continue
		}

		owner := dns.CanonicalName(rr.Header().Name)
		if owner != e.domain && dns.IsSubDomain(e.domain, owner) {
			return owner
		}
	}

	return ""
}

// dnameAbove returns the DNAME of the zone r tells name lies below, if any.
func (e *explorer) dnameAbove(r *dns.Msg, name string) dns.RR {
	for _, rr := range r.Answer {
		owner := dns.CanonicalName(rr.Header().Name)
		if rr.Header().Rrtype == dns.TypeDNAME && owner != name && dns.IsSubDomain(owner, name) && dns.IsSubDomain(e.domain, owner) {
			return rr
		}
	}
	return nil
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
// compares two RRsets regardless of their owner, TTL and order. The DNSSEC
// records are left out: a signed zone adds them to every answer.
func rdataByType(rrs []dns.RR) map[uint16]string {
	byType := map[uint16][]string{}
	for _, rr := range rrs {
		rrtype := rr.Header().Rrtype
		if helpers.IsDNSSECType(rrtype) {
			continue
		}
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
