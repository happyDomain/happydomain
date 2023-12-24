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

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/utils"
)

type TXT struct {
	Content string `json:"content" happydomain:"label=Content,description=Your text to publish in the zone"`
}

func (ss *TXT) GetNbResources() int {
	return 1
}

func (ss *TXT) GenComment(origin string) string {
	return ss.Content
}

func (ss *TXT) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	rr := utils.NewRecordConfig(domain, "TXT", ttl, origin)
	rr.SetTargetTXT(ss.Content)
	rrs = append(rrs, rr)
	return
}

func txt_analyze(a *Analyzer) (err error) {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeTXT}) {
		// Skip DNSSEC record added by dnscontrol
		if strings.HasPrefix(record.Name, "__dnssec") {
			continue
		}

		if record.Type == "TXT" {
			err = a.UseRR(
				record,
				record.NameFQDN,
				&TXT{Content: record.GetTargetTXTJoined()},
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
			return &TXT{}
		},
		txt_analyze,
		ServiceInfos{
			Name:        "Text Record",
			Description: "Publishes a text string in your zone.",
			RecordTypes: []uint16{
				dns.TypeTXT,
			},
			Restrictions: ServiceRestrictions{
				NeedTypes: []uint16{
					dns.TypeTXT,
				},
			},
		},
		100,
	)
}
