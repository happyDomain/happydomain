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
	"errors"
	"reflect"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

// ServiceAnalyzer is a callback function that inspects DNS records in an
// Analyzer and claims those that belong to a particular service type.
type ServiceAnalyzer func(*Analyzer) error

// recordPool holds DNS records and tracks which ones have been claimed by
// service analyzers.
type recordPool struct {
	zone    []happydns.Record
	claimed []bool
}

// SearchRR returns all unclaimed records that match at least one of the given
// filters. Each record appears at most once in the result.
func (p *recordPool) SearchRR(arrs ...AnalyzerRecordFilter) (rrs []happydns.Record) {
	for i, record := range p.zone {
		if p.claimed[i] {
			continue
		}

		for _, arr := range arrs {
			rhdr := record.Header()
			rdtype := rhdr.Rrtype
			if strings.HasPrefix(rhdr.Name, arr.Prefix) &&
				strings.HasSuffix(rhdr.Name, arr.SubdomainsOf) &&
				(arr.Domain == "" || rhdr.Name == arr.Domain || rhdr.Name == strings.TrimSuffix(arr.Domain, ".")) &&
				(arr.Type == 0 || rdtype == arr.Type) &&
				(arr.Ttl == 0 || rhdr.Ttl == arr.Ttl) &&
				(arr.Contains == "" || strings.Contains(record.String(), arr.Contains)) {
				rrs = append(rrs, record)
				break
			}
		}
	}

	return
}

// markClaimed marks a record as claimed. Returns an error if the record is not
// found or was already claimed.
func (p *recordPool) markClaimed(rr happydns.Record) error {
	for k, record := range p.zone {
		if record == rr {
			if p.claimed[k] {
				return errors.New("Record already claimed.")
			}
			p.claimed[k] = true
			return nil
		}
	}
	return errors.New("Record not found.")
}

// serviceAccumulator collects services discovered during zone analysis,
// keyed by subdomain. It deduplicates services and normalizes domain names.
type serviceAccumulator struct {
	origin     string
	defaultTTL uint32
	services   map[happydns.Subdomain][]*happydns.Service
}

// addService registers a service for the given domain. If the same service
// instance is already registered, it is a no-op.
func (sa *serviceAccumulator) addService(rr happydns.Record, domain string, svc happydns.ServiceBody) error {
	// Remove origin to get a relative domain here
	domain = strings.TrimSuffix(helpers.DomainRelative(domain, sa.origin), ".")
	if domain == "@" {
		domain = ""
	}

	for _, service := range sa.services[happydns.Subdomain(domain)] {
		if service.Service == svc {
			return nil
		}
	}

	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		return err
	}

	var ttl uint32 = 0
	if rr.Header().Ttl != sa.defaultTTL {
		ttl = rr.Header().Ttl
	}

	sa.services[happydns.Subdomain(domain)] = append(sa.services[happydns.Subdomain(domain)], &happydns.Service{
		Service: svc,
		ServiceMeta: happydns.ServiceMeta{
			Id:     id,
			Type:   reflect.Indirect(reflect.ValueOf(svc)).Type().String(),
			Domain: domain,
			Ttl:    ttl,
		},
	})

	return nil
}

// AnalyzerRecordFilter specifies criteria for matching DNS records.
// Zero-value fields are treated as wildcards (match anything).
type AnalyzerRecordFilter struct {
	Prefix       string
	Domain       string
	SubdomainsOf string
	Contains     string
	Type         uint16
	Ttl          uint32
}

// Analyzer holds the state for zone analysis. It is composed of a recordPool
// (DNS records with mark-delete claiming) and a serviceAccumulator (services
// discovered so far, keyed by subdomain).
//
// # Claim protocol
//
// ServiceAnalyzer callbacks are invoked in weight order (lowest first).
// Each analyzer inspects the pool via SearchRR, then claims matching records
// with UseRR. UseRR marks the record as claimed and registers the service.
// Claimed records are invisible to subsequent SearchRR calls.
//
// For SPF, analyzers call ClaimSPFDirective to claim individual directives
// without claiming the whole TXT record. The SPF analyzer (weight 1) runs
// later and filters out claimed directives.
//
// After all analyzers run, unclaimed records are wrapped as Orphan services.
type Analyzer struct {
	recordPool
	serviceAccumulator
	claimedSPFDirectives map[string]map[string]bool // domain -> directive -> claimed
}

