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
	"reflect"
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

	if before, ok := strings.CutSuffix(subdomain, origin); ok {
		subdomain = strings.TrimSuffix(before, ".")
	}

	if subdomain == "" {
		return "@"
	}

	return subdomain
}

func NewRecord(domain string, rrtype string, ttl uint32, origin string) happydns.Record {
	rdtype, ok := dns.StringToType[rrtype]
	if !ok {
		// The pseudo-types DNSControl has to keep seeing as such are
		// deliberately absent from dns.StringToType, but they do have a
		// constructor in dns.TypeToRR.
		pt, known := happydns.PseudoTypeByName(rrtype)
		if !known {
			return nil
		}

		rdtype = pt.Rrtype
	}

	rr := dns.TypeToRR[rdtype]()

	// Fill in the header.
	rr.Header().Name = DomainFQDN(domain, origin)
	rr.Header().Rrtype = rdtype
	rr.Header().Class = dns.ClassINET
	rr.Header().Ttl = ttl

	return rr
}

func ParseRecord(input, origin string) (happydns.Record, error) {
	zp := dns.NewZoneParser(strings.NewReader(input), origin, "")
	zp.SetDefaultTTL(0)
	zp.SetIncludeAllowed(false)

	rr, _ := zp.Next()
	if err := zp.Err(); err != nil {
		return nil, err
	}

	if rr.Header().Rrtype == dns.TypeTXT {
		return happydns.NewTXT(rr.(*dns.TXT)), nil
	} else if rr.Header().Rrtype == dns.TypeSPF {
		return happydns.NewSPF(rr.(*dns.SPF)), nil
	} else if rr.Header().Rrtype == dns.TypeSVCB {
		return happydns.NewSVCB(rr.(*dns.SVCB)), nil
	} else if rr.Header().Rrtype == dns.TypeHTTPS {
		return happydns.NewHTTPS(rr.(*dns.HTTPS)), nil
	}

	return rr, nil
}

// RRRelative strips the end of the given RR if it is relative to origin.
func RRRelative(rr happydns.Record, origin string) happydns.Record {
	if !strings.HasSuffix(origin, ".") {
		origin += "."
	}

	// Make header relative
	if before, ok := strings.CutSuffix(rr.Header().Name, origin); ok {
		rr.Header().Name = strings.TrimSuffix(before, ".")
	}

	return RDataRelative(rr, origin)
}

// RRRelativeSubdomain strips the end of the given RR if it is relative to origin.
func RRRelativeSubdomain(rr happydns.Record, origin, subdomain string) happydns.Record {
	if !strings.HasSuffix(origin, ".") {
		origin += "."
	}

	subdomain = DomainFQDN(subdomain, origin)

	// Make header relative
	if before, ok := strings.CutSuffix(rr.Header().Name, subdomain); ok {
		rr.Header().Name = strings.TrimSuffix(before, ".")
	}

	return RDataRelative(rr, origin)
}

// rdataFields returns the struct holding the rdata of rr, or false when rr is
// not one whose rdata is described by miekg/dns tags.
func rdataFields(rr happydns.Record) (reflect.Value, bool) {
	switch rr.(type) {
	case *dns.PrivateRR:
		// Its rdata is opaque to miekg/dns.
		return reflect.Value{}, false
	case *dns.TSIG, *dns.TKEY:
		// Their Algorithm is the name of an algorithm, not of the zone.
		return reflect.Value{}, false
	}

	v := reflect.ValueOf(rr)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	v = v.Elem()
	if v.NumField() == 1 && v.Type().Field(0).Anonymous {
		// This is a redirection to another type, like HTTPS to SVCB
		v = v.Field(0)
	}

	return v, true
}

// isDomainNameField tells whether miekg/dns tags the field as a domain name.
func isDomainNameField(f reflect.StructField) bool {
	tag := f.Tag.Get("dns")
	return tag == "domain-name" || tag == "cdomain-name"
}

// rdataNames replaces each domain name held in the rdata of rr by f(name).
func rdataNames(rr happydns.Record, f func(string) string) {
	if prr, ok := rr.(*dns.PrivateRR); ok {
		// Pseudo-types (ALIAS, R53_ALIAS, ...): their rdata is opaque to
		// miekg/dns, so ask it for its target itself. The ones whose target is
		// not a domain name (AZURE_ALIAS carries a resource id) are left alone.
		if rdata, ok := prr.Data.(happydns.TargetRdata); ok && rdata.TargetIsHostname() {
			rdata.SetTarget(f(rdata.GetTarget()))
		}
		return
	}

	v, ok := rdataFields(rr)
	if !ok {
		return
	}

	for i := 0; i < v.NumField(); i++ {
		if !isDomainNameField(v.Type().Field(i)) {
			continue
		}

		switch field := v.Field(i); field.Kind() {
		case reflect.String:
			field.SetString(f(field.String()))
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				for j := 0; j < field.Len(); j++ {
					field.Index(j).SetString(f(field.Index(j).String()))
				}
			}
		}
	}
}

// RDataNameFields lists the rdata fields of rr holding a single domain name,
// the ones RDataRelative makes relative to the zone. Lists of names, like the
// rendezvous servers of HIP, are made relative too but are not listed.
func RDataNameFields(rr happydns.Record) (names []string) {
	v, ok := rdataFields(rr)
	if !ok {
		return nil
	}

	for i := 0; i < v.NumField(); i++ {
		if f := v.Type().Field(i); isDomainNameField(f) && f.Type.Kind() == reflect.String {
			names = append(names, f.Name)
		}
	}

	return
}

// RDataRelative strips the end of the given RData if it is relative to origin.
func RDataRelative(rr happydns.Record, origin string) happydns.Record {
	rdataNames(rr, func(name string) string {
		return DomainRelative(name, origin)
	})

	return rr
}

// RRAbsolute transforms relative RR to absolute RR in place.
func RRAbsolute(rr happydns.Record, origin string) happydns.Record {
	if origin != "" {
		if !strings.HasSuffix(origin, ".") {
			origin += "."
		}

		// Make header absolute
		rr.Header().Name = DomainFQDN(rr.Header().Name, origin)
	}

	rdataNames(rr, func(name string) string {
		return DomainFQDN(name, origin)
	})

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
