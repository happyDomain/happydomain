// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

package helpers

import (
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

// SplitN splits a string into N sized string chunks.
// This function is a copy of https://github.com/miekg/dns/blob/master/types.go#L1509
// awaiting its exportation
func SplitN(s string, n int) []string {
	if len(s) < n {
		return []string{s}
	}
	sx := []string{}
	p, i := 0, n
	for {
		if i <= len(s) {
			sx = append(sx, s[p:i])
		} else {
			sx = append(sx, s[p:])
			break

		}
		p, i = p+n, i+n
	}

	return sx
}

// DomainFQDN normalizes the domain by adding the origin if it is relative (not
// ended by .).
func DomainFQDN(subdomain string, origin string) string {
	if len(subdomain) > 0 && subdomain[len(subdomain)-1] == '.' {
		return subdomain
	} else if subdomain == "" || subdomain == "@" {
		return origin
	} else {
		return subdomain + "." + origin
	}
}

// DomainJoin appends each relative domains passed as argument.
func DomainJoin(domains ...string) (ret string) {
	for _, d := range domains {
		if d == "@" {
			break
		} else if d != "" {
			ret += "." + d
		}

		if len(ret) > 0 && ret[len(ret)-1] == '.' {
			break
		}
	}

	if len(ret) >= 1 {
		ret = ret[1:]
	}

	return
}

// DomainRelative strips the end of the given FQDN if it is relative to origin.
func DomainRelative(subdomain string, origin string) string {
	if !strings.HasSuffix(origin, ".") {
		origin += "."
	}

	if strings.HasSuffix(subdomain, origin) {
		subdomain = strings.TrimSuffix(strings.TrimSuffix(subdomain, origin), ".")
	}

	if subdomain == "" {
		return "@"
	}

	return subdomain
}

func NewRecord(domain string, rrtype string, ttl uint32, origin string) happydns.Record {
	rdtype := dns.StringToType[rrtype]

	rr := dns.TypeToRR[rdtype]()

	// Fill in the header.
	rr.Header().Name = DomainFQDN(domain, origin)
	rr.Header().Rrtype = rdtype
	rr.Header().Class = dns.ClassINET
	rr.Header().Ttl = ttl

	return rr
}

// RRRelative strips the end of the given RR if it is relative to origin.
func RRRelative(rr happydns.Record, origin string) happydns.Record {
	if !strings.HasSuffix(origin, ".") {
		origin += "."
	}

	if strings.HasSuffix(rr.Header().Name, origin) {
		rr.Header().Name = strings.TrimSuffix(strings.TrimSuffix(rr.Header().Name, origin), ".")
	}

	return rr
}
