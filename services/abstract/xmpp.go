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
	"reflect"
	"strconv"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/utils"
)

type XMPP struct {
	Client []*svcs.SRV `json:"client,omitempty" happydomain:"label=Client Connection"`
	Server []*svcs.SRV `json:"server,omitempty" happydomain:"label=Server Connection"`
	Jabber []*svcs.SRV `json:"jabber,omitempty" happydomain:"label=Jabber Connection (legacy)"`
}

func (s *XMPP) GetNbResources() (max int) {
	for _, i := range []int{len(s.Client), len(s.Server), len(s.Jabber)} {
		if i > max {
			max = i
		}
	}
	return
}

func (s *XMPP) GenComment(origin string) string {
	dest := map[string][]uint16{}

destloop:
	for _, srv := range append(append(s.Client, s.Server...), s.Jabber...) {
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

func (s *XMPP) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	for _, jabber := range s.Client {
		rrs = append(rrs, jabber.GenRRs(utils.DomainJoin("_jabber._tcp", domain), ttl, origin)...)
	}

	for _, client := range s.Client {
		rrs = append(rrs, client.GenRRs(utils.DomainJoin("_xmpp-client._tcp", domain), ttl, origin)...)
	}

	for _, server := range s.Server {
		rrs = append(rrs, server.GenRRs(utils.DomainJoin("_xmpp-server._tcp", domain), ttl, origin)...)
	}

	return
}

func xmpp_subanalyze(a *svcs.Analyzer, prefix string, xmppDomains map[string]*XMPP, field string) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Prefix: prefix, Type: dns.TypeSRV}) {
		if srv := svcs.ParseSRV(record); srv != nil {
			domain := strings.TrimPrefix(record.NameFQDN, prefix)

			if _, ok := xmppDomains[domain]; !ok {
				xmppDomains[domain] = &XMPP{}
			}

			v := reflect.Indirect(reflect.ValueOf(xmppDomains[domain]))
			v.FieldByName(field).Set(reflect.Append(v.FieldByName(field), reflect.ValueOf(srv)))

			a.UseRR(
				record,
				domain,
				xmppDomains[domain],
			)
		}
	}

	return nil
}

func xmpp_analyze(a *svcs.Analyzer) error {
	xmppDomains := map[string]*XMPP{}

	xmpp_subanalyze(a, "_jabber._tcp.", xmppDomains, "Jabber")
	xmpp_subanalyze(a, "_xmpp-client._tcp.", xmppDomains, "Client")
	xmpp_subanalyze(a, "_xmpp-server._tcp.", xmppDomains, "Server")

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
				"im",
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
