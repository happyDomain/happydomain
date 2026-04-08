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

// redactedPatterns is the list of substrings that, when found in a contact
// field, indicate the data is privacy-protected rather than meaningful.
var redactedPatterns = []string{
	"redacted",
	"privacy",
	"not disclosed",
	"whoisguard",
	"withheld",
	"data protected",
	"contact privacy",
}

// domainContactRule compares the registered domain contacts (registrant,
// admin, tech) against user-supplied expected values, with redaction
// detection for privacy-protected domains.
type domainContactRule struct{}

func (r *domainContactRule) Name() string {
	return "domain_contact_check"
}

func (r *domainContactRule) Description() string {
	return "Verifies that domain contacts (name/organization/email) match expected values"
}

// validRoles enumerates the contact roles supported by the WHOIS observation.
var validRoles = map[string]bool{
	"registrant": true,
	"admin":      true,
	"tech":       true,
}

func (r *domainContactRule) ValidateOptions(opts happydns.CheckerOptions) error {
	for _, key := range []string{"expectedName", "expectedOrganization", "expectedEmail", "checkRoles"} {
		if v, ok := opts[key]; ok {
			if _, ok := v.(string); !ok {
				return fmt.Errorf("%s must be a string", key)
			}
		}
	}

	if v, ok := opts["checkRoles"].(string); ok && v != "" {
		hasOne := false
		for _, p := range strings.Split(v, ",") {
			role := strings.TrimSpace(p)
			if role == "" {
				continue
			}
			if !validRoles[role] {
				return fmt.Errorf("checkRoles: unknown role %q (allowed: registrant, admin, tech)", role)
			}
			hasOne = true
		}
		if !hasOne {
			return fmt.Errorf("checkRoles must contain at least one role")
		}
	}

	return nil
}

func (r *domainContactRule) Evaluate(ctx context.Context, obs happydns.ObservationGetter, opts happydns.CheckerOptions) happydns.CheckState {
	var whois WHOISData
	if err := obs.Get(ctx, ObservationKeyWhois, &whois); err != nil {
		return happydns.CheckState{
			Status:  happydns.StatusError,
			Message: fmt.Sprintf("Failed to get WHOIS data: %v", err),
			Code:    "contact_error",
		}
	}

	expectedName, _ := opts["expectedName"].(string)
	expectedOrg, _ := opts["expectedOrganization"].(string)
	expectedEmail, _ := opts["expectedEmail"].(string)

	if expectedName == "" && expectedOrg == "" && expectedEmail == "" {
		return happydns.CheckState{
			Status:  happydns.StatusUnknown,
			Message: "No expected contact values configured",
			Code:    "contact_skipped",
		}
	}

	checkRolesStr := "registrant"
	if v, ok := opts["checkRoles"].(string); ok && v != "" {
		checkRolesStr = v
	}

	var roles []string
	for s := range strings.SplitSeq(checkRolesStr, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			roles = append(roles, s)
		}
	}
	if len(roles) == 0 {
		return happydns.CheckState{
			Status:  happydns.StatusUnknown,
			Message: "No contact roles to check",
			Code:    "contact_skipped",
		}
	}

	worst := happydns.StatusOK
	var lines []string

	for _, role := range roles {
		contact, found := whois.Contacts[role]
		if !found || contact == nil {
			lines = append(lines, fmt.Sprintf("%s: contact not found", role))
			worst = worseStatus(worst, happydns.StatusWarn)
			continue
		}

		if isRedacted(contact) {
			lines = append(lines, fmt.Sprintf("%s: contact info is redacted/private", role))
			worst = worseStatus(worst, happydns.StatusInfo)
			continue
		}

		var mismatches []string
		if expectedName != "" && !strings.EqualFold(expectedName, contact.Name) {
			mismatches = append(mismatches, fmt.Sprintf("name: got %q, expected %q", contact.Name, expectedName))
		}
		if expectedOrg != "" && !strings.EqualFold(expectedOrg, contact.Organization) {
			mismatches = append(mismatches, fmt.Sprintf("organization: got %q, expected %q", contact.Organization, expectedOrg))
		}
		if expectedEmail != "" && !strings.EqualFold(expectedEmail, contact.Email) {
			mismatches = append(mismatches, fmt.Sprintf("email: got %q, expected %q", contact.Email, expectedEmail))
		}

		if len(mismatches) > 0 {
			lines = append(lines, fmt.Sprintf("%s: %s", role, strings.Join(mismatches, ", ")))
			worst = worseStatus(worst, happydns.StatusWarn)
		} else {
			lines = append(lines, fmt.Sprintf("%s: contact info matches", role))
		}
	}

	return happydns.CheckState{
		Status:  worst,
		Message: strings.Join(lines, "; "),
		Code:    "contact_result",
	}
}

// isRedacted reports whether a contact's fields look privacy-protected.
func isRedacted(c *happydns.ContactInfo) bool {
	for _, field := range []string{c.Name, c.Organization, c.Email} {
		lower := strings.ToLower(field)
		for _, pattern := range redactedPatterns {
			if strings.Contains(lower, pattern) {
				return true
			}
		}
	}
	return false
}

// worseStatus returns the more severe of two statuses. The SDK orders
// statuses as Unknown < OK < Info < Warn < Crit < Error, so the higher
// numeric value is the more severe one.
func worseStatus(a, b happydns.Status) happydns.Status {
	if b > a {
		return b
	}
	return a
}

func init() {
	checker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   "domain_contact",
		Name: "Domain Contact Consistency",
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
					Id:          "expectedName",
					Type:        "string",
					Label:       "Expected registrant name",
					Description: "If set, the configured roles must report this exact name (case-insensitive).",
				},
				{
					Id:          "expectedOrganization",
					Type:        "string",
					Label:       "Expected organization",
					Description: "If set, the configured roles must report this exact organization (case-insensitive).",
				},
				{
					Id:          "expectedEmail",
					Type:        "string",
					Label:       "Expected email",
					Description: "If set, the configured roles must report this exact email (case-insensitive).",
				},
				{
					Id:          "checkRoles",
					Type:        "string",
					Label:       "Contact roles to check",
					Description: "Comma-separated list of roles among: registrant, admin, tech.",
					Default:     "registrant",
					Placeholder: "registrant",
				},
			},
		},
		Rules: []happydns.CheckRule{
			&domainContactRule{},
		},
		Interval: &happydns.CheckIntervalSpec{
			Min:     1 * time.Hour,
			Max:     7 * 24 * time.Hour,
			Default: 24 * time.Hour,
		},
	})
}
