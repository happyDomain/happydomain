// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package orchestrator

import (
	"time"

	"github.com/miekg/dns"

	svc "git.happydns.org/happyDomain/internal/service"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

// SetPropagationTimes stamps each service in newServices with a PropagatedAt
// time based on whether the service changed compared to the provider state.
// It reuses the same matching technique as ReassociateMetadata (subdomain +
// type + ServiceRDataHash).
//
// For changed/updated services: PropagatedAt = publishTime + old service TTL.
// For new services (additions): PropagatedAt = publishTime + SOA minimum TTL
// (negative cache duration), falling back to defaultTTL.
func SetPropagationTimes(
	newServices map[happydns.Subdomain][]*happydns.Service,
	providerRecords []happydns.Record,
	origin string,
	defaultTTL uint32,
	publishTime time.Time,
) {
	// Find SOA minimum TTL for negative cache duration (used for additions).
	negativeCacheTTL := defaultTTL
	for _, rr := range providerRecords {
		if rr.Header().Rrtype == dns.TypeSOA {
			if soa, ok := rr.(*dns.SOA); ok {
				negativeCacheTTL = soa.Minttl
			}
			break
		}
	}

	// Analyze provider records into old services for comparison.
	oldServices, oldDefaultTTL, err := svc.AnalyzeZone(origin, providerRecords)
	if err != nil {
		return
	}

	for dn, newSvcs := range newServices {
		oldSvcs := oldServices[dn]

		// Group old services by type.
		oldByType := map[string][]*happydns.Service{}
		for _, s := range oldSvcs {
			oldByType[s.Type] = append(oldByType[s.Type], s)
		}

		for _, newSvc := range newSvcs {
			candidates := oldByType[newSvc.Type]

			if len(candidates) == 0 {
				// New service (addition): use SOA negative cache TTL.
				propagatedAt := publishTime.Add(time.Duration(negativeCacheTTL) * time.Second)
				newSvc.PropagatedAt = &propagatedAt
				continue
			}

			newHash := zoneUC.ServiceRDataHash(newSvc, origin, defaultTTL)

			if len(candidates) == 1 {
				oldSvc := candidates[0]
				oldHash := zoneUC.ServiceRDataHash(oldSvc, origin, oldDefaultTTL)
				if newHash != oldHash {
					// Service changed: use old service TTL.
					oldTTL := oldDefaultTTL
					if oldSvc.Ttl != 0 {
						oldTTL = oldSvc.Ttl
					}
					propagatedAt := publishTime.Add(time.Duration(oldTTL) * time.Second)
					newSvc.PropagatedAt = &propagatedAt
				}
				continue
			}

			// Multiple candidates: try to find exact RDATA match.
			matched := false
			for _, oldSvc := range candidates {
				if zoneUC.ServiceRDataHash(oldSvc, origin, oldDefaultTTL) == newHash {
					// Exact match: service unchanged, don't touch PropagatedAt.
					matched = true
					break
				}
			}
			if !matched {
				// No exact match: service was modified. Use the max TTL
				// across all candidates of the same type as a conservative
				// upper bound.
				var maxOldTTL uint32
				for _, oldSvc := range candidates {
					ttl := oldDefaultTTL
					if oldSvc.Ttl != 0 {
						ttl = oldSvc.Ttl
					}
					if ttl > maxOldTTL {
						maxOldTTL = ttl
					}
				}
				propagatedAt := publishTime.Add(time.Duration(maxOldTTL) * time.Second)
				newSvc.PropagatedAt = &propagatedAt
			}
		}
	}
}
