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
	"bytes"
	"regexp"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

var (
	TLSA_DOMAIN = regexp.MustCompile(`^_([0-9]+)\._(tcp|udp)\.(.*)$`)
)

type TLSAs struct {
	Records []*dns.TLSA `json:"tlsa"`
}

func (ss *TLSAs) GetNbResources() int {
	return len(ss.Records)
}

func (ss *TLSAs) GenComment() string {
	mapProto := map[string][]string{}
protoloop:
	for _, tlsa := range ss.Records {
		subdomains := TLSA_DOMAIN.FindStringSubmatch(tlsa.Header().Name)
		if len(subdomains) > 2 {
			for _, port := range mapProto[subdomains[2]] {
				if port == subdomains[1] {
					continue protoloop
				}
			}
			mapProto[subdomains[2]] = append(mapProto[subdomains[2]], subdomains[1])
		}
	}

	var buffer bytes.Buffer
	first := true
	for proto, ports := range mapProto {
		if !first {
			buffer.WriteString(" - ")
		} else {
			first = !first
		}
		buffer.WriteString(proto)
		buffer.WriteString(" (")
		firstport := true
		for _, port := range ports {
			if !firstport {
				buffer.WriteString(", ")
			} else {
				firstport = !firstport
			}
			buffer.WriteString(port)
		}
		buffer.WriteString(")")
	}

	return buffer.String()
}

func (ss *TLSAs) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rrs := make([]happydns.Record, len(ss.Records))
	for i, r := range ss.Records {
		rrs[i] = r
	}
	return rrs, nil
}

type TLSAFields struct {
	Proto        string              `json:"proto" happydomain:"label=Protocol,description=Protocol used to establish the connection.,choices=tcp;udp"`
	Port         uint16              `json:"port" happydomain:"label=Service Port,description=Port number where people will establish the connection."`
	CertUsage    uint8               `json:"certusage"`
	Selector     uint8               `json:"selector"`
	MatchingType uint8               `json:"matchingtype"`
	Certificate  happydns.HexaString `json:"certificate"`
}

func tlsa_analyze(a *Analyzer) (err error) {
	pool := map[string]*TLSAs{}
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeTLSA}) {
		subdomains := TLSA_DOMAIN.FindStringSubmatch(record.Header().Name)
		if _, ok := record.(*dns.TLSA); ok && len(subdomains) == 4 {
			if _, ok := pool[subdomains[3]]; !ok {
				pool[subdomains[3]] = &TLSAs{}
			}

			pool[subdomains[3]].Records = append(
				pool[subdomains[3]].Records,
				helpers.RRRelativeSubdomain(record, a.GetOrigin(), subdomains[3]).(*dns.TLSA),
			)

			err = a.UseRR(
				record,
				subdomains[3],
				pool[subdomains[3]],
			)
			if err != nil {
				return
			}
		}
	}

	return nil
}

func init() {
	RegisterService(
		func() happydns.ServiceBody {
			return &TLSAs{}
		},
		tlsa_analyze,
		happydns.ServiceInfos{
			Name:        "TLSA records",
			Description: "Publish TLS certificates exposed by your services.",
			Categories: []string{
				"security",
			},
			RecordTypes: []uint16{
				dns.TypeTLSA,
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				Single:    true,
				NeedTypes: []uint16{
					dns.TypeTLSA,
				},
			},
		},
		100,
	)
}
