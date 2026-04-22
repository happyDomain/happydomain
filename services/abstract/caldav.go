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
	"bytes"
	"strconv"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	svc "git.happydns.org/happyDomain/internal/service"
	"git.happydns.org/happyDomain/model"
)

// CalDAV groups the SRV records that announce a CalDAV (calendar) server for
// a domain, per RFC 6764. Both the secure (_caldavs._tcp) and legacy
// plaintext (_caldav._tcp) prefixes are accepted; the service is referenced
// by the checker-caldav Availability via the identifier "abstract.CalDAV".
//
// Paths holds the optional RFC 6764 §4 "context path" TXT records collected
// at the same labels as the SRV records (e.g. `_caldavs._tcp TXT path=/caldav`).
// The full TXT is preserved so Hdr (name, TTL) and Txt round-trip verbatim.
type CalDAV struct {
	Records []*dns.SRV      `json:"records"`
	Paths   []*happydns.TXT `json:"paths,omitempty"`
}

// caldavPrefixes are the SRV name prefixes that collectively describe a
// CalDAV deployment. Order drives the display order in GenComment (secure
// first, plaintext second).
var caldavPrefixes = []string{
	"_caldavs._tcp.",
	"_caldav._tcp.",
}

func (s *CalDAV) GetNbResources() int {
	return len(s.Records)
}

// GenComment renders a one-line summary like "dav.example.com:443 (TLS)" or
// "dav.example.com:443 (TLS) + dav.example.com:80 (plain)" so list views can
// show what the service points to without expanding.
func (s *CalDAV) GenComment() string {
	protoOf := func(name string) string {
		switch {
		case strings.HasPrefix(name, "_caldavs._tcp."):
			return "TLS"
		case strings.HasPrefix(name, "_caldav._tcp."):
			return "plain"
		}
		return ""
	}

	type entry struct {
		target string
		ports  []uint16
		protos map[string]bool
	}
	byTarget := map[string]*entry{}
	var order []string

	for _, srv := range s.Records {
		e, ok := byTarget[srv.Target]
		if !ok {
			e = &entry{target: srv.Target, protos: map[string]bool{}}
			byTarget[srv.Target] = e
			order = append(order, srv.Target)
		}
		if p := protoOf(srv.Hdr.Name); p != "" {
			e.protos[p] = true
		}
		seen := false
		for _, pp := range e.ports {
			if pp == srv.Port {
				seen = true
				break
			}
		}
		if !seen {
			e.ports = append(e.ports, srv.Port)
		}
	}

	var buf bytes.Buffer
	for i, tgt := range order {
		if i > 0 {
			buf.WriteString(" + ")
		}
		e := byTarget[tgt]
		buf.WriteString(strings.TrimSuffix(tgt, "."))
		buf.WriteString(":")
		for j, p := range e.ports {
			if j > 0 {
				buf.WriteString(",")
			}
			buf.WriteString(strconv.Itoa(int(p)))
		}
		if len(e.protos) > 0 {
			protos := make([]string, 0, len(e.protos))
			// Stable order: TLS before plain.
			for _, p := range []string{"TLS", "plain"} {
				if e.protos[p] {
					protos = append(protos, p)
				}
			}
			buf.WriteString(" (")
			buf.WriteString(strings.Join(protos, "/"))
			buf.WriteString(")")
		}
	}
	return buf.String()
}

func (s *CalDAV) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rrs := make([]happydns.Record, 0, len(s.Records)+len(s.Paths))
	for _, srv := range s.Records {
		rrs = append(rrs, srv)
	}
	for _, txt := range s.Paths {
		rrs = append(rrs, txt)
	}
	return rrs, nil
}

func caldav_analyze(a *svc.Analyzer) error {
	caldavDomains := map[string]*CalDAV{}

	for _, prefix := range caldavPrefixes {
		for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Prefix: prefix, Type: dns.TypeSRV}) {
			domain := strings.TrimPrefix(record.Header().Name, prefix)

			srv, ok := record.(*dns.SRV)
			if !ok {
				continue
			}

			if _, exists := caldavDomains[domain]; !exists {
				caldavDomains[domain] = &CalDAV{}
			}

			caldavDomains[domain].Records = append(
				caldavDomains[domain].Records,
				helpers.RRRelativeSubdomain(srv, a.GetOrigin(), domain).(*dns.SRV),
			)

			if err := a.UseRR(srv, domain, caldavDomains[domain]); err != nil {
				return err
			}
		}
	}

	// RFC 6764 §4: a companion TXT at the same label may advertise a
	// context path via `path=...`. Consume it only when an SRV for the
	// same domain was already registered, so stray TXT don't spawn an
	// SRV-less CalDAV service.
	for _, prefix := range caldavPrefixes {
		for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Prefix: prefix, Type: dns.TypeTXT}) {
			domain := strings.TrimPrefix(record.Header().Name, prefix)

			cd, ok := caldavDomains[domain]
			if !ok {
				continue
			}

			txt, ok := record.(*happydns.TXT)
			if !ok {
				continue
			}

			if !strings.HasPrefix(strings.TrimSpace(txt.Txt), "path=") {
				continue
			}

			cd.Paths = append(
				cd.Paths,
				helpers.RRRelativeSubdomain(txt, a.GetOrigin(), domain).(*happydns.TXT),
			)

			if err := a.UseRR(record, domain, cd); err != nil {
				return err
			}
		}
	}
	return nil
}

func init() {
	svc.RegisterService(
		func() happydns.ServiceBody {
			return &CalDAV{}
		},
		caldav_analyze,
		happydns.ServiceInfos{
			Name:        "CalDAV (Calendar)",
			Description: "Announce a CalDAV calendar server for the domain via SRV records (RFC 6764).",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"service",
				"groupware",
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				Single:    true,
				NeedTypes: []uint16{
					dns.TypeSRV,
					dns.TypeTXT,
				},
			},
		},
		1,
	)
}
