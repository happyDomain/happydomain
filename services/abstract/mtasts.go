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

package abstract

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	svc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	"git.happydns.org/happyDomain/model"
)

// MTASTSPolicyPath is where RFC 8461 sec. 3.3 mandates the policy file be
// served, under the mta-sts. host of the domain it applies to.
const MTASTSPolicyPath = "/.well-known/mta-sts.txt"

// MTASTSPolicyHostPrefix is the host RFC 8461 sec. 3.3 mandates the policy be
// served on, relative to the domain it applies to.
const MTASTSPolicyHostPrefix = "mta-sts."

// MTASTSDefaultMaxAge is the max_age proposed to users that don't pick one:
// one week, the lower bound RFC 8461 sec. 3.2 recommends.
const MTASTSDefaultMaxAge = 604800

// MTASTS publishes a complete RFC 8461 setup: the `_mta-sts.<domain>` TXT
// record announcing the policy, *and* the policy file itself, served by
// happyDomain at https://mta-sts.<domain>/.well-known/mta-sts.txt behind the
// `mta-sts.<domain>` CNAME.
//
// The low-level svcs.MTA_STS service only carries the TXT record; it remains
// the right choice for someone who serves the policy file themselves.
//
// As for the other hosted services, the records are stored verbatim: the
// frontend editor builds them, so a zone that was analyzed is republished
// byte-for-byte identical. The Mode/MaxAge/MX fields are never published in
// DNS — they are the policy file happyDomain serves.
type MTASTS struct {
	Mode   string   `json:"mode,omitempty" happydomain:"label=Policy Mode,choices=testing;enforce;none,default=testing,description=enforce refuses to deliver when TLS fails; testing only reports; none withdraws the policy."`
	MaxAge uint32   `json:"maxAge,omitempty" happydomain:"label=Max Age,placeholder=604800,description=How long (in seconds) senders should cache this policy."`
	MX     []string `json:"mx,omitempty" happydomain:"label=Authorized MX,placeholder=mail.example.com,description=Host names allowed to receive mail for this domain. A leading *. matches exactly one label."`

	Record      *happydns.TXT `json:"txt,omitempty"`
	PolicyCNAME *dns.CNAME    `json:"policyCNAME,omitempty"`
}

// txtConfigured reports whether the TXT pointer was actually filled in, as
// opposed to being a zero stub left over from the service-spec
// auto-initializer (which pre-allocates pointer-to-DNS fields).
func txtConfigured(txt *happydns.TXT) bool {
	return txt != nil && txt.Hdr.Name != ""
}

// IsHosted reports whether the user asked happyDomain to serve the policy
// file, as opposed to self-hosting it: hostedservice.IsManaged consults this
// so the Caddy on-demand TLS hook never authorises a cert for a domain that
// only configured the DNS half.
func (s *MTASTS) IsHosted() bool {
	return cnameConfigured(s.PolicyCNAME)
}

func (s *MTASTS) GetNbResources() int {
	n := 0
	if txtConfigured(s.Record) {
		n++
	}
	if cnameConfigured(s.PolicyCNAME) {
		n++
	}
	return n
}

func (s *MTASTS) GenComment() string {
	var b strings.Builder

	if s.Mode != "" {
		b.WriteString(s.Mode)
	}
	if len(s.MX) > 0 {
		if b.Len() > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%d MX", len(s.MX))
	}
	if cnameConfigured(s.PolicyCNAME) {
		if b.Len() > 0 {
			b.WriteString(" + ")
		}
		b.WriteString("hosted policy")
	}

	return b.String()
}

// GetRecords returns the stored records verbatim. The frontend editor is
// responsible for filling every field — the backend never synthesises or
// rewrites records, so what was analyzed (or what the user typed) is exactly
// what gets published.
func (s *MTASTS) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	var rrs []happydns.Record

	if txtConfigured(s.Record) {
		rrs = append(rrs, s.Record)
	}
	if cnameConfigured(s.PolicyCNAME) {
		rrs = append(rrs, s.PolicyCNAME)
	}

	return rrs, nil
}

