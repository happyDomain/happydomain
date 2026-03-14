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

package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	svcs "git.happydns.org/happyDomain/services"
)

func init() {
	RegisterChecker("deprecated-records", &DeprecatedRecordCheck{})
}

// deprecatedTypes maps DNS record type numbers to a human-readable reason.
var deprecatedTypes = map[uint16]string{
	dns.TypeSPF: "RFC 7208: use TXT instead",
	38:          "RFC 6563: use AAAA instead",  // A6
	dns.TypeNXT: "RFC 3755: use NSEC instead",
	dns.TypeSIG: "RFC 3755: use RRSIG instead",
	dns.TypeKEY: "RFC 3755: use DNSKEY instead",
	11:          "deprecated, not widely used", // WKS
}

// DeprecatedRecordFinding describes a single deprecated record type found.
type DeprecatedRecordFinding struct {
	TypeName string `json:"type"`
	Reason   string `json:"reason"`
}

type DeprecatedRecordCheck struct{}

func (d *DeprecatedRecordCheck) ID() string {
	return "deprecated-records"
}

func (d *DeprecatedRecordCheck) Name() string {
	return "Deprecated DNS Record Types"
}

func (d *DeprecatedRecordCheck) Availability() happydns.CheckerAvailability {
	return happydns.CheckerAvailability{
		ApplyToService: true,
	}
}

func (d *DeprecatedRecordCheck) Options() happydns.CheckerOptionsDocumentation {
	return happydns.CheckerOptionsDocumentation{
		RunOpts: []happydns.CheckerOptionDocumentation{
			{
				Id:       "service",
				Label:    "Service",
				AutoFill: happydns.AutoFillService,
			},
		},
	}
}

func (d *DeprecatedRecordCheck) RunCheck(ctx context.Context, options happydns.CheckerOptions, meta map[string]string) (*happydns.CheckResult, error) {
	service, ok := options["service"].(*happydns.ServiceMessage)
	if !ok {
		return nil, fmt.Errorf("service not defined")
	}

	svcBody, err := svcs.FindService(service.Type)
	if err != nil {
		return nil, fmt.Errorf("unknown service type %q: %w", service.Type, err)
	}

	if err := json.Unmarshal(service.Service, &svcBody); err != nil {
		return nil, fmt.Errorf("failed to decode service: %w", err)
	}

	records, err := svcBody.GetRecords(service.Domain, service.Ttl, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get records: %w", err)
	}

	// Collect unique deprecated types found.
	seen := map[uint16]bool{}
	var findings []DeprecatedRecordFinding
	for _, rr := range records {
		rrtype := rr.Header().Rrtype
		if reason, deprecated := deprecatedTypes[rrtype]; deprecated && !seen[rrtype] {
			seen[rrtype] = true
			findings = append(findings, DeprecatedRecordFinding{
				TypeName: dns.TypeToString[rrtype],
				Reason:   reason,
			})
		}
	}

	if len(findings) == 0 {
		return &happydns.CheckResult{
			Status:     happydns.CheckResultStatusOK,
			StatusLine: "No deprecated record types found",
			Report:     findings,
		}, nil
	}

	typeNames := make([]string, len(findings))
	for i, f := range findings {
		typeNames[i] = f.TypeName
	}
	return &happydns.CheckResult{
		Status:     happydns.CheckResultStatusWarn,
		StatusLine: "Deprecated record types found: " + strings.Join(typeNames, ", "),
		Report:     findings,
	}, nil
}
