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

package zone

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"git.happydns.org/happyDomain/model"
)

// ServiceRDataHash computes a SHA-256 hex digest of the RDATA of all records
// produced by the service. This is used to match services across re-analyses
// when multiple services of the same type exist under the same subdomain.
func ServiceRDataHash(svc *happydns.Service, origin string, defaultTTL uint32) string {
	ttl := defaultTTL
	if svc.Ttl != 0 {
		ttl = svc.Ttl
	}

	records, err := svc.Service.GetRecords(svc.Domain, ttl, origin)
	if err != nil {
		return ""
	}

	var parts []string
	for _, rr := range records {
		// Get the full string and strip the header to keep only RDATA
		full := rr.String()
		hdr := rr.Header().String()
		rdata := strings.TrimPrefix(full, hdr)
		parts = append(parts, rdata)
	}
	sort.Strings(parts)

	h := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return fmt.Sprintf("%x", h)
}

// transferMetadata copies metadata fields from an old service to a new one
// and calls EnrichFromPrevious if the new service body implements MetadataEnricher.
func transferMetadata(oldSvc, newSvc *happydns.Service, origin string, defaultTTL uint32) {
	newSvc.Id = oldSvc.Id
	newSvc.UserComment = oldSvc.UserComment
	newSvc.OwnerId = oldSvc.OwnerId
	newSvc.Aliases = oldSvc.Aliases

	if oldSvc.Ttl != 0 {
		serviceTtl := oldSvc.Ttl
		newSvc.Ttl = serviceTtl

		// Adjust record TTLs in the new service body.
		// GetRecords returns pointers to the records stored in the service body,
		// so mutating them here mutates the stored records.
		records, err := newSvc.Service.GetRecords(newSvc.Domain, serviceTtl, origin)
		if err == nil {
			for _, rr := range records {
				hdrTtl := rr.Header().Ttl
				if hdrTtl == 0 {
					// Records with TTL 0 mean "use zone default". After transferring
					// a custom service TTL, 0 would mean "use service TTL" instead.
					// Set explicitly to defaultTTL to preserve original meaning.
					rr.Header().Ttl = defaultTTL
				} else if hdrTtl == serviceTtl {
					// Records matching the service TTL should inherit from it
					// rather than storing a redundant absolute value.
					rr.Header().Ttl = 0
				}
			}
		}
	}

	if enricher, ok := newSvc.Service.(happydns.MetadataEnricher); ok {
		enricher.EnrichFromPrevious(oldSvc.Service)
	}
}

// ReassociateMetadata transfers metadata from old services to new services
// after a zone re-analysis. It matches services by type and subdomain,
// using RDATA hashing to disambiguate when multiple services of the same
// type exist under the same subdomain.
func ReassociateMetadata(oldServices, newServices map[happydns.Subdomain][]*happydns.Service, origin string, defaultTTL uint32) {
	for dn, newSvcs := range newServices {
		oldSvcs := oldServices[dn]
		if len(oldSvcs) == 0 {
			continue
		}

		// Group old services by type
		oldByType := map[string][]*happydns.Service{}
		for _, svc := range oldSvcs {
			oldByType[svc.Type] = append(oldByType[svc.Type], svc)
		}

		for _, newSvc := range newSvcs {
			candidates := oldByType[newSvc.Type]
			if len(candidates) == 0 {
				continue
			}

			if len(candidates) == 1 {
				// Unambiguous match
				transferMetadata(candidates[0], newSvc, origin, defaultTTL)
			} else {
				// Ambiguous: match by RDATA hash
				newHash := ServiceRDataHash(newSvc, origin, defaultTTL)
				for _, oldSvc := range candidates {
					if ServiceRDataHash(oldSvc, origin, defaultTTL) == newHash {
						transferMetadata(oldSvc, newSvc, origin, defaultTTL)
						break
					}
				}
			}
		}
	}
}
