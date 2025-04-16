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
	"encoding/json"
	"fmt"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

type Orphan struct {
	Record happydns.Record `json:"record"`
}

func (s *Orphan) GetNbResources() int {
	return 1
}

func (s *Orphan) GenComment() string {
	return fmt.Sprintf("%s %s", dns.Type(s.Record.Header().Rrtype).String(), s.Record.String()[len(s.Record.Header().String()):])
}

func (s *Orphan) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	return []happydns.Record{s.Record}, nil
}

func (s *Orphan) UnmarshalJSON(b []byte) error {
	var rrtype struct {
		Record struct{ Hdr dns.RR_Header } `json:"record"`
	}

	err := json.Unmarshal(b, &rrtype)
	if err != nil {
		return err
	}

	var myOrphan struct {
		Record dns.RR `json:"record"`
	}
	if newrr, ok := dns.TypeToRR[rrtype.Record.Hdr.Rrtype]; ok {
		myOrphan.Record = newrr()
	} else {
		return fmt.Errorf("unknwon rr type %d", rrtype.Record.Hdr.Rrtype)
	}

	err = json.Unmarshal(b, &myOrphan)
	if err != nil {
		return err
	}

	s.Record = myOrphan.Record

	return nil
}

func init() {
	RegisterService(
		func() happydns.ServiceBody {
			return &Orphan{}
		},
		nil,
		happydns.ServiceInfos{
			Name:        "Orphan Record",
			Description: "A record not yet handled by happyDomain. Ask us to support it.",
			Categories:  []string{},
		},
		99999999,
	)
}
