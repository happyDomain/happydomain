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
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

type TXT struct {
	Record *happydns.TXT `json:"txt"`
}

func (ss *TXT) GetNbResources() int {
	return 1
}

func (ss *TXT) GenComment() string {
	return ss.Record.Txt
}

func (ss *TXT) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	return []happydns.Record{ss.Record}, nil
}

func txt_analyze(a *Analyzer) (err error) {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeTXT}) {
		// Skip DNSSEC record added by dnscontrol
		if strings.HasPrefix(record.Header().Name, "__dnssec") {
			continue
		}

		if txt, ok := record.(*happydns.TXT); ok {
			domain := record.Header().Name
			err = a.UseRR(record, domain, &TXT{
				Record: helpers.RRRelative(txt, domain).(*happydns.TXT),
			})
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
			return &TXT{}
		},
		txt_analyze,
		happydns.ServiceInfos{
			Name:        "Text Record",
			Description: "Publishes a text string in your zone.",
			RecordTypes: []uint16{
				dns.TypeTXT,
			},
			Restrictions: happydns.ServiceRestrictions{
				NeedTypes: []uint16{
					dns.TypeTXT,
				},
			},
		},
		100,
	)
}
