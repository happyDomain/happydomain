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
	"strings"

	"github.com/miekg/dns"

	"github.com/StackExchange/dnscontrol/v4/pkg/spflib"

	"git.happydns.org/happyDomain/internal/helpers"
	svc "git.happydns.org/happyDomain/internal/service"
	"git.happydns.org/happyDomain/model"
)

type SPF struct {
	Record *happydns.TXT `json:"txt"`
}

func (s *SPF) GetNbResources() int {
	return 1
}

func (s *SPF) GenComment() string {
	t := SPFFields{}
	t.Analyze(s.Record.Txt)

	return fmt.Sprintf("%d directives", len(t.Directives))
}

func (s *SPF) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	return []happydns.Record{s.Record}, nil
}

type SPFFields struct {
	Version    uint     `json:"version" happydomain:"label=Version,placeholder=1,required,description=The version of SPF to use.,default=1,hidden"`
	Directives []string `json:"directives" happydomain:"label=Directives,placeholder=ip4:203.0.113.12"`
}

func (t *SPFFields) Analyze(txt string) error {
	_, err := spflib.Parse(txt, nil)
	if err != nil {
		return err
	}

	t.Version = 1

	fields := strings.Fields(txt)

	// Avoid doublon
	for _, directive := range fields[1:] {
		exists := false
		for _, known := range t.Directives {
			if known == directive {
				exists = true
				break
			}
		}

		if !exists {
			t.Directives = append(t.Directives, directive)
		}
	}

	return nil
}

func (s *SPFFields) String() string {
	directives := append([]string{fmt.Sprintf("v=spf%d", s.Version)}, s.Directives...)
	return strings.Join(directives, " ")
}

// GetSPFDirectives implements happydns.SPFContributor by parsing the stored
// TXT record and returning all directives except the "all" mechanism.
func (s *SPF) GetSPFDirectives() []string {
	t := SPFFields{}
	if err := t.Analyze(s.Record.Txt); err != nil {
		return nil
	}

	var directives []string
	for _, d := range t.Directives {
		if !strings.HasSuffix(d, "all") {
			directives = append(directives, d)
		}
	}
	return directives
}

// GetSPFAllPolicy implements happydns.SPFContributor by extracting the "all"
// mechanism from the stored TXT record.
func (s *SPF) GetSPFAllPolicy() string {
	t := SPFFields{}
	if err := t.Analyze(s.Record.Txt); err != nil {
		return ""
	}

	for _, d := range t.Directives {
		if strings.HasSuffix(d, "all") {
			return d
		}
	}
	return ""
}

// spfAllPolicyRank returns a numeric rank for SPF "all" policies.
// Higher rank means stricter policy.
func spfAllPolicyRank(policy string) int {
	switch policy {
	case "-all":
		return 4
	case "~all":
		return 3
	case "?all":
		return 2
	case "+all":
		return 1
	default:
		return 0
	}
}

// ResolveSPFAllPolicy picks the strictest "all" policy from the given set.
// Returns "~all" if no valid policy is provided.
func ResolveSPFAllPolicy(policies []string) string {
	best := ""
	bestRank := 0
	for _, p := range policies {
		if r := spfAllPolicyRank(p); r > bestRank {
			bestRank = r
			best = p
		}
	}
	if best == "" {
		return "~all"
	}
	return best
}

// MergeSPFDirectives deduplicates directives across multiple sets.
func MergeSPFDirectives(directiveSets ...[]string) []string {
	seen := map[string]bool{}
	var merged []string
	for _, set := range directiveSets {
		for _, d := range set {
			if !seen[d] {
				seen[d] = true
				merged = append(merged, d)
			}
		}
	}
	return merged
}

// filterClaimedDirectives removes directives that have been claimed by other
// services (e.g. GSuite claiming "include:_spf.google.com") from the SPF
// record text. Returns the filtered TXT content.
func filterClaimedDirectives(txt string, claimed map[string]bool) string {
	if len(claimed) == 0 {
		return txt
	}

	fields := strings.Fields(txt)
	var kept []string
	for _, f := range fields {
		if !claimed[f] {
			kept = append(kept, f)
		}
	}
	return strings.Join(kept, " ")
}

func spf_analyze(a *svc.Analyzer) (err error) {
	for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeTXT, Contains: "v=spf1"}) {
		domain := record.Header().Name
		claimed := a.GetClaimedSPFDirectives(domain)

		relRecord := helpers.RRRelativeSubdomain(record, a.GetOrigin(), domain).(*happydns.TXT)

		if len(claimed) > 0 {
			relRecord.Txt = filterClaimedDirectives(relRecord.Txt, claimed)
		}

		err = a.UseRR(record, domain, &SPF{
			Record: relRecord,
		})
		if err != nil {
			return
		}
	}

	for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeSPF, Contains: "v=spf1"}) {
		spf, ok := record.(*happydns.SPF)
		if !ok {
			continue
		}

		domain := record.Header().Name
		claimed := a.GetClaimedSPFDirectives(domain)

		txt := &happydns.TXT{
			Hdr: spf.Hdr,
			Txt: spf.Txt,
		}
		relRecord := helpers.RRRelativeSubdomain(txt, a.GetOrigin(), domain).(*happydns.TXT)

		if len(claimed) > 0 {
			relRecord.Txt = filterClaimedDirectives(relRecord.Txt, claimed)
		}

		err = a.UseRR(record, domain, &SPF{
			Record: relRecord,
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
			return &SPF{}
		},
		spf_analyze,
		happydns.ServiceInfos{
			Name:        "SPF",
			Description: "Sender Policy Framework, to authenticate domain name on email sending.",
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeTXT,
				dns.TypeSPF,
			},
			Restrictions: happydns.ServiceRestrictions{
				NeedTypes: []uint16{
					dns.TypeTXT,
				},
			},
		},
		1,
	)
}
