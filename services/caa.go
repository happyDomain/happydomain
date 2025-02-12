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

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/common"
	"git.happydns.org/happyDomain/utils"
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

func (s *CAA) GenComment(origin string) (ret string) {
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

func (s *CAA) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records, e error) {
	if s.DisallowIssue {
		rc := utils.NewRecordConfig(domain, "CAA", ttl, origin)
		rc.CaaFlag = 0
		rc.CaaTag = "issue"
		rc.SetTarget(";")

		rrs = append(rrs, rc)
	} else {
		for _, issue := range s.Issue {
			rc := utils.NewRecordConfig(domain, "CAA", ttl, origin)
			rc.CaaFlag = 0
			rc.CaaTag = "issue"
			rc.SetTarget(issue.String())

			rrs = append(rrs, rc)
		}

		if s.DisallowWildcardIssue {
			rc := utils.NewRecordConfig(domain, "CAA", ttl, origin)
			rc.CaaFlag = 0
			rc.CaaTag = "issuewild"
			rc.SetTarget(";")

			rrs = append(rrs, rc)
		} else {
			for _, issue := range s.IssueWild {
				rc := utils.NewRecordConfig(domain, "CAA", ttl, origin)
				rc.CaaFlag = 0
				rc.CaaTag = "issuewild"
				rc.SetTarget(issue.String())

				rrs = append(rrs, rc)
			}
		}
	}

	if s.DisallowMailIssue {
		rc := utils.NewRecordConfig(domain, "CAA", ttl, origin)
		rc.CaaFlag = 0
		rc.CaaTag = "issuemail"
		rc.SetTarget(";")

		rrs = append(rrs, rc)
	} else {
		for _, issue := range s.IssueMail {
			rc := utils.NewRecordConfig(domain, "CAA", ttl, origin)
			rc.CaaFlag = 0
			rc.CaaTag = "issuemail"
			rc.SetTarget(issue.String())

			rrs = append(rrs, rc)
		}
	}

	if len(s.Iodef) > 0 {
		for _, iodef := range s.Iodef {
			rc := utils.NewRecordConfig(domain, "CAA", ttl, origin)
			rc.CaaFlag = 0
			rc.CaaTag = "iodef"
			rc.SetTarget(iodef.String())

			rrs = append(rrs, rc)
		}
	}

	return
}

func caa_analyze(a *Analyzer) (err error) {
	pool := map[string]*CAA{}

	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeCAA}) {
		domain := record.NameFQDN

		if record.Type == "CAA" {
			if _, ok := pool[domain]; !ok {
				pool[domain] = &CAA{}
			}

			analyzed := pool[domain]

			if record.CaaTag == "issue" {
				value := record.GetTargetField()
				if value == ";" {
					analyzed.DisallowIssue = true
				} else {
					analyzed.Issue = append(analyzed.Issue, parseIssueValue(value))
				}
			}

			if record.CaaTag == "issuewild" {
				value := record.GetTargetField()
				if value == ";" {
					analyzed.DisallowWildcardIssue = true
				} else {
					analyzed.IssueWild = append(analyzed.IssueWild, parseIssueValue(value))
				}
			}

			if record.CaaTag == "issuemail" {
				value := record.GetTargetField()
				if value == ";" {
					analyzed.DisallowMailIssue = true
				} else {
					analyzed.IssueMail = append(analyzed.Issue, parseIssueValue(value))
				}
			}

			if record.CaaTag == "iodef" {
				if u, err := url.Parse(record.GetTargetField()); err != nil {
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
		func() happydns.Service {
			return &CAA{}
		},
		caa_analyze,
		ServiceInfos{
			Name:        "Certification Authority Authorization",
			Description: "Indicate to certificate authorities whether they are authorized to issue digital certificates for a particular domain name.",
			Categories: []string{
				"security",
			},
			RecordTypes: []uint16{
				dns.TypeCAA,
			},
			Restrictions: ServiceRestrictions{
				Single: true,
				NeedTypes: []uint16{
					dns.TypeCAA,
				},
			},
		},
		1,
	)
}
