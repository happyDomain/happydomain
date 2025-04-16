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
	"net/url"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/common"
)

type CAAPolicy struct {
	Records []*dns.CAA `json:"caa"`
}

func (s *CAAPolicy) GetNbResources() int {
	return len(s.Records)
}

func (s *CAAPolicy) GenComment() (ret string) {
	t := CAAFields{}
	for _, caa := range s.Records {
		t.Analyze(caa.Flag, caa.Tag, caa.Value)
	}

	if t.DisallowIssue {
		ret = "Certificate issuance disallowed"
	} else {
		var issuance []string
		for _, iss := range t.Issue {
			issuance = append(issuance, iss.IssuerDomainName)
		}

		ret = strings.Join(issuance, ", ")

		if t.DisallowWildcardIssue {
			if ret != "" {
				ret += "; "
			}
			ret += "Wildcard issuance disallowed"
		} else if len(t.IssueWild) > 0 {
			if ret != "" {
				ret += "; wildcard: "
			}

			var issuancew []string
			for _, iss := range t.IssueWild {
				issuancew = append(issuancew, iss.IssuerDomainName)
			}

			ret += strings.Join(issuancew, ", ")
		}
	}

	if t.DisallowMailIssue {
		if ret != "" {
			ret += "; "
		}
		ret += "S/MIME issuance disallowed"
	} else if len(t.IssueMail) > 0 {
		if ret != "" {
			ret += "; S/MIME: "
		}

		var issuancem []string
		for _, iss := range t.IssueMail {
			issuancem = append(issuancem, iss.IssuerDomainName)
		}

		ret += strings.Join(issuancem, ", ")
	}

	return
}

func (s *CAAPolicy) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rrs := make([]happydns.Record, len(s.Records))
	for i, r := range s.Records {
		rrs[i] = r
	}
	return rrs, nil
}

type CAAParameter struct {
	Tag   string
	Value string
}

type CAAIssueValue struct {
	IssuerDomainName string
	Parameters       []CAAParameter
}

func parseIssueValue(value string) (ret CAAIssueValue) {
	tmp := strings.Split(value, ";")
	ret.IssuerDomainName = strings.TrimSpace(tmp[0])

	for _, param := range tmp[1:] {
		tmpparam := strings.SplitN(param, "=", 2)
		ret.Parameters = append(ret.Parameters, CAAParameter{
			Tag:   strings.TrimSpace(tmpparam[0]),
			Value: strings.TrimSpace(tmpparam[1]),
		})
	}

	return
}

func (v *CAAIssueValue) String() string {
	var b strings.Builder

	b.WriteString(v.IssuerDomainName)

	if len(v.Parameters) > 0 {
		b.WriteString(";")

		for _, param := range v.Parameters {
			b.WriteString(param.Tag)
			b.WriteString("=")
			b.WriteString(param.Value)
		}
	}

	return b.String()
}

type CAAFields struct {
	DisallowIssue         bool
	Issue                 []CAAIssueValue
	DisallowWildcardIssue bool
	IssueWild             []CAAIssueValue
	DisallowMailIssue     bool
	IssueMail             []CAAIssueValue
	Iodef                 []*common.URL
}

func (analyzed *CAAFields) Analyze(flag uint8, tag, value string) error {
	if tag == "issue" {
		if value == ";" {
			analyzed.DisallowIssue = true
		} else {
			analyzed.Issue = append(analyzed.Issue, parseIssueValue(value))
		}
	}

	if tag == "issuewild" {
		if value == ";" {
			analyzed.DisallowWildcardIssue = true
		} else {
			analyzed.IssueWild = append(analyzed.IssueWild, parseIssueValue(value))
		}
	}

	if tag == "issuemail" {
		if value == ";" {
			analyzed.DisallowMailIssue = true
		} else {
			analyzed.IssueMail = append(analyzed.Issue, parseIssueValue(value))
		}
	}

	if tag == "iodef" {
		if u, err := url.Parse(value); err != nil {
			return fmt.Errorf("unable to parse CAA field: %q: %w", value, err)
		} else {
			tmp := common.URL(*u)
			analyzed.Iodef = append(analyzed.Iodef, &tmp)
		}
	}

	return nil
}

func caa_analyze(a *Analyzer) (err error) {
	pool := map[string]*CAAPolicy{}

	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeCAA}) {
		domain := record.Header().Name

		if record.Header().Rrtype == dns.TypeCAA {
			if _, ok := pool[domain]; !ok {
				pool[domain] = &CAAPolicy{}
			}

			analyzed := pool[domain]
			analyzed.Records = append(analyzed.Records, helpers.RRRelative(record, domain).(*dns.CAA))

			err = a.UseRR(record, domain, pool[domain])
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
			return &CAAPolicy{}
		},
		caa_analyze,
		happydns.ServiceInfos{
			Name:        "Certification Authority Authorization",
			Description: "Indicate to certificate authorities whether they are authorized to issue digital certificates for a particular domain name.",
			Categories: []string{
				"security",
			},
			RecordTypes: []uint16{
				dns.TypeCAA,
			},
			Restrictions: happydns.ServiceRestrictions{
				Single: true,
				NeedTypes: []uint16{
					dns.TypeCAA,
				},
			},
		},
		1,
	)
}
