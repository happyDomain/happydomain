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
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/utils"
)

type Analyzer struct {
	origin     string
	zone       models.Records
	services   map[string][]*happydns.ServiceCombined
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

func (a *Analyzer) SearchRR(arrs ...AnalyzerRecordFilter) (rrs models.Records) {
	for _, record := range a.zone {
		for _, arr := range arrs {
			if rdtype, ok := dns.StringToType[record.Type]; strings.HasPrefix(record.NameFQDN, arr.Prefix) &&
				strings.HasSuffix(record.NameFQDN, arr.SubdomainsOf) &&
				(arr.Domain == "" || record.NameFQDN == strings.TrimSuffix(arr.Domain, ".")) &&
				(arr.Type == 0 || (ok && rdtype == arr.Type)) &&
				(arr.Ttl == 0 || record.TTL == arr.Ttl) &&
				(arr.Contains == "" || strings.Contains(fmt.Sprintf("%s. %d IN %s %s", record.NameFQDN, record.TTL, record.Type, record.String()), arr.Contains)) {
				rrs = append(rrs, record)
			}
		}
	}

	return
}

func (a *Analyzer) UseRR(rr *models.RecordConfig, domain string, svc happydns.Service) error {
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
	if rr.TTL != a.defaultTTL {
		ttl = rr.TTL
	}

	a.services[domain] = append(a.services[domain], &happydns.ServiceCombined{
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

func getMostUsedTTL(zone models.Records) uint32 {
	ttls := map[uint32]int{}
	for _, rr := range zone {
		ttls[rr.TTL] += 1
	}

	var max uint32 = 0
	for k, v := range ttls {
		if w, ok := ttls[max]; !ok || v > w {
			max = k
		}
	}

	return max
}

func AnalyzeZone(origin string, zone models.Records) (svcs map[string][]*happydns.ServiceCombined, defaultTTL uint32, err error) {
	defaultTTL = getMostUsedTTL(zone)

	a := Analyzer{
		origin:     origin,
		zone:       zone,
		services:   map[string][]*happydns.ServiceCombined{},
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
		if rdtype, ok := dns.StringToType[record.Type]; ok && utils.IsDNSSECType(rdtype) {
			continue
		}
		if record.NameFQDN == "__dnssec."+origin && record.Type == "TXT" {
			continue
		}

		domain := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(record.NameFQDN, "."), strings.TrimSuffix(a.origin, ".")), ".")

		hash := sha1.New()
		io.WriteString(hash, record.String())

		orphan := &Orphan{record.Type, record.String()}
		svcs[domain] = append(svcs[domain], &happydns.ServiceCombined{
			Service: orphan,
			ServiceMeta: happydns.ServiceMeta{
				Id:          hash.Sum(nil),
				Type:        reflect.Indirect(reflect.ValueOf(orphan)).Type().String(),
				Domain:      domain,
				Ttl:         record.TTL,
				NbResources: 1,
				Comment:     orphan.GenComment(a.origin),
			},
		})
	}

	return
}
