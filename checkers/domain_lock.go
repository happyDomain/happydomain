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

package checkers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"git.happydns.org/happyDomain/internal/checker"
	"git.happydns.org/happyDomain/model"
)

const defaultRequiredLockStatuses = "clientTransferProhibited"

// domainLockRule verifies that a domain carries the expected EPP lock
// statuses (e.g. clientTransferProhibited) as reported by RDAP/WHOIS.
type domainLockRule struct{}

func (r *domainLockRule) Name() string {
	return "domain_lock_check"
}

func (r *domainLockRule) Description() string {
	return "Verifies that a domain carries the expected EPP lock statuses (e.g. clientTransferProhibited)"
}

func (r *domainLockRule) ValidateOptions(opts happydns.CheckerOptions) error {
	v, ok := opts["requiredStatuses"]
	if !ok {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("requiredStatuses must be a string")
	}
	for _, p := range strings.Split(s, ",") {
		if strings.TrimSpace(p) != "" {
			return nil
		}
	}
	return fmt.Errorf("requiredStatuses must contain at least one EPP status code")
}

func (r *domainLockRule) Evaluate(ctx context.Context, obs happydns.ObservationGetter, opts happydns.CheckerOptions) happydns.CheckState {
	var whois WHOISData
	if err := obs.Get(ctx, ObservationKeyWhois, &whois); err != nil {
		return happydns.CheckState{
			Status:  happydns.StatusError,
			Message: fmt.Sprintf("Failed to get WHOIS data: %v", err),
			Code:    "lock_error",
		}
	}

	requiredStr := defaultRequiredLockStatuses
	if v, ok := opts["requiredStatuses"].(string); ok && v != "" {
		requiredStr = v
	}

	var required []string
	for _, s := range strings.Split(requiredStr, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			required = append(required, s)
		}
	}

	if len(required) == 0 {
		return happydns.CheckState{
			Status:  happydns.StatusUnknown,
			Message: "No required lock statuses configured",
			Code:    "lock_skipped",
		}
	}

	present := make(map[string]bool, len(whois.Status))
	for _, s := range whois.Status {
		present[strings.ToLower(s)] = true
	}

	var missing []string
	for _, req := range required {
		if !present[strings.ToLower(req)] {
			missing = append(missing, req)
		}
	}

	if len(missing) > 0 {
		return happydns.CheckState{
			Status:  happydns.StatusCrit,
			Message: fmt.Sprintf("Missing lock status: %s", strings.Join(missing, ", ")),
			Code:    "lock_missing",
			Meta: map[string]any{
				"missing": missing,
				"present": whois.Status,
			},
		}
	}

	return happydns.CheckState{
		Status:  happydns.StatusOK,
		Message: fmt.Sprintf("All required statuses present: %s", strings.Join(required, ", ")),
		Code:    "lock_ok",
		Meta: map[string]any{
			"required": required,
		},
	}
}

func init() {
	checker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   "domain_lock",
		Name: "Domain Lock Status",
		Availability: happydns.CheckerAvailability{
			ApplyToDomain: true,
		},
		ObservationKeys: []happydns.ObservationKey{ObservationKeyWhois},
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{
					Id:       "domainName",
					Type:     "string",
					AutoFill: happydns.AutoFillDomainName,
					Hide:     true,
				},
				{
					Id:          "requiredStatuses",
					Type:        "string",
					Label:       "Required lock statuses",
					Description: "Comma-separated list of EPP status codes that must be present on the domain (e.g. clientTransferProhibited, clientUpdateProhibited, clientDeleteProhibited).",
					Default:     defaultRequiredLockStatuses,
					Placeholder: defaultRequiredLockStatuses,
				},
			},
		},
		Rules: []happydns.CheckRule{
			&domainLockRule{},
		},
		Interval: &happydns.CheckIntervalSpec{
			Min:     1 * time.Hour,
			Max:     7 * 24 * time.Hour,
			Default: 24 * time.Hour,
		},
	})
}
