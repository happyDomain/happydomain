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
	"fmt"
	"regexp"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

var (
	SRV_DOMAIN = regexp.MustCompile(`^_([^.]+)\._(tcp|udp)(?:\.(.*))?$`)
)

type UnknownSRV struct {
	Records []*dns.SRV `json:"srv"`
}

func (s *UnknownSRV) GetNbResources() int {
	return len(s.Records)
}

func (s *UnknownSRV) GenComment() string {
	if len(s.Records) == 0 {
		return ""
	}

	subdomains := SRV_DOMAIN.FindStringSubmatch(s.Records[0].Hdr.Name)
	return fmt.Sprintf("%s (%s)", subdomains[1], subdomains[2])
}

func (s *UnknownSRV) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rrs := make([]happydns.Record, len(s.Records))
	for i, srv := range s.Records {
		rrs[i] = srv
	}
	return rrs, nil
}

func srv_analyze(a *Analyzer) error {
	srvDomains := map[string]map[string]*UnknownSRV{}

	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeSRV}) {
		subdomains := SRV_DOMAIN.FindStringSubmatch(record.Header().Name)
		if len(subdomains) < 4 {
			continue
		}

		svc := subdomains[1] + "." + subdomains[2]
		domain := subdomains[3]

		if _, ok := srvDomains[domain]; !ok {
			srvDomains[domain] = map[string]*UnknownSRV{}
		}

		if _, ok := srvDomains[domain][svc]; !ok {
			srvDomains[domain][svc] = &UnknownSRV{}
		}

		srvDomains[domain][svc].Records = append(srvDomains[domain][svc].Records, helpers.RRRelativeSubdomain(record, a.GetOrigin(), domain).(*dns.SRV))

		a.UseRR(
			record,
			subdomains[3],
			srvDomains[domain][svc],
		)
	}
	return nil
}

func init() {
	RegisterService(
		func() happydns.ServiceBody {
			return &UnknownSRV{}
		},
		srv_analyze,
		happydns.ServiceInfos{
			Name:        "Service Record",
			Description: "Indicates to dedicated software the existance of the given service in the domain.",
			Categories: []string{
				"service",
			},
			RecordTypes: []uint16{
				dns.TypeSRV,
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeSRV,
				},
			},
		},
		99999,
	)
}
