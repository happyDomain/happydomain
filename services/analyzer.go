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

package svcs

import (
	"crypto/sha1"
	"errors"
	"io"
	"reflect"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
)

type ServiceAnalyzer func(*Analyzer) error

type Analyzer struct {
	origin     string
	zone       []happydns.Record
	services   map[string][]*happydns.Service
	defaultTTL uint32
}

func (a *Analyzer) GetOrigin() string {
	return a.origin
}

type AnalyzerRecordFilter struct {
	Prefix       string
	Domain       string
	SubdomainsOf string
	Contains     string
	Type         uint16
	Ttl          uint32
}

func (a *Analyzer) SearchRR(arrs ...AnalyzerRecordFilter) (rrs []happydns.Record) {
	for _, record := range a.zone {
		for _, arr := range arrs {
			rhdr := record.Header()
			rdtype := rhdr.Rrtype
			if strings.HasPrefix(rhdr.Name, arr.Prefix) &&
				strings.HasSuffix(rhdr.Name, arr.SubdomainsOf) &&
				(arr.Domain == "" || rhdr.Name == strings.TrimSuffix(arr.Domain, ".")) &&
				(arr.Type == 0 || rdtype == arr.Type) &&
				(arr.Ttl == 0 || rhdr.Ttl == arr.Ttl) &&
				(arr.Contains == "" || strings.Contains(record.String(), arr.Contains)) {
				rrs = append(rrs, record)
			}
		}
	}

	return
}

func (a *Analyzer) UseRR(rr happydns.Record, domain string, svc happydns.ServiceBody) error {
	found := false
	for k, record := range a.zone {
		if record == rr {
			found = true
			a.zone[k] = a.zone[len(a.zone)-1]
			a.zone = a.zone[:len(a.zone)-1]
		}
	}

	if !found {
		return errors.New("Record not found.")
	}

	// svc nil, just drop the record from the zone (probably handle another way)
	if svc == nil {
		return nil
	}

	// Remove origin to get an relative domain here
	domain = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(domain, "."), strings.TrimSuffix(a.origin, ".")), ".")

	for _, service := range a.services[domain] {
		if service.Service == svc {
			service.Comment = svc.GenComment(a.origin)
			service.NbResources = svc.GetNbResources()
			return nil
		}
	}

	hash := sha1.New()
	io.WriteString(hash, rr.String())

	var ttl uint32 = 0
	if rr.Header().Ttl != a.defaultTTL {
		ttl = rr.Header().Ttl
	}

	a.services[domain] = append(a.services[domain], &happydns.Service{
		Service: svc,
		ServiceMeta: happydns.ServiceMeta{
			Id:          hash.Sum(nil),
			Type:        reflect.Indirect(reflect.ValueOf(svc)).Type().String(),
			Domain:      domain,
			Ttl:         ttl,
			Comment:     svc.GenComment(a.origin),
			NbResources: svc.GetNbResources(),
		},
	})

	return nil
}

func getMostUsedTTL(zone []happydns.Record) uint32 {
	ttls := map[uint32]int{}
	for _, rr := range zone {
		ttls[rr.Header().Ttl] += 1
	}

	var max uint32 = 0
	for k, v := range ttls {
		if w, ok := ttls[max]; !ok || v > w {
			max = k
		}
	}

	return max
}

func AnalyzeZone(origin string, zone []happydns.Record) (svcs map[string][]*happydns.Service, defaultTTL uint32, err error) {
	defaultTTL = getMostUsedTTL(zone)

	a := Analyzer{
		origin:     origin,
		zone:       zone,
		services:   map[string][]*happydns.Service{},
		defaultTTL: defaultTTL,
	}

	// Find services between all registered ones
	for _, service := range OrderedServices() {
		if service.Analyzer == nil {
			continue
		}

		if err = service.Analyzer(&a); err != nil {
			return
		}
	}

	svcs = a.services

	// Consider records not used by services as Orphan
	for _, record := range a.zone {
		// Skip DNSSEC records
		if utils.IsDNSSECType(record.Header().Rrtype) {
			continue
		}
		if record.Header().Name == "__dnssec."+origin && record.Header().Rrtype == dns.TypeTXT {
			continue
		}

		// Special treatment for TXT-like records
		switch record.(type) {
		case *dns.TXT:
			record = happydns.NewTXT((record.(*dns.TXT)))
		case *dns.SPF:
			record = happydns.NewSPF((record.(*dns.SPF)))
		}

		domain := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(record.Header().Name, "."), strings.TrimSuffix(a.origin, ".")), ".")

		hash := sha1.New()
		io.WriteString(hash, record.String())

		orphan := &Orphan{dns.TypeToString[record.Header().Rrtype], record.String()}
		svcs[domain] = append(svcs[domain], &happydns.Service{
			Service: orphan,
			ServiceMeta: happydns.ServiceMeta{
				Id:          hash.Sum(nil),
				Type:        reflect.Indirect(reflect.ValueOf(orphan)).Type().String(),
				Domain:      domain,
				Ttl:         record.Header().Ttl,
				NbResources: 1,
				Comment:     orphan.GenComment(a.origin),
			},
		})
	}

	return
}
