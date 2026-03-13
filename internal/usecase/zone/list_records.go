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

package zone

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
	svcs "git.happydns.org/happyDomain/services"
)

type ListRecordsUsecase struct {
	serviceListRecordsUC *service.ListRecordsUsecase
}

func NewListRecordsUsecase(serviceListRecordsUC *service.ListRecordsUsecase) *ListRecordsUsecase {
	return &ListRecordsUsecase{
		serviceListRecordsUC: serviceListRecordsUC,
	}
}

func (uc *ListRecordsUsecase) ToZoneFile(domain *happydns.Domain, zone *happydns.Zone) (string, error) {
	records, err := uc.List(domain, zone)
	if err != nil {
		return "", happydns.InternalError{
			Err: fmt.Errorf("unable to retrieve records for old zone: %w", err),
		}
	}

	var ret string

	for _, rr := range records {
		ret += rr.String() + "\n"
	}

	return ret, nil
}

func (uc *ListRecordsUsecase) List(domain *happydns.Domain, zone *happydns.Zone) (rrs []happydns.Record, err error) {
	var svc_rrs []happydns.Record

	// Collect SPF contributions keyed by absolute domain name.
	type spfContrib struct {
		directives [][]string
		policies   []string
	}
	spfContribs := map[string]*spfContrib{}

	for _, services := range zone.Services {
		for _, svc := range services {
			svc_rrs, err = uc.serviceListRecordsUC.List(svc, domain.DomainName, zone.DefaultTTL)
			if err != nil {
				return
			}

			// If the service is an SPF contributor, collect its directives
			// and filter out any SPF TXT records it emits (they'll be
			// replaced by the merged record).
			if contributor, ok := svc.Service.(happydns.SPFContributor); ok {
				directives := contributor.GetSPFDirectives()
				policy := contributor.GetSPFAllPolicy()

				// Compute the absolute domain for this service.
				absDomain := svc.Domain
				if domain.DomainName != "" {
					if absDomain == "" {
						absDomain = domain.DomainName
					} else {
						absDomain = absDomain + "." + domain.DomainName
					}
				}
				if !strings.HasSuffix(absDomain, ".") {
					absDomain += "."
				}

				if spfContribs[absDomain] == nil {
					spfContribs[absDomain] = &spfContrib{}
				}
				spfContribs[absDomain].directives = append(spfContribs[absDomain].directives, directives)
				if policy != "" {
					spfContribs[absDomain].policies = append(spfContribs[absDomain].policies, policy)
				}

				// Drop SPF TXT records from this service's output.
				filtered := svc_rrs[:0]
				for _, rr := range svc_rrs {
					if txt, ok := rr.(*happydns.TXT); ok && strings.HasPrefix(txt.Txt, "v=spf1") {
						continue
					}
					filtered = append(filtered, rr)
				}
				svc_rrs = filtered
			}

			rrs = append(rrs, svc_rrs...)
		}

		// Ensure SOA is the first record
		for i, rr := range rrs {
			if rr.Header().Rrtype == dns.TypeSOA {
				rrs[0], rrs[i] = rrs[i], rrs[0]
				break
			}
		}
	}

	// Emit one merged SPF TXT record per domain that has SPF contributions.
	for absDomain, contrib := range spfContribs {
		merged := svcs.MergeSPFDirectives(contrib.directives...)
		policy := svcs.ResolveSPFAllPolicy(contrib.policies)
		merged = append(merged, policy)

		spfFields := svcs.SPFFields{
			Version:    1,
			Directives: merged,
		}

		rr := &happydns.TXT{
			Hdr: dns.RR_Header{
				Name:   absDomain,
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    zone.DefaultTTL,
			},
			Txt: spfFields.String(),
		}
		rrs = append(rrs, rr)
	}

	return
}
