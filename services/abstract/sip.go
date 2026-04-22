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

package abstract

import (
	"bytes"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	svc "git.happydns.org/happyDomain/internal/service"
	"git.happydns.org/happyDomain/model"
)

// SIP groups together the SRV records that describe a SIP/VoIP deployment
// for a domain (_sip._udp, _sip._tcp, _sips._tcp), per RFC 3263.
type SIP struct {
	Records []*dns.SRV `json:"records"`
}

// sipPrefixes are the SRV name prefixes that collectively describe a SIP
// deployment. The order also drives the display order in GenComment.
var sipPrefixes = []string{
	"_sips._tcp.",
	"_sip._tcp.",
	"_sip._udp.",
}

func (s *SIP) GetNbResources() int {
	return len(s.Records)
}

func (s *SIP) GenComment() string {
	type entry struct {
		target string
		ports  []uint16
		protos map[string]bool
	}
	byTarget := map[string]*entry{}
	order := []string{}

	protoOf := func(name string) string {
		switch {
		case strings.HasPrefix(name, "_sips._tcp."):
			return "tls"
		case strings.HasPrefix(name, "_sip._tcp."):
			return "tcp"
		case strings.HasPrefix(name, "_sip._udp."):
			return "udp"
		}
		return ""
	}

	for _, srv := range s.Records {
		e, ok := byTarget[srv.Target]
		if !ok {
			e = &entry{target: srv.Target, protos: map[string]bool{}}
			byTarget[srv.Target] = e
			order = append(order, srv.Target)
		}
		if p := protoOf(srv.Hdr.Name); p != "" {
			e.protos[p] = true
		}
		if !slices.Contains(e.ports, srv.Port) {
			e.ports = append(e.ports, srv.Port)
		}
	}

	var buf bytes.Buffer
	for i, tgt := range order {
		if i > 0 {
			buf.WriteString("; ")
		}
		e := byTarget[tgt]
		buf.WriteString(strings.TrimSuffix(tgt, "."))
		buf.WriteString(":")
		for j, p := range e.ports {
			if j > 0 {
				buf.WriteString(",")
			}
			buf.WriteString(strconv.Itoa(int(p)))
		}
		if len(e.protos) > 0 {
			protos := make([]string, 0, len(e.protos))
			for p := range e.protos {
				protos = append(protos, p)
			}
			sort.Strings(protos)
			buf.WriteString(" (")
			buf.WriteString(strings.Join(protos, "/"))
			buf.WriteString(")")
		}
	}
	return buf.String()
}

func (s *SIP) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rrs := make([]happydns.Record, len(s.Records))
	for i, srv := range s.Records {
		rrs[i] = srv
	}
	return rrs, nil
}

func sip_analyze(a *svc.Analyzer) error {
	sipDomains := map[string]*SIP{}

	for _, prefix := range sipPrefixes {
		for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Prefix: prefix, Type: dns.TypeSRV}) {
			domain := strings.TrimPrefix(record.Header().Name, prefix)

			srv, ok := record.(*dns.SRV)
			if !ok {
				continue
			}

			if _, exists := sipDomains[domain]; !exists {
				sipDomains[domain] = &SIP{}
			}

			sipDomains[domain].Records = append(
				sipDomains[domain].Records,
				helpers.RRRelativeSubdomain(srv, a.GetOrigin(), domain).(*dns.SRV),
			)

			if err := a.UseRR(srv, domain, sipDomains[domain]); err != nil {
				return err
			}
		}
	}
	return nil
}

func init() {
	svc.RegisterService(
		func() happydns.ServiceBody {
			return &SIP{}
		},
		sip_analyze,
		happydns.ServiceInfos{
			Name:        "SIP / VoIP",
			Description: "Expose SIP/VoIP endpoints for your domain (voice, video, messaging).",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"service",
				"voip",
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				Single:    true,
				NeedTypes: []uint16{
					dns.TypeSRV,
				},
			},
		},
		1,
	)
}
