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
	"strconv"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

type MTA_STS struct {
	Version uint   `json:"version" happydomain:"label=Version,placeholder=1,required,description=The version of MTA-STS to use.,default=1,hidden"`
	Id      string `json:"id" happydomain:"label=Policy Identifier,placeholder=,description=A short string used to track policy updates."`
}

func (t *MTA_STS) GetNbResources() int {
	return 1
}

func (t *MTA_STS) GenComment() string {
	return t.Id
}

func (t *MTA_STS) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	rr := helpers.NewRecord(helpers.DomainJoin("_mta-sts", domain), "TXT", ttl, origin)
	rr.(*dns.TXT).Txt = []string{t.String()}

	rrs = append(rrs, rr)
	return
}

func (t *MTA_STS) Analyze(txt string) error {
	fields := strings.Split(txt, ";")

	if len(fields) < 2 {
		return fmt.Errorf("not a valid MTA-STS record: should have a version AND a id, only one field found")
	}
	if len(fields) > 3 || (len(fields) == 3 && fields[2] != "") {
		return fmt.Errorf("not a valid MTA-STS record: should have exactly 2 fields: seen %d", len(fields))
	}

	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}

	if !strings.HasPrefix(fields[0], "v=STSv") {
		return fmt.Errorf("not a valid MTA-STS record: should begin with v=STSv1, seen %q", fields[0])
	}

	version, err := strconv.ParseUint(fields[0][6:], 10, 32)
	if err != nil {
		return fmt.Errorf("not a valid MTA-STS record: bad version number: %w", err)
	}
	t.Version = uint(version)

	if !strings.HasPrefix(fields[1], "id=") {
		return fmt.Errorf("not a valid MTA-STS record: expected id=, found %q", fields[1])
	}

	t.Id = strings.TrimSpace(strings.TrimPrefix(fields[1], "id="))

	return nil
}

func (t *MTA_STS) String() string {
	return fmt.Sprintf("v=STSv%d; id=%s", t.Version, t.Id)
}

func mtasts_analyze(a *Analyzer) (err error) {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_mta-sts."}) {
		// rfc8461: 3.1 records that do not begin with "v=STSv1;" are discarded
		if !strings.HasPrefix(strings.Join(record.(*dns.TXT).Txt, ""), "v=STS") {
			continue
		}

		service := &MTA_STS{}

		err = service.Analyze(strings.Join(record.(*dns.TXT).Txt, ""))
		if err != nil {
			return
		}

		err = a.UseRR(record, strings.TrimPrefix(record.Header().Name, "_mta-sts."), service)
		if err != nil {
			return
		}
	}

	return
}

func init() {
	RegisterService(
		func() happydns.ServiceBody {
			return &MTA_STS{}
		},
		mtasts_analyze,
		happydns.ServiceInfos{
			Name:        "MTA-STS",
			Description: "SMTP MTA Strict Transport Security.",
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeTXT,
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeTXT,
				},
			},
		},
		1,
	)
}
