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

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

type Orphan struct {
	Type string
	RR   string
}

func (s *Orphan) GetNbResources() int {
	return 1
}

func (s *Orphan) GenComment() string {
	return fmt.Sprintf("%s %s", s.Type, s.RR)
}

func (s *Orphan) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	if _, ok := dns.StringToType[s.Type]; ok {
		rr, err := dns.NewRR(fmt.Sprintf("%s %d IN %s %s", helpers.DomainJoin(domain, origin), ttl, s.Type, s.RR))
		if err != nil {
			return nil, err
		}

		return []happydns.Record{rr}, nil
	}

	return nil, fmt.Errorf("unknown record type")
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
