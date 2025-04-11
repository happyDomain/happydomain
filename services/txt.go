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

	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
)

type TXT struct {
	Content string `json:"content" happydomain:"label=Content,description=Your text to publish in the zone"`
}

func (ss *TXT) GetNbResources() int {
	return 1
}

func (ss *TXT) GenComment() string {
	return ss.Content
}

func (ss *TXT) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	rr := utils.NewRecord(domain, "TXT", ttl, origin)
	rr.(*dns.TXT).Txt = []string{ss.Content}
	rrs = append(rrs, rr)
	return
}

func txt_analyze(a *Analyzer) (err error) {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeTXT}) {
		// Skip DNSSEC record added by dnscontrol
		if strings.HasPrefix(record.Header().Name, "__dnssec") {
			continue
		}

		if txt, ok := record.(*dns.TXT); ok {
			err = a.UseRR(
				record,
				record.Header().Name,
				&TXT{Content: strings.Join(txt.Txt, "")},
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