// PolicyFile renders the policy as RFC 8461 sec. 3.2 defines it: a sequence of
// `key: value` lines terminated by CRLF, one `mx` line per authorized host.
//
// It returns nil when the service carries no policy at all, or when the mode
// requires at least one mx directive (rfc8461 3.2) but none is configured —
// serving a testing/enforce policy with no authorized MX would tell senders
// no MX is authorized, locking mail out of the domain, so it's better to
// serve nothing than to serve that.
func (s *MTASTS) PolicyFile() []byte {
	// Without the CNAME, happyDomain does not answer on mta-sts.<domain>: the
	// user asked to self-host the file, so there is nothing for us to serve.
	if !cnameConfigured(s.PolicyCNAME) {
		return nil
	}

	mode := s.Mode
	if mode == "" {
		mode = "testing"
	}

	var mxLines strings.Builder
	for _, mx := range s.MX {
		mx = strings.TrimSpace(mx)
		if mx == "" {
			continue
		}
		fmt.Fprintf(&mxLines, "mx: %s\r\n", strings.TrimSuffix(mx, "."))
	}
	if mode != "none" && mxLines.Len() == 0 {
		return nil
	}

	maxAge := s.MaxAge
	if maxAge == 0 {
		maxAge = MTASTSDefaultMaxAge
	}

	var b strings.Builder
	b.WriteString("version: STSv1\r\n")
	fmt.Fprintf(&b, "mode: %s\r\n", mode)
	b.WriteString(mxLines.String())
	fmt.Fprintf(&b, "max_age: %d\r\n", maxAge)

	return []byte(b.String())
}

// mtasts_analyze reconstructs an MTASTS from a zone import. It only claims
// records when the `_mta-sts.` TXT *and* the `mta-sts.` CNAME are both there:
// that pair is the unambiguous signal the policy file is hosted elsewhere than
// on the user's own web server. A lone TXT record is left to
// svcs.MTA_STS (weight 5), which runs right after this analyzer.
func mtasts_analyze(a *svc.Analyzer) error {
	for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_mta-sts."}) {
		txt, ok := record.(*happydns.TXT)
		// rfc8461: 3.1 records that do not begin with "v=STSv1;" are discarded
		if !ok || !strings.HasPrefix(txt.Txt, "v=STS") {
			continue
		}

		domain := strings.TrimPrefix(record.Header().Name, "_mta-sts.")

		var policyCNAME *dns.CNAME
		for _, candidate := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeCNAME, Prefix: "mta-sts." + domain}) {
			if cname, ok := candidate.(*dns.CNAME); ok && cname.Header().Name == "mta-sts."+domain && hostingTargetMatches(cname.Target) {
				policyCNAME = cname
				break
			}
		}
		if policyCNAME == nil {
			continue
		}

		origin := a.GetOrigin()
		service := &MTASTS{
			Record:      helpers.RRRelativeSubdomain(record, origin, domain).(*happydns.TXT),
			PolicyCNAME: helpers.RRRelativeSubdomain(policyCNAME, origin, domain).(*dns.CNAME),
		}

		if err := a.UseRR(record, domain, service); err != nil {
			return err
		}
		if err := a.UseRR(policyCNAME, domain, service); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	svc.RegisterService(
		func() happydns.ServiceBody { return &MTASTS{} },
		mtasts_analyze,
		happydns.ServiceInfos{
			Name:        "MTA-STS (hosted policy)",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories:  []string{"email"},
			RecordTypes: []uint16{dns.TypeTXT, dns.TypeCNAME},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				Single:    true,
				NeedTypes: []uint16{dns.TypeTXT, dns.TypeCNAME},
			},
		},
		1,
	)
}
