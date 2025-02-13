// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package abstract

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/utils"
)

type RFC6186 struct {
	Records []*dns.SRV `json:"srv"`
}

func (s *RFC6186) GetNbResources() int {
	return len(s.Records)
}

func (s *RFC6186) GenComment(origin string) string {
	var b strings.Builder

	var submission, submissionS, pop3, pop3s, imap, imaps uint

	for _, record := range s.Records {
		domain := record.Hdr.Name

		if strings.HasPrefix(domain, "_submission._tcp.") {
			submission += 1
		} else if strings.HasPrefix(domain, "_submissions._tcp.") {
			submissionS += 1
		} else if strings.HasPrefix(domain, "_imap._tcp.") {
			imap += 1
		} else if strings.HasPrefix(domain, "_imaps._tcp.") {
			imaps += 1
		} else if strings.HasPrefix(domain, "_pop3._tcp.") {
			pop3 += 1
		} else if strings.HasPrefix(domain, "_pop3s._tcp.") {
			pop3s += 1
		}
	}

	if submission > 1 {
		fmt.Fprintf(&b, "%d submissions", submission)
	} else if submission > 0 {
		b.WriteString("Submission")
	}

	if submissionS > 0 {
		if b.Len() > 0 {
			b.WriteString(" + ")
		}
		if submissionS > 1 {
			fmt.Fprintf(&b, "%d secured submissions", submissionS)
		} else if submissionS > 0 {
			if b.Len() > 0 {
				b.WriteString("secured submission")
			} else {
				b.WriteString("Secured submission")
			}
		}
	}

	if imap > 0 {
		if b.Len() > 0 {
			b.WriteString(" + ")
		}
		if imap > 1 {
			fmt.Fprintf(&b, "%d IMAP", imap)
		} else if imap > 0 {
			b.WriteString("IMAP")
		}
	}

	if imaps > 0 {
		if b.Len() > 0 {
			b.WriteString(" + ")
		}
		if imaps > 1 {
			fmt.Fprintf(&b, "%d secured IMAP", imaps)
		} else if imaps > 0 {
			b.WriteString("secured IMAP")
		}
	}

	if pop3 > 0 {
		if b.Len() > 0 {
			b.WriteString(" + ")
		}
		if pop3 > 1 {
			fmt.Fprintf(&b, "%d POP3", pop3)
		} else if pop3 > 0 {
			b.WriteString("POP3")
		}
	}

	if pop3s > 0 {
		if b.Len() > 0 {
			b.WriteString(" + ")
		}
		if pop3s > 1 {
			fmt.Fprintf(&b, "%d secured POP3", pop3s)
		} else if pop3s > 0 {
			b.WriteString("secured POP3")
		}
	}

	return b.String()
}

func (s *RFC6186) GetRecords(domain string, ttl uint32, origin string) ([]dns.RR, error) {
	rrs := make([]dns.RR, len(s.Records))
	for i, r := range s.Records {
		srv := *r
		srv.Target = utils.DomainFQDN(srv.Target, origin)
		rrs[i] = &srv
	}
	return rrs, nil
}

func rfc6186_analyze(a *svcs.Analyzer) error {
	emailDomains := map[string]*RFC6186{}

	for _, prefix := range []string{
		"_submission._tcp.",
		"_submissions._tcp.", // RFC 8314
		"_imap._tcp.",
		"_imaps._tcp.",
		"_pop3._tcp.",
		"_pop3s._tcp.",
	} {
		for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Prefix: prefix, Type: dns.TypeSRV}) {
			domain := strings.TrimPrefix(record.NameFQDN, prefix)

			if _, ok := emailDomains[domain]; !ok {
				emailDomains[domain] = &RFC6186{}
			}

			// Make record relative
			record.SetTarget(utils.DomainRelative(record.GetTargetField(), a.GetOrigin()))

			emailDomains[domain].Records = append(emailDomains[domain].Records, utils.RRRelative(record.ToRR(), domain).(*dns.SRV))

			a.UseRR(
				record,
				domain,
				emailDomains[domain],
			)
		}
	}

	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &RFC6186{}
		},
		rfc6186_analyze,
		svcs.ServiceInfos{
			Name:        "E-Mail Services Discovery",
			Description: "Make email clients aware of the domain configuration to send and receive emails. RFC 6186",
			Family:      svcs.Abstract,
			Categories: []string{
				"email",
			},
			Restrictions: svcs.ServiceRestrictions{
				NearAlone: true,
				Single:    true,
				NeedTypes: []uint16{
					dns.TypeSRV,
				},
			},
		},
		2,
	)
}
