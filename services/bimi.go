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
	"fmt"
	"strconv"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	svc "git.happydns.org/happyDomain/internal/service"
	"git.happydns.org/happyDomain/model"
)

type BIMI struct {
	Record *happydns.TXT `json:"txt"`
}

func (s *BIMI) GetNbResources() int {
	return 1
}

func (s *BIMI) GenComment() string {
	return strings.SplitN(s.Record.Header().Name, ".", 2)[0]
}

func (s *BIMI) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	return []happydns.Record{s.Record}, nil
}

type BIMIFields struct {
	Version   uint   `json:"version" happydomain:"label=Version,placeholder=1,required,description=The version of BIMI to use.,default=1,hidden"`
	Location  string `json:"l" happydomain:"label=Logo,description=HTTPS URL of the SVG Tiny Portable/Secure logo.,placeholder=https://example.com/logo.svg,required"`
	Authority string `json:"a" happydomain:"label=Authority,description=HTTPS URL of the Verified Mark Certificate (PEM).,placeholder=https://example.com/vmc.pem"`
	Evidence  string `json:"e" happydomain:"label=Evidence,description=HTTPS URL of an evidence document."`
}

func (t *BIMIFields) Analyze(txt string) error {
	fields := analyseFields(txt)

	v, ok := fields["v"]
	if !ok {
		return fmt.Errorf("not a valid BIMI record: version not found")
	}
	if !strings.HasPrefix(v, "BIMI") {
		return fmt.Errorf("not a valid BIMI record: should begin with v=BIMI1, seen v=%q", v)
	}
	version, err := strconv.ParseUint(v[4:], 10, 32)
	if err != nil {
		return fmt.Errorf("not a valid BIMI record: bad version number: %w", err)
	}
	t.Version = uint(version)

	if l, ok := fields["l"]; ok {
		t.Location = l
	}
	if a, ok := fields["a"]; ok {
		t.Authority = a
	}
	if e, ok := fields["e"]; ok {
		t.Evidence = e
	}

	return nil
}

func (t *BIMIFields) String() string {
	fields := []string{
		fmt.Sprintf("v=BIMI%d", t.Version),
	}

	if t.Location != "" {
		fields = append(fields, fmt.Sprintf("l=%s", t.Location))
	}
	if t.Authority != "" {
		fields = append(fields, fmt.Sprintf("a=%s", t.Authority))
	}
	if t.Evidence != "" {
		fields = append(fields, fmt.Sprintf("e=%s", t.Evidence))
	}

	return strings.Join(fields, ";")
}

func bimi_analyze(a *svc.Analyzer) (err error) {
	for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeTXT}) {
		idx := strings.Index(record.Header().Name, "._bimi.")
		if idx <= 0 {
			continue
		}
		txt, ok := record.(*happydns.TXT)
		if !ok || !strings.HasPrefix(txt.Txt, "v=BIMI") {
			continue
		}
		domain := record.Header().Name[idx+len("._bimi."):]

		err = a.UseRR(record, domain, &BIMI{
			Record: helpers.RRRelativeSubdomain(record, a.GetOrigin(), domain).(*happydns.TXT),
		})
		if err != nil {
			return
		}
	}

	return
}

func init() {
	svc.RegisterService(
		func() happydns.ServiceBody {
			return &BIMI{}
		},
		bimi_analyze,
		happydns.ServiceInfos{
			Name:        "BIMI",
			Description: "Brand Indicators for Message Identification, display brand logos in supporting mail clients.",
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
