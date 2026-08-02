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

package svcs

import (
	"encoding/json"
	"fmt"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	svc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	"git.happydns.org/happyDomain/model"
)

// AliasRecordTypes are the record types pointing a name at another one.
//
// Beside CNAME and DNAME, they are pseudo-types: DNSControl handles them, but
// they have no DNS wire format, so happyDomain gives them a private use type
// code (see happydns.PseudoTypes) and only a provider declaring the matching
// capability accepts them.
var AliasRecordTypes = []uint16{
	dns.TypeCNAME,
	dns.TypeDNAME,
	happydns.TypeALIAS,
	happydns.TypeANAME,
	happydns.TypeAKAMAICDN,
	happydns.TypeR53ALIAS,
	happydns.TypeAZUREALIAS,
	happydns.TypeAKAMAITLC,
}

// Alias points a name at another one, whichever flavour of alias the provider
// supports. It holds the record itself, so that what the user chose is what
// gets published, without any translation.
type Alias struct {
	Record happydns.Record `json:"record"`
}

func (s *Alias) GetNbResources() int {
	return 1
}

func (s *Alias) GenComment() string {
	if s.Record == nil {
		return ""
	}

	target := aliasTarget(s.Record)

	if s.Record.Header().Rrtype == dns.TypeCNAME {
		return target
	}

	return dns.TypeToString[s.Record.Header().Rrtype] + " " + target
}

func (s *Alias) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	if s.Record == nil {
		return nil, fmt.Errorf("this alias has no record")
	}

	return []happydns.Record{s.Record}, nil
}

// Initialize gives the service the CNAME every provider supports, the frontend
// then offers the flavours the provider declares.
func (s *Alias) Initialize() (any, error) {
	s.Record = helpers.NewRecord("", "CNAME", 0, "")

	return s, nil
}

func (s *Alias) UnmarshalJSON(b []byte) error {
	var stored struct {
		Record json.RawMessage `json:"record"`

		// The shape of the CNAME service this one replaces.
		LegacyCNAME json.RawMessage `json:"cname"`
	}

	if err := json.Unmarshal(b, &stored); err != nil {
		return err
	}

	payload := stored.Record
	if payload == nil {
		payload = stored.LegacyCNAME
	}
	if payload == nil {
		return fmt.Errorf("no record found in this Alias service")
	}

	record, err := UnmarshalRecord(payload)
	if err != nil {
		return err
	}

	s.Record = record

	return nil
}

// aliasTarget returns the name the given alias points at, whichever of the
// alias record types it is.
func aliasTarget(rr happydns.Record) string {
	switch record := rr.(type) {
	case *dns.CNAME:
		return record.Target
	case *dns.DNAME:
		return record.Target
	case *dns.PrivateRR:
		if rdata, ok := record.Data.(happydns.TargetRdata); ok {
			return rdata.GetTarget()
		}
	}

	return ""
}

func alias_analyze(a *svc.Analyzer) error {
	for _, rrtype := range AliasRecordTypes {
		for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: rrtype}) {
			domain := record.Header().Name

			if err := a.UseRR(record, domain, &Alias{
				Record: helpers.RRRelativeSubdomain(record, a.GetOrigin(), domain),
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func init() {
	svc.RegisterService(
		func() happydns.ServiceBody {
			return &Alias{}
		},
		alias_analyze,
		happydns.ServiceInfos{
			Name:        "Alias",
			Categories: []string{
				"alias",
			},
			RecordTypes: AliasRecordTypes,
			Restrictions: happydns.ServiceRestrictions{
				Single: true,
				// Neither Alone nor NeedTypes: a CNAME has to be the only
				// record of its name (RFC 1034 § 3.6.2), but an ALIAS at the
				// apex exists precisely to sit next to the MX and the NS. The
				// rule now depends on the kind of alias, so it is checked by
				// the compliance layer rather than declared here.
			},
		},
		99999998,
		// The type this service goes by in the zones stored before it grew
		// beyond the CNAME.
		"svcs.CNAME",
	)
}
