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
	"strconv"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	svc "git.happydns.org/happyDomain/internal/service"
	"git.happydns.org/happyDomain/model"
)

// Kerberos groups the SRV records that advertise a Kerberos realm:
// KDC (TCP & UDP on 88), master KDC, admin server (kadmin) and
// kpasswd. Each slice is optional; the presence of at least one
// `_kerberos._tcp.` or `_kerberos._udp.` record is what advertises the
// realm to clients.
type Kerberos struct {
	KDCTCP     []*dns.SRV `json:"kdc_tcp,omitempty"`
	KDCUDP     []*dns.SRV `json:"kdc_udp,omitempty"`
	Master     []*dns.SRV `json:"master,omitempty"`
	Admin      []*dns.SRV `json:"admin,omitempty"`
	KPasswdTCP []*dns.SRV `json:"kpasswd_tcp,omitempty"`
	KPasswdUDP []*dns.SRV `json:"kpasswd_udp,omitempty"`
}

func (s *Kerberos) all() []*dns.SRV {
	out := make([]*dns.SRV, 0,
		len(s.KDCTCP)+len(s.KDCUDP)+len(s.Master)+len(s.Admin)+len(s.KPasswdTCP)+len(s.KPasswdUDP))
	out = append(out, s.KDCTCP...)
	out = append(out, s.KDCUDP...)
	out = append(out, s.Master...)
	out = append(out, s.Admin...)
	out = append(out, s.KPasswdTCP...)
	out = append(out, s.KPasswdUDP...)
	return out
}

func (s *Kerberos) GetNbResources() int {
	return len(s.all())
}

func (s *Kerberos) GenComment() string {
	dest := map[string][]uint16{}

destloop:
	for _, srv := range s.KDCTCP {
		for _, port := range dest[srv.Target] {
			if port == srv.Port {
				continue destloop
			}
		}
		dest[srv.Target] = append(dest[srv.Target], srv.Port)
	}
	for _, srv := range s.KDCUDP {
		dest[srv.Target] = append(dest[srv.Target], srv.Port)
	}

	var buffer bytes.Buffer
	first := true
	for dn, ports := range dest {
		if !first {
			buffer.WriteString("; ")
		} else {
			first = false
		}
		buffer.WriteString(dn)
		buffer.WriteString(" (")
		firstport := true
		for _, port := range ports {
			if !firstport {
				buffer.WriteString(", ")
			} else {
				firstport = false
			}
			buffer.WriteString(strconv.Itoa(int(port)))
		}
		buffer.WriteString(")")
	}
	return buffer.String()
}

func (s *Kerberos) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	all := s.all()
	rrs := make([]happydns.Record, len(all))
	for i, srv := range all {
		rrs[i] = srv
	}
	return rrs, nil
}

func kerberos_analyze(a *svc.Analyzer) error {
	realms := map[string]*Kerberos{}

	type bucket struct {
		prefix string
		append func(k *Kerberos, s *dns.SRV)
	}
	buckets := []bucket{
		{"_kerberos._tcp.", func(k *Kerberos, s *dns.SRV) { k.KDCTCP = append(k.KDCTCP, s) }},
		{"_kerberos._udp.", func(k *Kerberos, s *dns.SRV) { k.KDCUDP = append(k.KDCUDP, s) }},
		{"_kerberos-master._tcp.", func(k *Kerberos, s *dns.SRV) { k.Master = append(k.Master, s) }},
		{"_kerberos-adm._tcp.", func(k *Kerberos, s *dns.SRV) { k.Admin = append(k.Admin, s) }},
		{"_kpasswd._tcp.", func(k *Kerberos, s *dns.SRV) { k.KPasswdTCP = append(k.KPasswdTCP, s) }},
		{"_kpasswd._udp.", func(k *Kerberos, s *dns.SRV) { k.KPasswdUDP = append(k.KPasswdUDP, s) }},
	}

	for _, b := range buckets {
		for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Prefix: b.prefix, Type: dns.TypeSRV}) {
			domain := strings.TrimPrefix(record.Header().Name, b.prefix)

			if _, ok := realms[domain]; !ok {
				realms[domain] = &Kerberos{}
			}

			srv, ok := record.(*dns.SRV)
			if !ok {
				continue
			}

			rel := helpers.RRRelativeSubdomain(srv, a.GetOrigin(), domain).(*dns.SRV)
			b.append(realms[domain], rel)

			if err := a.UseRR(srv, domain, realms[domain]); err != nil {
				return err
			}
		}
	}
	return nil
}

func init() {
	svc.RegisterService(
		func() happydns.ServiceBody {
			return &Kerberos{}
		},
		kerberos_analyze,
		happydns.ServiceInfos{
			Name:        "Kerberos",
			Description: "Advertise a Kerberos realm (KDC, kadmin, kpasswd) through DNS.",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"service",
				"authentication",
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
