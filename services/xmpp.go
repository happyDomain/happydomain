// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package svcs

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/utils"
)

type XMPP struct {
	Client []*SRV `json:"client,omitempty" happydns:"label=Client Connection"`
	Server []*SRV `json:"server,omitempty" happydns:"label=Server Connection"`
	Jabber []*SRV `json:"jabber,omitempty" happydns:"label=Jabber Connection (legacy)"`
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

func (s *XMPP) GenRRs(domain string, ttl uint32, origin string) (rrs []dns.RR) {
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

func xmpp_subanalyze(a *Analyzer, prefix string, xmppDomains map[string]*XMPP, field string) error {
	for _, record := range a.searchRR(AnalyzerRecordFilter{Prefix: prefix, Type: dns.TypeSRV}) {
		if srv := parseSRV(record); srv != nil {
			domain := strings.TrimPrefix(record.Header().Name, prefix)

			if _, ok := xmppDomains[domain]; !ok {
				xmppDomains[domain] = &XMPP{}
			}

			v := reflect.Indirect(reflect.ValueOf(xmppDomains[domain]))
			v.FieldByName(field).Set(reflect.Append(v.FieldByName(field), reflect.ValueOf(srv)))

			a.useRR(
				record,
				domain,
				xmppDomains[domain],
			)
		}
	}

	return nil
}

func xmpp_analyze(a *Analyzer) error {
	xmppDomains := map[string]*XMPP{}

	xmpp_subanalyze(a, "_jabber._tcp.", xmppDomains, "Jabber")
	xmpp_subanalyze(a, "_xmpp-client._tcp.", xmppDomains, "Client")
	xmpp_subanalyze(a, "_xmpp-server._tcp.", xmppDomains, "Server")

	return nil
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &XMPP{}
		},
		xmpp_analyze,
		ServiceInfos{
			Name:        "XMPP IM",
			Description: "Communicate over XMPP with your domain.",
			Categories: []string{
				"im",
			},
			Restrictions: ServiceRestrictions{
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
