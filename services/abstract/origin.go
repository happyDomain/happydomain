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
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/services/common"
)

type NSOnlyOrigin struct {
	NameServers []string `json:"ns" happydomain:"label=Zone's Name Servers"`
}

func (s *NSOnlyOrigin) GetNbResources() int {
	return len(s.NameServers)
}

func (s *NSOnlyOrigin) GenComment(origin string) string {
	return fmt.Sprintf("%d NS", len(s.NameServers))
}

func (s *NSOnlyOrigin) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rrs := make([]happydns.Record, len(s.NameServers))
	for i, r := range s.NameServers {
		ns := utils.NewRecord(utils.DomainJoin(domain), "NS", ttl, origin)
		ns.(*dns.NS).Ns = utils.DomainFQDN(r, origin)
		rrs[i] = ns
	}
	return rrs, nil
}

type Origin struct {
	Ns          string          `json:"mname" happydomain:"label=Name Server,placeholder=ns0,required,description=The domain name of the name server that was the original or primary source of data for this zone."`
	Mbox        string          `json:"rname" happydomain:"label=Contact Email,placeholder=dnsmaster,required,description=A <domain-name> which specifies the mailbox of the person responsible for this zone."`
	Serial      uint32          `json:"serial" happydomain:"label=Zone Serial,required,description=The unsigned 32 bit version number of the original copy of the zone.  Zone transfers preserve this value.  This value wraps and should be compared using sequence space arithmetic."`
	Refresh     common.Duration `json:"refresh" happydomain:"label=Slave Refresh Time,required,description=The time interval before the zone should be refreshed by name servers other than the primary."`
	Retry       common.Duration `json:"retry" happydomain:"label=Retry Interval on failed refresh,required,description=The time interval that should elapse before a failed refresh should be retried by a slave name server."`
	Expire      common.Duration `json:"expire" happydomain:"label=Authoritative Expiry,required,description=Time value that specifies the upper limit on the time interval that can elapse before the zone is no longer authoritative."`
	Negttl      common.Duration `json:"nxttl" happydomain:"label=Negative Caching Time,required,description=Maximal time a resolver should cache a negative authoritative answer (such as NXDOMAIN ...)."`
	NameServers []string        `json:"ns" happydomain:"label=Zone's Name Servers"`
}

func (s *Origin) GetNbResources() int {
	if s.Ns == "" {
		return len(s.NameServers)
	} else {
		return len(s.NameServers) + 1
	}
}

func (s *Origin) GenComment(origin string) string {
	if s.Ns == "" {
		return fmt.Sprintf("%d NS", len(s.NameServers))
	}

	ns := ""
	if s.NameServers != nil {
		ns = fmt.Sprintf(" + %d NS", len(s.NameServers))
	}

	return fmt.Sprintf("%s %s %d"+ns, strings.TrimSuffix(s.Ns, "."+origin), strings.TrimSuffix(s.Mbox, "."+origin), s.Serial)
}

func (s *Origin) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rrs := make([]happydns.Record, len(s.NameServers))
	for i, r := range s.NameServers {
		ns := utils.NewRecord(utils.DomainJoin(domain), "NS", ttl, origin)
		ns.(*dns.NS).Ns = utils.DomainFQDN(r, origin)
		rrs[i] = ns
	}

	if s.Ns != "" {
		rr := utils.NewRecord(domain, "SOA", ttl, origin)
		rr.(*dns.SOA).Ns = utils.DomainFQDN(s.Ns, origin)
		rr.(*dns.SOA).Mbox = utils.DomainFQDN(s.Mbox, origin)
		rr.(*dns.SOA).Serial = s.Serial
		rr.(*dns.SOA).Refresh = uint32(s.Refresh.Seconds())
		rr.(*dns.SOA).Retry = uint32(s.Retry.Seconds())
		rr.(*dns.SOA).Expire = uint32(s.Expire.Seconds())
		rr.(*dns.SOA).Minttl = uint32(s.Negttl.Seconds())

		rrs = append(rrs, rr)
	}

	return rrs, nil
}

func origin_analyze(a *svcs.Analyzer) error {
	hasSOA := false

	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeSOA}) {
		if soa, ok := record.(*dns.SOA); ok {
			hasSOA = true

			// Make record relative
			soa.Ns = utils.DomainRelative(soa.Ns, a.GetOrigin())
			soa.Mbox = utils.DomainRelative(soa.Mbox, a.GetOrigin())

			origin := &Origin{
				Ns:      soa.Ns,
				Mbox:    soa.Mbox,
				Serial:  soa.Serial,
				Refresh: common.Duration(time.Duration(soa.Refresh) * time.Second),
				Retry:   common.Duration(time.Duration(soa.Retry) * time.Second),
				Expire:  common.Duration(time.Duration(soa.Expire) * time.Second),
				Negttl:  common.Duration(time.Duration(soa.Minttl) * time.Second),
			}

			a.UseRR(
				record,
				record.Header().Name,
				origin,
			)

			for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeNS, Domain: record.Header().Name}) {
				if ns, ok := record.(*dns.NS); ok {
					// Make record relative
					ns.Ns = utils.DomainRelative(ns.Ns, a.GetOrigin())

					origin.NameServers = append(origin.NameServers, ns.Ns)
					a.UseRR(
						record,
						record.Header().Name,
						origin,
					)
				}
			}
		}
	}

	if !hasSOA {
		origin := &NSOnlyOrigin{}

		for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeNS, Domain: a.GetOrigin()}) {
			if ns, ok := record.(*dns.NS); ok {
				// Make record relative
				ns.Ns = utils.DomainRelative(ns.Ns, a.GetOrigin())

				origin.NameServers = append(origin.NameServers, ns.Ns)
				a.UseRR(
					record,
					record.Header().Name,
					origin,
				)
			}
		}
	}

	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.ServiceBody {
			return &Origin{}
		},
		origin_analyze,
		happydns.ServiceInfos{
			Name:        "Origin",
			Description: "This is the root of your domain.",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"domain name",
			},
			RecordTypes: []uint16{
				dns.TypeSOA,
				dns.TypeNS,
			},
			Restrictions: happydns.ServiceRestrictions{
				RootOnly: true,
				Single:   true,
				NeedTypes: []uint16{
					dns.TypeSOA,
				},
			},
		},
		0,
	)
	svcs.RegisterService(
		func() happydns.ServiceBody {
			return &NSOnlyOrigin{}
		},
		nil,
		happydns.ServiceInfos{
			Name:        "Origin",
			Description: "This is the root of your domain.",
			Family:      happydns.SERVICE_FAMILY_HIDDEN,
			Categories: []string{
				"domain name",
			},
			RecordTypes: []uint16{
				dns.TypeNS,
			},
			Restrictions: happydns.ServiceRestrictions{
				RootOnly: true,
				Single:   true,
				NeedTypes: []uint16{
					dns.TypeNS,
				},
			},
		},
		0,
	)
}
