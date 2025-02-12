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
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/utils"
)

type DKIM struct {
	Version        uint     `json:"version" happydomain:"label=Version,placeholder=1,required,description=The version of DKIM to use.,default=1,hidden"`
	AcceptableHash []string `json:"h" happydomain:"label=Hash Algorithms,choices=*;sha1;sha256"`
	KeyType        string   `json:"k" happydomain:"label=Key Type,choices=rsa"`
	Notes          string   `json:"n" happydomain:"label=Notes,description=Notes intended for a foreign postmaster"`
	PublicKey      []byte   `json:"p" happydomain:"label=Public Key,placeholder=a0b1c2d3e4f5==,required"`
	ServiceType    []string `json:"s" happydomain:"label=Service Types,choices=*;email"`
	Flags          []string `json:"t" happydomain:"label=Flags,choices=y;s"`
}

func (t *DKIM) Analyze(txt string) error {
	fields := analyseFields(txt)

	if v, ok := fields["v"]; ok {
		if !strings.HasPrefix(v, "DKIM") {
			return fmt.Errorf("not a valid DKIM record: should begin with v=DKIMv1, seen v=%q", v)
		}

		version, err := strconv.ParseUint(v[4:], 10, 32)
		if err != nil {
			return fmt.Errorf("not a valid DKIM record: bad version number: %w", err)
		}
		t.Version = uint(version)
	} else {
		return fmt.Errorf("not a valid DKIM record: version not found")
	}

	if h, ok := fields["h"]; ok {
		t.AcceptableHash = strings.Split(h, ":")
	} else {
		t.AcceptableHash = []string{"*"}
	}
	if k, ok := fields["k"]; ok {
		t.KeyType = k
	}
	if n, ok := fields["n"]; ok {
		t.Notes = n
	}
	if p, ok := fields["p"]; ok {
		var err error
		t.PublicKey, err = base64.StdEncoding.DecodeString(p)
		if err != nil {
			return fmt.Errorf("not a valid DKIM record: public key is not base64 valid: %w", err)
		}
	}
	if s, ok := fields["s"]; ok {
		t.ServiceType = strings.Split(s, ":")
	} else {
		t.ServiceType = []string{"*"}
	}
	if f, ok := fields["t"]; ok {
		t.Flags = strings.Split(f, ":")
	}

	return nil
}

func (t *DKIM) String() string {
	fields := []string{
		fmt.Sprintf("v=DKIM%d", t.Version),
	}

	if len(t.AcceptableHash) > 1 || (len(t.AcceptableHash) > 0 && t.AcceptableHash[0] != "*") {
		fields = append(fields, fmt.Sprintf("h=%s", strings.Join(t.AcceptableHash, ":")))
	}
	if t.KeyType != "" {
		fields = append(fields, fmt.Sprintf("k=%s", t.KeyType))
	}
	if t.Notes != "" {
		fields = append(fields, fmt.Sprintf("n=%s", t.Notes))
	}
	if len(t.PublicKey) > 0 {
		fields = append(fields, fmt.Sprintf("p=%s", base64.StdEncoding.EncodeToString(t.PublicKey)))
	}
	if len(t.ServiceType) > 1 || (len(t.ServiceType) > 0 && t.ServiceType[0] != "*") {
		fields = append(fields, fmt.Sprintf("s=%s", strings.Join(t.ServiceType, ":")))
	}
	if len(t.Flags) > 0 {
		fields = append(fields, fmt.Sprintf("t=%s", strings.Join(t.Flags, ":")))
	}

	return strings.Join(fields, ";")
}

type DKIMRecord struct {
	Selector string `json:"selector" happydomain:"label=Selector,placeholder=reykjavik,required,description=Name of the key"`
	DKIM
}

func (s *DKIMRecord) GetNbResources() int {
	return 1
}

func (s *DKIMRecord) GenComment(origin string) string {
	return s.Selector
}

func (s *DKIMRecord) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records, e error) {
	rc := utils.NewRecordConfig(utils.DomainJoin(s.Selector+"._domainkey", domain), "TXT", ttl, origin)
	rc.SetTargetTXT(s.String())

	rrs = append(rrs, rc)

	return
}

func dkim_analyze(a *Analyzer) (err error) {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeTXT}) {
		dkidx := strings.Index(record.NameFQDN, "._domainkey.")
		if dkidx <= 0 {
			continue
		}

		service := &DKIMRecord{
			Selector: record.NameFQDN[:dkidx],
		}

		err = service.Analyze(record.GetTargetTXTJoined())
		if err != nil {
			return
		}

		err = a.UseRR(record, record.NameFQDN[dkidx+12:], service)
		if err != nil {
			return
		}
	}

	return
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &DKIMRecord{}
		},
		dkim_analyze,
		ServiceInfos{
			Name:        "DKIM",
			Description: "DomainKeys Identified Mail, authenticate outgoing emails.",
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeTXT,
			},
			Restrictions: ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeTXT,
				},
			},
		},
		1,
	)
}
