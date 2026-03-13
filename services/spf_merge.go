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

package svcs

import (
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

// spfContrib collects SPF directives and policies for a single domain.
type spfContrib struct {
	directives [][]string
	policies   []string
}

// CollectAndMergeSPF scans zone services for SPFContributor implementations,
// collects their directives per absolute domain, filters SPF TXT records from
// the input records, and appends merged SPF TXT records. The domainName is
// the zone's domain name (e.g. "example.com."), defaultTTL is used for the
// merged records.
func CollectAndMergeSPF(domainName string, zone *happydns.Zone, records []happydns.Record, defaultTTL uint32) []happydns.Record {
	contribs := map[string]*spfContrib{}

	for _, domainSvcs := range zone.Services {
		for _, svc := range domainSvcs {
			contributor, ok := svc.Service.(happydns.SPFContributor)
			if !ok {
				continue
			}

			directives := contributor.GetSPFDirectives()
			policy := contributor.GetSPFAllPolicy()

			// Compute the absolute domain for this service.
			absDomain := svc.Domain
			if domainName != "" {
				if absDomain == "" {
					absDomain = domainName
				} else {
					absDomain = absDomain + "." + domainName
				}
			}
			if !strings.HasSuffix(absDomain, ".") {
				absDomain += "."
			}

			if contribs[absDomain] == nil {
				contribs[absDomain] = &spfContrib{}
			}
			contribs[absDomain].directives = append(contribs[absDomain].directives, directives)
			if policy != "" {
				contribs[absDomain].policies = append(contribs[absDomain].policies, policy)
			}
		}
	}

	if len(contribs) == 0 {
		return records
	}

	// Filter out SPF TXT records emitted by individual services.
	filtered := records[:0]
	for _, rr := range records {
		if txt, ok := rr.(*happydns.TXT); ok && strings.HasPrefix(txt.Txt, "v=spf1") {
			continue
		}
		filtered = append(filtered, rr)
	}
	records = filtered

	// Emit one merged SPF TXT record per domain that has SPF contributions.
	for absDomain, contrib := range contribs {
		merged := MergeSPFDirectives(contrib.directives...)
		policy := ResolveSPFAllPolicy(contrib.policies)
		merged = append(merged, policy)

		spfFields := SPFFields{
			Version:    1,
			Directives: merged,
		}

		rr := &happydns.TXT{
			Hdr: dns.RR_Header{
				Name:   absDomain,
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    defaultTTL,
			},
			Txt: spfFields.String(),
		}
		records = append(records, rr)
	}

	return records
}
