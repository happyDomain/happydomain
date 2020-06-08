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
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
)

type TLSA struct {
	Proto        string `json:"proto" happydns:"label=Protocol,description=Protocol used to establish the connection.,choices=tcp;udp"`
	Port         uint16 `json:"port" happydns:"label=Service Port,description=Port number where people will establish the connection."`
	CertUsage    uint8  `json:"certusage"`
	Selector     uint8  `json:"selector"`
	MatchingType uint8  `json:"matchingtype"`
	Certificate  []byte `json:"certificate"`
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

func (ss *TLSAs) GenRRs(domain string, ttl uint32) (rrs []dns.RR) {
	for _, s := range ss.TLSA {
		if len(s.Certificate) > 0 {
			rrs = append(rrs, &dns.TLSA{
				Hdr: dns.RR_Header{
					Name:   fmt.Sprintf("_%d._%s.%d", s.Port, s.Proto, domain),
					Rrtype: dns.TypeTLSA,
					Class:  dns.ClassINET,
					Ttl:    ttl,
				},
				Usage:        s.CertUsage,
				Selector:     s.Selector,
				MatchingType: s.MatchingType,
				Certificate:  hex.EncodeToString(s.Certificate),
			})
		}
	}
	return
}

var (
	TLSA_DOMAIN = regexp.MustCompile(`^_([0-9]+)\._(tcp|udp)\.(.*)$`)
)

func tlsa_analyze(a *Analyzer) (err error) {
	pool := map[string]*TLSAs{}
	for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeTLSA}) {
		subdomains := TLSA_DOMAIN.FindStringSubmatch(record.Header().Name)
		if tlsa, ok := record.(*dns.TLSA); len(subdomains) == 4 && ok {
			var port uint64
			port, err = strconv.ParseUint(subdomains[1], 10, 16)

			var cert []byte
			cert, err = hex.DecodeString(tlsa.Certificate)
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
					CertUsage:    tlsa.Usage,
					Selector:     tlsa.Selector,
					MatchingType: tlsa.MatchingType,
					Certificate:  cert,
				},
			)

			err = a.useRR(
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
			Description: "",
			Categories: []string{
				"tls",
			},
		},
		100,
	)
}
