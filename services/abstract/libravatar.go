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

package abstract

import (
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type LibravatarServer struct {
	Records []*dns.SRV `json:"srv"`
}

func (s *LibravatarServer) GetNbResources() int {
	return len(s.Records)
}

func (s *LibravatarServer) GenComment() string {
	if len(s.Records) == 0 {
		return ""
	}

	return s.Records[0].Target
}

func (s *LibravatarServer) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rrs := make([]happydns.Record, len(s.Records))
	for i, r := range s.Records {
		srv := *r
		srv.Target = helpers.DomainFQDN(srv.Target, origin)
		rrs[i] = &srv
	}
	return rrs, nil
}

func libavatar_analyze(a *svcs.Analyzer) error {
	alreadyUsed := map[string]*LibravatarServer{}

	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeSRV, Prefix: "_avatars"}) {
		domain := ""
		if strings.HasPrefix(record.Header().Name, "_avatars._tcp.") {
			domain = strings.TrimPrefix(record.Header().Name, "_avatars._tcp.")
		} else if strings.HasPrefix(record.Header().Name, "_avatars-sec._tcp.") {
			domain = strings.TrimPrefix(record.Header().Name, "_avatars-sec._tcp.")
		} else {
			continue
		}

		if srv, ok := record.(*dns.SRV); ok {
			var rr *LibravatarServer

			// Make record relative
			srv.Target = helpers.DomainRelative(srv.Target, a.GetOrigin())

			if ls, ok := alreadyUsed[srv.Target]; ok {
				rr = ls
				rr.Records = append(rr.Records, helpers.RRRelative(srv, a.GetOrigin()).(*dns.SRV))
			} else {
				rr = &LibravatarServer{
					Records: []*dns.SRV{helpers.RRRelative(srv, a.GetOrigin()).(*dns.SRV)},
				}

				alreadyUsed[srv.Target] = rr
			}

			a.UseRR(record, domain, rr)
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.ServiceBody {
			return &LibravatarServer{}
		},
		libavatar_analyze,
		happydns.ServiceInfos{
			Name:        "Federated Avatar",
			Description: "Declare a libravatar server for this subdomain.",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"service",
			},
		},
		2,
	)
}
