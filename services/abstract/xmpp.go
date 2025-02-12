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

package abstract

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/utils"
)

type XMPP struct {
	Records []*dns.SRV `json:"records"`
}

func (s *XMPP) GetNbResources() (max int) {
	return len(s.Records)
}

func (s *XMPP) GenComment(origin string) string {
	dest := map[string][]uint16{}

destloop:
	for _, srv := range s.Records {
		for _, port := range dest[srv.Target] {
			if port == srv.Port {
				continue destloop
			}
		}
		dest[srv.Target] = append(dest[srv.Target], srv.Port)
	}

	var buffer bytes.Buffer
	first := true
	for dn, ports := range dest {
		if !first {
			buffer.WriteString("; ")
		} else {
			first = !first
		}
		buffer.WriteString(strings.TrimSuffix(dn, "."+origin))
		buffer.WriteString(" (")
		firstport := true
		for _, port := range ports {
			if !firstport {
				buffer.WriteString(", ")
			} else {
				firstport = !firstport
			}
			buffer.WriteString(strconv.Itoa(int(port)))
		}
		buffer.WriteString(")")
	}

	return buffer.String()
}

func (s *XMPP) GenRRs(domain string, ttl uint32, origin string) (models.Records, error) {
	return utils.RRstoRCs(s.Records, origin)
}

func xmpp_subanalyze(a *svcs.Analyzer, prefix string, xmppDomains map[string]*XMPP) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Prefix: prefix, Type: dns.TypeSRV}) {
		domain := strings.TrimPrefix(record.NameFQDN, prefix)

		if _, ok := xmppDomains[domain]; !ok {
			xmppDomains[domain] = &XMPP{}
		}

		xmppDomains[domain].Records = append(xmppDomains[domain].Records, record.ToRR().(*dns.SRV))

		a.UseRR(
			record,
			domain,
			xmppDomains[domain],
		)
	}

	return nil
}

func xmpp_analyze(a *svcs.Analyzer) error {
	xmppDomains := map[string]*XMPP{}

	xmpp_subanalyze(a, "_jabber._tcp.", xmppDomains)
	xmpp_subanalyze(a, "_xmpp-client._tcp.", xmppDomains)
	xmpp_subanalyze(a, "_xmpp-server._tcp.", xmppDomains)

	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &XMPP{}
		},
		xmpp_analyze,
		svcs.ServiceInfos{
			Name:        "XMPP IM",
			Description: "Communicate over XMPP with your domain.",
			Family:      svcs.Abstract,
			Categories: []string{
				"service",
			},
			Restrictions: svcs.ServiceRestrictions{
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
