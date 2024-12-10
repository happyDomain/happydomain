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
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/utils"
)

type TLSA struct {
	Proto        string              `json:"proto" happydomain:"label=Protocol,description=Protocol used to establish the connection.,choices=tcp;udp"`
	Port         uint16              `json:"port" happydomain:"label=Service Port,description=Port number where people will establish the connection."`
	CertUsage    uint8               `json:"certusage"`
	Selector     uint8               `json:"selector"`
	MatchingType uint8               `json:"matchingtype"`
	Certificate  happydns.HexaString `json:"certificate"`
}

type TLSAs struct {
	TLSA []*TLSA `json:"tlsa,omitempty"`
}

func (ss *TLSAs) GetNbResources() int {
	return len(ss.TLSA)
}

func (ss *TLSAs) GenComment(origin string) string {
	mapProto := map[string][]uint16{}
protoloop:
	for _, tlsa := range ss.TLSA {
		for _, port := range mapProto[tlsa.Proto] {
			if port == tlsa.Port {
				continue protoloop
			}
		}
		mapProto[tlsa.Proto] = append(mapProto[tlsa.Proto], tlsa.Port)
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
			buffer.WriteString(strconv.Itoa(int(port)))
		}
		buffer.WriteString(")")
	}

	return buffer.String()
}

func (ss *TLSAs) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	for _, s := range ss.TLSA {
		if len(s.Certificate) > 0 {
			rr := utils.NewRecordConfig(utils.DomainJoin(fmt.Sprintf("_%d._%s", s.Port, s.Proto), domain), "TLSA", ttl, origin)
			rr.TlsaUsage = s.CertUsage
			rr.TlsaSelector = s.Selector
			rr.TlsaMatchingType = s.MatchingType
			rr.SetTarget(hex.EncodeToString(s.Certificate))
			rrs = append(rrs, rr)
		}
	}
	return
}

var (
	TLSA_DOMAIN = regexp.MustCompile(`^_([0-9]+)\._(tcp|udp)\.(.*)$`)
)

func tlsa_analyze(a *Analyzer) (err error) {
	pool := map[string]*TLSAs{}
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeTLSA}) {
		subdomains := TLSA_DOMAIN.FindStringSubmatch(record.NameFQDN)
		if record.Type == "TLSA" && len(subdomains) == 4 {
			var port uint64
			port, err = strconv.ParseUint(subdomains[1], 10, 16)

			var cert []byte
			cert, err = hex.DecodeString(record.GetTargetField())
			if err != nil {
				return
			}

			if _, ok := pool[subdomains[3]]; !ok {
				pool[subdomains[3]] = &TLSAs{}
			}

			pool[subdomains[3]].TLSA = append(
				pool[subdomains[3]].TLSA,
				&TLSA{
					Port:         uint16(port),
					Proto:        subdomains[2],
					CertUsage:    record.TlsaUsage,
					Selector:     record.TlsaSelector,
					MatchingType: record.TlsaMatchingType,
					Certificate:  cert,
				},
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
		func() happydns.Service {
			return &TLSAs{}
		},
		tlsa_analyze,
		ServiceInfos{
			Name:        "TLSA records",
			Description: "Publish TLS certificates exposed by your services.",
			Categories: []string{
				"security",
			},
			RecordTypes: []uint16{
				dns.TypeTLSA,
			},
			Restrictions: ServiceRestrictions{
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
