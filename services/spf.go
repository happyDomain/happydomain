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
	"fmt"
	"strings"

	"github.com/miekg/dns"

	"github.com/StackExchange/dnscontrol/v4/pkg/spflib"

	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
)

type SPF struct {
	Version    uint     `json:"version" happydomain:"label=Version,placeholder=1,required,description=The version of SPF to use.,default=1,hidden"`
	Directives []string `json:"directives" happydomain:"label=Directives,placeholder=ip4:203.0.113.12"`
}

func (s *SPF) GetNbResources() int {
	return 1
}

func (s *SPF) GenComment(origin string) string {
	return fmt.Sprintf("%d directives", len(s.Directives))
}

func (s *SPF) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	rr := utils.NewRecord(domain, "TXT", ttl, origin)
	rr.(*dns.TXT).Txt = []string{s.String()}

	rrs = append(rrs, rr)

	return
}

func (t *SPF) Analyze(txt string) error {
	_, err := spflib.Parse(txt, nil)
	if err != nil {
		return err
	}

	t.Version = 1

	fields := strings.Fields(txt)

	// Avoid doublon
	for _, directive := range fields[1:] {
		exists := false
		for _, known := range t.Directives {
			if known == directive {
				exists = true
				break
			}
		}

		if !exists {
			t.Directives = append(t.Directives, directive)
		}
	}

	return nil
}

func spf_analyze(a *Analyzer) (err error) {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeTXT, Contains: "v=spf1"}) {
		service := &SPF{}

		err = service.Analyze(strings.Join(record.(*dns.TXT).Txt, ""))
		if err != nil {
			return
		}

		err = a.UseRR(record, record.Header().Name, service)
		if err != nil {
			return
		}
	}

	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeSPF, Contains: "v=spf1"}) {
		service := &SPF{}

		err = service.Analyze(strings.Join(record.(*dns.SPF).Txt, ""))
		if err != nil {
			return
		}

		err = a.UseRR(record, record.Header().Name, service)
		if err != nil {
			return
		}
	}

	return
}

func (s *SPF) String() string {
	directives := append([]string{fmt.Sprintf("v=spf%d", s.Version)}, s.Directives...)
	return strings.Join(directives, " ")
}

func init() {
	RegisterService(
		func() happydns.ServiceBody {
			return &SPF{}
		},
		spf_analyze,
		happydns.ServiceInfos{
			Name:        "SPF",
			Description: "Sender Policy Framework, to authenticate domain name on email sending.",
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeTXT,
				dns.TypeSPF,
			},
			Restrictions: happydns.ServiceRestrictions{
				NeedTypes: []uint16{
					dns.TypeTXT,
				},
			},
		},
		1,
	)
}
