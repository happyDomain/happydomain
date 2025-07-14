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
	"log"
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

	// Make header relative
	if strings.HasSuffix(rr.Header().Name, origin) {
		rr.Header().Name = strings.TrimSuffix(strings.TrimSuffix(rr.Header().Name, origin), ".")
	}

	// Make RData relative
	if ns, ok := rr.(*dns.NS); ok {
		ns.Ns = DomainRelative(ns.Ns, origin)
	} else if md, ok := rr.(*dns.MD); ok {
		md.Md = DomainRelative(md.Md, origin)
	} else if mf, ok := rr.(*dns.MF); ok {
		mf.Mf = DomainRelative(mf.Mf, origin)
	} else if cname, ok := rr.(*dns.CNAME); ok {
		cname.Target = DomainRelative(cname.Target, origin)
	} else if soa, ok := rr.(*dns.SOA); ok {
		soa.Ns = DomainRelative(soa.Ns, origin)
		soa.Mbox = DomainRelative(soa.Mbox, origin)
	} else if mb, ok := rr.(*dns.MB); ok {
		mb.Mb = DomainRelative(mb.Mb, origin)
	} else if mg, ok := rr.(*dns.MG); ok {
		mg.Mg = DomainRelative(mg.Mg, origin)
	} else if mr, ok := rr.(*dns.MR); ok {
		mr.Mr = DomainRelative(mr.Mr, origin)
	} else if ptr, ok := rr.(*dns.PTR); ok {
		ptr.Ptr = DomainRelative(ptr.Ptr, origin)
	} else if minfo, ok := rr.(*dns.MINFO); ok {
		minfo.Rmail = DomainRelative(minfo.Rmail, origin)
		minfo.Email = DomainRelative(minfo.Email, origin)
	} else if mx, ok := rr.(*dns.MX); ok {
		mx.Mx = DomainRelative(mx.Mx, origin)
	} else if rp, ok := rr.(*dns.RP); ok {
		rp.Mbox = DomainRelative(rp.Mbox, origin)
		rp.Txt = DomainRelative(rp.Txt, origin)
	} else if afsdb, ok := rr.(*dns.AFSDB); ok {
		afsdb.Hostname = DomainRelative(afsdb.Hostname, origin)
	} else if rt, ok := rr.(*dns.RT); ok {
		rt.Host = DomainRelative(rt.Host, origin)
	} else if ptr, ok := rr.(*dns.NSAPPTR); ok {
		ptr.Ptr = DomainRelative(ptr.Ptr, origin)
	} else if sig, ok := rr.(*dns.SIG); ok {
		sig.SignerName = DomainRelative(sig.SignerName, origin)
	} else if px, ok := rr.(*dns.PX); ok {
		px.Map822 = DomainRelative(px.Map822, origin)
		px.Mapx400 = DomainRelative(px.Mapx400, origin)
	} else if nxt, ok := rr.(*dns.NXT); ok {
		nxt.NextDomain = DomainRelative(nxt.NextDomain, origin)
	} else if srv, ok := rr.(*dns.SRV); ok {
		srv.Target = DomainRelative(srv.Target, origin)
	} else if ptr, ok := rr.(*dns.NAPTR); ok {
		ptr.Replacement = DomainRelative(ptr.Replacement, origin)
	} else if kx, ok := rr.(*dns.KX); ok {
		kx.Exchanger = DomainRelative(kx.Exchanger, origin)
	} else if dname, ok := rr.(*dns.DNAME); ok {
		dname.Target = DomainRelative(dname.Target, origin)
	} else if sig, ok := rr.(*dns.RRSIG); ok {
		sig.SignerName = DomainRelative(sig.SignerName, origin)
	} else if nxt, ok := rr.(*dns.NSEC); ok {
		nxt.NextDomain = DomainRelative(nxt.NextDomain, origin)
	} else if hip, ok := rr.(*dns.HIP); ok {
		for i := range hip.RendezvousServers {
			hip.RendezvousServers[i] = DomainRelative(hip.RendezvousServers[i], origin)
		}
	} else if talink, ok := rr.(*dns.TALINK); ok {
		talink.PreviousName = DomainRelative(talink.PreviousName, origin)
		talink.NextName = DomainRelative(talink.NextName, origin)
	} else if lp, ok := rr.(*dns.LP); ok {
		lp.Fqdn = DomainRelative(lp.Fqdn, origin)
	}

	return rr
}

func CopyRecord(rr happydns.Record) happydns.Record {
	if dnsrr, ok := rr.(dns.RR); ok {
		return dns.Copy(dnsrr)
	}

	if copiablerr, ok := rr.(happydns.CopiableRecord); ok {
		return copiablerr.Copy()
	}

	log.Fatalf("Type %T doesn't implement Copy method", rr)
	return nil
}
