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

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
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
	Record *happydns.TXT `json:"txt"`
}

func (s *DKIMRecord) Analyze() (*DKIM, error) {
	dkim := &DKIM{}

	err := dkim.Analyze(s.Record.Txt)
	if err != nil {
		return nil, err
	}

	return dkim, nil
}

func (s *DKIMRecord) GetNbResources() int {
	return 1
}

func (s *DKIMRecord) GenComment() string {
	return strings.SplitN(s.Record.Header().Name, ".", 2)[0]
}

func (s *DKIMRecord) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	return []happydns.Record{s.Record}, nil
}

type DKIMRedirection struct {
	Record *dns.CNAME `json:"cname"`
}

func (s *DKIMRedirection) GetNbResources() int {
	return 1
}

func (s *DKIMRedirection) GenComment() string {
	return strings.SplitN(s.Record.Header().Name, ".", 2)[0]
}

func (s *DKIMRedirection) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	cname := *s.Record
	cname.Target = helpers.DomainFQDN(cname.Target, origin)
	return []happydns.Record{&cname}, nil
}

func dkim_analyze(a *Analyzer) (err error) {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeTXT, DomainContains: "._domainkey."}) {
		dkidx := strings.Index(record.Header().Name, "._domainkey.")
		if dkidx <= 0 {
			continue
		}
		domain := record.Header().Name[dkidx+12:]

		service := &DKIMRecord{
			Record: helpers.RRRelative(record, domain).(*happydns.TXT),
		}

		_, err = service.Analyze()
		if err != nil {
			return
		}

		err = a.UseRR(record, domain, service)
		if err != nil {
			return
		}
	}

	return
}

func dkimcname_analyze(a *Analyzer) (err error) {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeCNAME}) {
		dkidx := strings.Index(record.Header().Name, "._domainkey.")
		if dkidx <= 0 {
			continue
		}
		if cname, ok := record.(*dns.CNAME); ok {
			// Make record relative
			cname.Target = helpers.DomainRelative(cname.Target, a.GetOrigin())

			domain := record.Header().Name[dkidx+12:]
			err = a.UseRR(record, domain, &DKIMRedirection{
				Record: helpers.RRRelative(cname, domain).(*dns.CNAME),
			})
			if err != nil {
				return
			}
		}
	}

	return
}

func init() {
	RegisterService(
		func() happydns.ServiceBody {
			return &DKIMRecord{}
		},
		dkim_analyze,
		happydns.ServiceInfos{
			Name:        "DKIM",
			Description: "DomainKeys Identified Mail, authenticate outgoing emails.",
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
	RegisterService(
		func() happydns.ServiceBody {
			return &DKIMRedirection{}
		},
		dkimcname_analyze,
		happydns.ServiceInfos{
			Name:        "DKIM external",
			Description: "DKIM record redirected to another resource.",
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeCNAME,
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeCNAME,
				},
			},
		},
		1,
	)
}
