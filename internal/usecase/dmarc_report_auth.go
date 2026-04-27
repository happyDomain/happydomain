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

package usecase

import (
	"strings"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

const dmarcReportAuthTimeout = 5 * time.Second

// CheckDMARCReportAuth resolves the RFC 7489 sec. 7.1 cross-domain reporting
// authorization record for (Owner, ExternalDomain).
func (ru *resolverUsecase) CheckDMARCReportAuth(req happydns.DMARCReportAuthRequest) (*happydns.DMARCReportAuthResponse, error) {
	owner := strings.TrimSuffix(strings.TrimSpace(req.Owner), ".")
	external := strings.TrimSuffix(strings.TrimSpace(req.ExternalDomain), ".")
	if owner == "" {
		return nil, happydns.ValidationError{Msg: "owner is required"}
	}
	if external == "" {
		return nil, happydns.ValidationError{Msg: "externalDomain is required"}
	}

	queried := dns.Fqdn(owner + "._report._dmarc." + external)
	resp := &happydns.DMARCReportAuthResponse{QueriedName: strings.TrimSuffix(queried, ".")}

	resolver, err := ru.pickResolver(req.Resolver, req.Custom)
	if err != nil {
		return nil, err
	}

	client := dns.Client{Timeout: dmarcReportAuthTimeout}
	m := new(dns.Msg)
	m.SetQuestion(queried, dns.TypeTXT)
	m.RecursionDesired = true
	m.SetEdns0(4096, true)

	r, _, err := client.Exchange(m, resolver)
	if err != nil {
		resp.Status = "resolver-error"
		resp.ErrorMsg = err.Error()
		return resp, nil
	}
	if r == nil {
		resp.Status = "resolver-error"
		resp.ErrorMsg = "no answer"
		return resp, nil
	}
	switch r.Rcode {
	case dns.RcodeNameError:
		resp.Status = "not-found"
		resp.ErrorMsg = "NXDOMAIN"
		return resp, nil
	case dns.RcodeSuccess:
		// fallthrough
	default:
		resp.Status = "dns-error"
		resp.ErrorMsg = dns.RcodeToString[r.Rcode]
		return resp, nil
	}

	for _, ans := range r.Answer {
		if txt, ok := ans.(*dns.TXT); ok {
			resp.Records = append(resp.Records, strings.Join(txt.Txt, ""))
		}
	}
	if len(resp.Records) == 0 {
		resp.Status = "not-found"
		resp.ErrorMsg = "no TXT record"
		return resp, nil
	}
	for _, rec := range resp.Records {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(rec)), "v=dmarc1") {
			resp.Status = "ok"
			return resp, nil
		}
	}
	resp.Status = "no-dmarc-record"
	resp.ErrorMsg = "no TXT starts with v=DMARC1"
	return resp, nil
}
