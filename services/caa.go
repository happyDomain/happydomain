// Copyright or © or Copr. happyDNS (2023)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

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

	return nb + len(s.Iodef)
}

func (s *CAA) GenComment(origin string) string {
	if s.DisallowIssue {
		return "Certificate issuance disallowed"
	} else {
		var issuance []string
		for _, iss := range s.Issue {
			issuance = append(issuance, iss.IssuerDomainName)
		}

		ret := strings.Join(issuance, ", ")

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

		return ret
	}
}

func (s *CAA) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
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
			Name:        "Certification Authority Authorization (CAA)",
			Description: "Indicate to certificate authorities whether they are authorized to issue digital certificates for a particular domain name.",
			Categories: []string{
				"tls",
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
