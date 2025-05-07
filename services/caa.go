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
	"net/url"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/common"
)

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

type CAA struct {
	DisallowIssue         bool
	Issue                 []CAAIssueValue
	DisallowWildcardIssue bool
	IssueWild             []CAAIssueValue
	DisallowMailIssue     bool
	IssueMail             []CAAIssueValue
	Iodef                 []*common.URL
}

func (s *CAA) GetNbResources() int {
	nb := 0

	if s.DisallowIssue {
		nb += 1
	} else {
		nb += len(s.Issue)
		if s.DisallowWildcardIssue {
			nb += 1
		} else {
			nb += len(s.IssueWild)
		}
	}

	if s.DisallowMailIssue {
		nb += 1
	} else {
		nb += len(s.IssueMail)
	}

	return nb + len(s.Iodef)
}

func (s *CAA) GenComment() (ret string) {
	if s.DisallowIssue {
		ret = "Certificate issuance disallowed"
	} else {
		var issuance []string
		for _, iss := range s.Issue {
			issuance = append(issuance, iss.IssuerDomainName)
		}

		ret = strings.Join(issuance, ", ")

		if s.DisallowWildcardIssue {
			if ret != "" {
				ret += "; "
			}
			ret += "Wildcard issuance disallowed"
		} else if len(s.IssueWild) > 0 {
			if ret != "" {
				ret += "; wildcard: "
			}

			var issuancew []string
			for _, iss := range s.IssueWild {
				issuancew = append(issuancew, iss.IssuerDomainName)
			}

			ret += strings.Join(issuancew, ", ")
		}
	}

	if s.DisallowMailIssue {
		if ret != "" {
			ret += "; "
		}
		ret += "S/MIME issuance disallowed"
	} else if len(s.IssueMail) > 0 {
		if ret != "" {
			ret += "; S/MIME: "
		}

		var issuancem []string
		for _, iss := range s.IssueMail {
			issuancem = append(issuancem, iss.IssuerDomainName)
		}

		ret += strings.Join(issuancem, ", ")
	}

	return
}

func (s *CAA) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	if s.DisallowIssue {
		rr := helpers.NewRecord(domain, "CAA", ttl, origin)
		rr.(*dns.CAA).Flag = 0
		rr.(*dns.CAA).Tag = "issue"
		rr.(*dns.CAA).Tag = "issue"
		rr.(*dns.CAA).Value = ";"

		rrs = append(rrs, rr)
	} else {
		for _, issue := range s.Issue {
			rr := helpers.NewRecord(domain, "CAA", ttl, origin)
			rr.(*dns.CAA).Flag = 0
			rr.(*dns.CAA).Tag = "issue"
			rr.(*dns.CAA).Value = issue.String()

			rrs = append(rrs, rr)
		}

		if s.DisallowWildcardIssue {
			rr := helpers.NewRecord(domain, "CAA", ttl, origin)
			rr.(*dns.CAA).Flag = 0
			rr.(*dns.CAA).Tag = "issuewild"
			rr.(*dns.CAA).Value = ";"

			rrs = append(rrs, rr)
		} else {
			for _, issue := range s.IssueWild {
				rr := helpers.NewRecord(domain, "CAA", ttl, origin)
				rr.(*dns.CAA).Flag = 0
				rr.(*dns.CAA).Tag = "issuewild"
				rr.(*dns.CAA).Value = issue.String()

				rrs = append(rrs, rr)
			}
		}
	}

	if s.DisallowMailIssue {
		rr := helpers.NewRecord(domain, "CAA", ttl, origin)
		rr.(*dns.CAA).Flag = 0
		rr.(*dns.CAA).Tag = "issuemail"
		rr.(*dns.CAA).Value = ";"

		rrs = append(rrs, rr)
	} else {
		for _, issue := range s.IssueMail {
			rr := helpers.NewRecord(domain, "CAA", ttl, origin)
			rr.(*dns.CAA).Flag = 0
			rr.(*dns.CAA).Tag = "issuemail"
			rr.(*dns.CAA).Value = issue.String()

			rrs = append(rrs, rr)
		}
	}

	if len(s.Iodef) > 0 {
		for _, iodef := range s.Iodef {
			rr := helpers.NewRecord(domain, "CAA", ttl, origin)
			rr.(*dns.CAA).Flag = 0
			rr.(*dns.CAA).Tag = "iodef"
			rr.(*dns.CAA).Value = iodef.String()

			rrs = append(rrs, rr)
		}
	}

	return
}

func caa_analyze(a *Analyzer) (err error) {
	pool := map[string]*CAA{}

	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeCAA}) {
		domain := record.Header().Name

		if caa, ok := record.(*dns.CAA); ok {
			if _, ok := pool[domain]; !ok {
				pool[domain] = &CAA{}
			}

			analyzed := pool[domain]

			if caa.Tag == "issue" {
				value := caa.Value
				if value == ";" {
					analyzed.DisallowIssue = true
				} else {
					analyzed.Issue = append(analyzed.Issue, parseIssueValue(value))
				}
			}

			if caa.Tag == "issuewild" {
				value := caa.Value
				if value == ";" {
					analyzed.DisallowWildcardIssue = true
				} else {
					analyzed.IssueWild = append(analyzed.IssueWild, parseIssueValue(value))
				}
			}

			if caa.Tag == "issuemail" {
				value := caa.Value
				if value == ";" {
					analyzed.DisallowMailIssue = true
				} else {
					analyzed.IssueMail = append(analyzed.Issue, parseIssueValue(value))
				}
			}

			if caa.Tag == "iodef" {
				if u, err := url.Parse(caa.Value); err != nil {
					continue
				} else {
					tmp := common.URL(*u)
					analyzed.Iodef = append(analyzed.Iodef, &tmp)
				}
			}

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
			return &CAA{}
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
