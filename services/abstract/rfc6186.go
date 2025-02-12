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

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/utils"
)

type RFC6186 struct {
	Submission  []*svcs.SRV `json:"submission" happydomain:"label=Email Submission,description=Identifies domain's Message Submission Agent"`
	IMAPS       []*svcs.SRV `json:"imaps" happydomain:"label=IMAP over TLS,description=Identifies domain's IMAP server running over TLS"`
	POP3S       []*svcs.SRV `json:"pop3s" happydomain:"label=POP3 over TLS,description=Identifies domain's POP3 server running over TLS"`
	SubmissionS []*svcs.SRV `json:"submissions" happydomain:"label=Email Submission over TLS,description=Identifies domain's Message Submission Agent running over TLS"` // RFC 8314
	IMAP        []*svcs.SRV `json:"imap" happydomain:"label=IMAP,description=Identifies domain's IMAP server running unencrypted"`
	POP3        []*svcs.SRV `json:"pop3" happydomain:"label=POP3,description=Identifies domain's POP3 server running unencrypted"`
}

func (s *RFC6186) GetNbResources() int {
	return len(s.Submission) + len(s.SubmissionS) + len(s.IMAP) + len(s.IMAPS) + len(s.POP3) + len(s.POP3S)
}

func (s *RFC6186) GenComment(origin string) string {
	var b strings.Builder

	if len(s.Submission) > 1 {
		fmt.Fprintf(&b, "%d submissions", len(s.Submission))
	} else if len(s.Submission) > 0 {
		b.WriteString("Submission")
	}

	if len(s.SubmissionS) > 0 {
		if b.Len() > 0 {
			b.WriteString(" + ")
		}
		if len(s.SubmissionS) > 1 {
			fmt.Fprintf(&b, "%d secured submissions", len(s.IMAP))
		} else if len(s.IMAP) > 0 {
			if b.Len() > 0 {
				b.WriteString("secured submission")
			} else {
				b.WriteString("Secured submission")
			}
		}
	}

	if len(s.IMAP) > 0 {
		if b.Len() > 0 {
			b.WriteString(" + ")
		}
		if len(s.IMAP) > 1 {
			fmt.Fprintf(&b, "%d IMAP", len(s.IMAP))
		} else if len(s.IMAP) > 0 {
			b.WriteString("IMAP")
		}
	}

	if len(s.IMAP) > 0 {
		if b.Len() > 0 {
			b.WriteString(" + ")
		}
		if len(s.IMAP) > 1 {
			fmt.Fprintf(&b, "%d IMAP", len(s.IMAP))
		} else if len(s.IMAP) > 0 {
			b.WriteString("IMAP")
		}
	}

	if len(s.IMAPS) > 0 {
		if b.Len() > 0 {
			b.WriteString(" + ")
		}
		if len(s.IMAPS) > 1 {
			fmt.Fprintf(&b, "%d secured IMAP", len(s.IMAPS))
		} else if len(s.IMAPS) > 0 {
			b.WriteString("secured IMAP")
		}
	}

	if len(s.POP3) > 0 {
		if b.Len() > 0 {
			b.WriteString(" + ")
		}
		if len(s.POP3) > 1 {
			fmt.Fprintf(&b, "%d POP3", len(s.POP3))
		} else if len(s.POP3) > 0 {
			b.WriteString("POP3")
		}
	}

	if len(s.POP3S) > 0 {
		if b.Len() > 0 {
			b.WriteString(" + ")
		}
		if len(s.POP3S) > 1 {
			fmt.Fprintf(&b, "%d secured POP3", len(s.POP3S))
		} else if len(s.POP3S) > 0 {
			b.WriteString("secured POP3")
		}
	}

	return b.String()
}