// GetOrigin returns the FQDN of the zone being analyzed.
func (a *Analyzer) GetOrigin() string {
	return a.origin
}

// GetDefaultTTL returns the default TTL for the zone being analyzed.
func (a *Analyzer) GetDefaultTTL() uint32 {
	return a.defaultTTL
}

// ClaimSPFDirective marks an SPF directive as claimed by the given service for
// a domain. The directive is not removed from the zone; instead it will be
// filtered out when the SPF service analyzer runs later.
func (a *Analyzer) ClaimSPFDirective(domain string, directive string, svc happydns.ServiceBody) error {
	if a.claimedSPFDirectives == nil {
		a.claimedSPFDirectives = map[string]map[string]bool{}
	}
	if a.claimedSPFDirectives[domain] == nil {
		a.claimedSPFDirectives[domain] = map[string]bool{}
	}
	a.claimedSPFDirectives[domain][directive] = true

	// Ensure the service is registered (addService deduplicates)
	for i, record := range a.zone {
		if !a.claimed[i] && record.Header().Name == domain {
			return a.addService(record, domain, svc)
		}
	}

	// If no record matched, use a synthetic one for the hash
	rr := helpers.NewRecord(domain, "TXT", a.defaultTTL, a.origin)
	return a.addService(rr, domain, svc)
}

// GetClaimedSPFDirectives returns the set of SPF directives claimed by other
// services for the given domain.
func (a *Analyzer) GetClaimedSPFDirectives(domain string) map[string]bool {
	if a.claimedSPFDirectives == nil {
		return nil
	}
	return a.claimedSPFDirectives[domain]
}

// UseRR claims a DNS record, marking it as claimed in the record pool,
// and associates it with the given service. If svc is nil the record is
// simply claimed without registering a service.
func (a *Analyzer) UseRR(rr happydns.Record, domain string, svc happydns.ServiceBody) error {
	if err := a.markClaimed(rr); err != nil {
		return err
	}

	// svc nil, just drop the record from the zone (probably handle another way)
	if svc == nil {
		return nil
	}

	return a.addService(rr, domain, svc)
}

// getMostUsedTTL returns the TTL value that appears most frequently across
// all records in the zone.
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

// AnalyzeZone converts raw DNS records into higher-level services by running
// each registered ServiceAnalyzer in priority order. Records not claimed by
// any analyzer are wrapped as Orphan services.
func AnalyzeZone(origin string, records []happydns.Record) (svcs map[happydns.Subdomain][]*happydns.Service, defaultTTL uint32, err error) {
	// Create a copy of the records as we'll change them in the process
	zone := make([]happydns.Record, len(records))
	for i, record := range records {
		zone[i] = helpers.CopyRecord(record)
	}

	defaultTTL = getMostUsedTTL(records)

	a := Analyzer{
		recordPool: recordPool{
			zone:    zone,
			claimed: make([]bool, len(zone)),
		},
		serviceAccumulator: serviceAccumulator{
			origin:     origin,
			defaultTTL: defaultTTL,
			services:   map[happydns.Subdomain][]*happydns.Service{},
		},
	}

	for i, record := range a.zone {
		// Convert TXT-like records: merge into one string
		switch record.(type) {
		case *dns.TXT:
			a.zone[i] = happydns.NewTXT((record.(*dns.TXT)))
		case *dns.SPF:
			a.zone[i] = happydns.NewSPF((record.(*dns.SPF)))
		}
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

	// Consider unclaimed records as Orphan
	for i, record := range a.zone {
		if a.claimed[i] {
			continue
		}
		// Skip DNSSEC records
		if helpers.IsDNSSECType(record.Header().Rrtype) {
			continue
		}
		if record.Header().Name == "__dnssec."+origin && record.Header().Rrtype == dns.TypeTXT {
			continue
		}

		domain := record.Header().Name

		a.addService(record, domain, &Orphan{helpers.RRRelativeSubdomain(record, a.GetOrigin(), domain)})
	}

	svcs = a.services

	return
}