func (s *RFC6186) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records, e error) {
	for _, service := range s.Submission {
		if service.Port == 0 {
			service.Port = 587
		}
		srrs, err := service.GenRRs(utils.DomainJoin("_submission._tcp", domain), ttl, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate submission records: %w", err)
		}
		rrs = append(rrs, srrs...)
	}
	for _, service := range s.SubmissionS {
		if service.Port == 0 {
			service.Port = 587
		}
		srrs, err := service.GenRRs(utils.DomainJoin("_submissions._tcp", domain), ttl, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate submissionS records: %w", err)
		}
		rrs = append(rrs, srrs...)
	}
	for _, service := range s.IMAP {
		if service.Port == 0 {
			service.Port = 143
		}
		srrs, err := service.GenRRs(utils.DomainJoin("_imap._tcp", domain), ttl, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate imap records: %w", err)
		}
		rrs = append(rrs, srrs...)
	}
	for _, service := range s.IMAPS {
		if service.Port == 0 {
			service.Port = 993
		}
		srrs, err := service.GenRRs(utils.DomainJoin("_imaps._tcp", domain), ttl, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate imaps records: %w", err)
		}
		rrs = append(rrs, srrs...)
	}
	for _, service := range s.POP3 {
		if service.Port == 0 {
			service.Port = 110
		}
		srrs, err := service.GenRRs(utils.DomainJoin("_pop3._tcp", domain), ttl, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate pop3 records: %w", err)
		}
		rrs = append(rrs, srrs...)
	}
	for _, service := range s.POP3S {
		if service.Port == 0 {
			service.Port = 995
		}
		srrs, err := service.GenRRs(utils.DomainJoin("_pop3s._tcp", domain), ttl, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate pop3s records: %w", err)
		}
		rrs = append(rrs, srrs...)
	}
	return
}

func rfc6186_analyze(a *svcs.Analyzer) error {
	emailDomains := map[string]*RFC6186{}

	// Looking for submission
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Prefix: "_submission._tcp.", Type: dns.TypeSRV}) {
		if srv := svcs.ParseSRV(record); srv != nil {
			domain := strings.TrimPrefix(record.NameFQDN, "_submission._tcp.")

			if _, ok := emailDomains[domain]; !ok {
				emailDomains[domain] = &RFC6186{}
			}

			emailDomains[domain].Submission = append(emailDomains[domain].Submission, srv)

			a.UseRR(
				record,
				domain,
				emailDomains[domain],
			)
		}
	}

	// Looking for submissionS
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Prefix: "_submissions._tcp.", Type: dns.TypeSRV}) {
		if srv := svcs.ParseSRV(record); srv != nil {
			domain := strings.TrimPrefix(record.NameFQDN, "_submissions._tcp.")

			if _, ok := emailDomains[domain]; !ok {
				emailDomains[domain] = &RFC6186{}
			}

			emailDomains[domain].SubmissionS = append(emailDomains[domain].SubmissionS, srv)

			a.UseRR(
				record,
				domain,
				emailDomains[domain],
			)
		}
	}

	// Looking for IMAP
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Prefix: "_imap._tcp.", Type: dns.TypeSRV}) {
		if srv := svcs.ParseSRV(record); srv != nil {
			domain := strings.TrimPrefix(record.NameFQDN, "_imap._tcp.")

			if _, ok := emailDomains[domain]; !ok {
				emailDomains[domain] = &RFC6186{}
			}

			emailDomains[domain].IMAP = append(emailDomains[domain].IMAP, srv)

			a.UseRR(
				record,
				domain,
				emailDomains[domain],
			)
		}
	}

	// Looking for IMAPS
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Prefix: "_imaps._tcp.", Type: dns.TypeSRV}) {
		if srv := svcs.ParseSRV(record); srv != nil {
			domain := strings.TrimPrefix(record.NameFQDN, "_imaps._tcp.")

			if _, ok := emailDomains[domain]; !ok {
				emailDomains[domain] = &RFC6186{}
			}

			emailDomains[domain].IMAPS = append(emailDomains[domain].IMAPS, srv)

			a.UseRR(
				record,
				domain,
				emailDomains[domain],
			)
		}
	}

	// Looking for POP3
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Prefix: "_pop3._tcp.", Type: dns.TypeSRV}) {
		if srv := svcs.ParseSRV(record); srv != nil {
			domain := strings.TrimPrefix(record.NameFQDN, "_pop3._tcp.")

			if _, ok := emailDomains[domain]; !ok {
				emailDomains[domain] = &RFC6186{}
			}

			emailDomains[domain].POP3 = append(emailDomains[domain].POP3, srv)

			a.UseRR(
				record,
				domain,
				emailDomains[domain],
			)
		}
	}

	// Looking for POP3S
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Prefix: "_pop3s._tcp.", Type: dns.TypeSRV}) {
		if srv := svcs.ParseSRV(record); srv != nil {
			domain := strings.TrimPrefix(record.NameFQDN, "_pop3s._tcp.")

			if _, ok := emailDomains[domain]; !ok {
				emailDomains[domain] = &RFC6186{}
			}

			emailDomains[domain].POP3S = append(emailDomains[domain].POP3S, srv)

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
