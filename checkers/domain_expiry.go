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

package checkers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	sdk "git.happydns.org/checker-sdk-go/checker"
	"git.happydns.org/happyDomain/internal/checker"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/pkg/domaininfo"
)

const (
	// ObservationKeyWhois is the observation key for WHOIS / domain expiry data.
	ObservationKeyWhois happydns.ObservationKey = "whois"

	defaultWarningDays  = 30
	defaultCriticalDays = 7
)

// WHOISData represents WHOIS observation data.
type WHOISData struct {
	ExpiryDate time.Time                          `json:"expiryDate"`
	Registrar  string                             `json:"registrar"`
	Contacts   map[string]*happydns.ContactInfo   `json:"contacts,omitempty"`
	Status     []string                           `json:"status,omitempty"`
}

// whoisProvider is a placeholder WHOIS observation provider.
type whoisProvider struct{}

func (p *whoisProvider) Key() happydns.ObservationKey {
	return ObservationKeyWhois
}

func (p *whoisProvider) Collect(ctx context.Context, opts happydns.CheckerOptions) (any, error) {
	domainName, _ := opts["domainName"].(string)
	if domainName == "" {
		return nil, fmt.Errorf("domainName is required")
	}

	info, err := domaininfo.GetDomainInfo(ctx, happydns.Origin(domainName))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve domain info: %w", err)
	}

	if info.ExpirationDate == nil {
		return nil, fmt.Errorf("expiration date not available for %s", domainName)
	}

	registrar := info.Registrar
	if registrar == "" {
		registrar = "Unknown"
	}

	return &WHOISData{
		ExpiryDate: *info.ExpirationDate,
		Registrar:  registrar,
		Contacts:   info.Contacts,
		Status:     info.Status,
	}, nil
}

// ExtractMetrics implements happydns.CheckerMetricsReporter.
func (p *whoisProvider) ExtractMetrics(ctx happydns.ReportContext, collectedAt time.Time) ([]happydns.CheckMetric, error) {
	var data WHOISData
	if err := json.Unmarshal(ctx.Data(), &data); err != nil {
		return nil, err
	}

	daysRemaining := data.ExpiryDate.Sub(collectedAt).Hours() / 24
	return []happydns.CheckMetric{{
		Name:      "domain_expiry_days_remaining",
		Value:     daysRemaining,
		Unit:      "days",
		Labels:    map[string]string{"registrar": data.Registrar},
		Timestamp: collectedAt,
	}}, nil
}

// domainExpiryRule checks whether a domain is nearing expiration.
type domainExpiryRule struct{}

func (r *domainExpiryRule) Name() string {
	return "domain_expiry_check"
}

func (r *domainExpiryRule) Description() string {
	return "Checks whether a domain name is nearing its expiration date"
}

func (r *domainExpiryRule) ValidateOptions(opts happydns.CheckerOptions) error {
	warningDays := float64(defaultWarningDays)
	criticalDays := float64(defaultCriticalDays)

	if v, ok := opts["warning_days"]; ok {
		d, ok := v.(float64)
		if !ok {
			return fmt.Errorf("warning_days must be a number")
		}
		if d <= 0 {
			return fmt.Errorf("warning_days must be positive")
		}
		warningDays = d
	}
	if v, ok := opts["critical_days"]; ok {
		d, ok := v.(float64)
		if !ok {
			return fmt.Errorf("critical_days must be a number")
		}
		if d <= 0 {
			return fmt.Errorf("critical_days must be positive")
		}
		criticalDays = d
	}

	if criticalDays >= warningDays {
		return fmt.Errorf("critical_days (%v) must be less than warning_days (%v)", criticalDays, warningDays)
	}

	return nil
}

func (r *domainExpiryRule) Evaluate(ctx context.Context, obs happydns.ObservationGetter, opts happydns.CheckerOptions) []happydns.CheckState {
	var whois WHOISData
	if err := obs.Get(ctx, ObservationKeyWhois, &whois); err != nil {
		return []happydns.CheckState{{
			Status:  happydns.StatusError,
			Message: fmt.Sprintf("Failed to get WHOIS data: %v", err),
			Code:    "whois_error",
		}}
	}

	// Read thresholds from options with defaults.
	warningDays := sdk.GetIntOption(opts, "warning_days", defaultWarningDays)
	criticalDays := sdk.GetIntOption(opts, "critical_days", defaultCriticalDays)

	daysRemaining := int(time.Until(whois.ExpiryDate).Hours() / 24)
	meta := map[string]any{"days_remaining": daysRemaining}

	if daysRemaining <= criticalDays {
		return []happydns.CheckState{{
			Status:  happydns.StatusCrit,
			Message: fmt.Sprintf("Domain expires in %d days", daysRemaining),
			Code:    "expiry_critical",
			Meta:    meta,
		}}
	}

	if daysRemaining <= warningDays {
		return []happydns.CheckState{{
			Status:  happydns.StatusWarn,
			Message: fmt.Sprintf("Domain expires in %d days", daysRemaining),
			Code:    "expiry_warning",
			Meta:    meta,
		}}
	}

	return []happydns.CheckState{{
		Status:  happydns.StatusOK,
		Message: fmt.Sprintf("Domain expires in %d days", daysRemaining),
		Code:    "expiry_ok",
		Meta:    meta,
	}}
}

func init() {
	checker.RegisterObservationProvider(&whoisProvider{})

	checker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   "domain_expiry",
		Name: "Domain Expiry",
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
					Id:          "warning_days",
					Type:        "uint",
					Label:       "Warning threshold (days)",
					Description: "Number of days before expiration to trigger a warning.",
					Default:     defaultWarningDays,
					Placeholder: strconv.Itoa(defaultWarningDays),
				},
				{
					Id:          "critical_days",
					Type:        "uint",
					Label:       "Critical threshold (days)",
					Description: "Number of days before expiration to trigger a critical alert.",
					Default:     defaultCriticalDays,
					Placeholder: strconv.Itoa(defaultCriticalDays),
				},
			},
		},
		Rules: []happydns.CheckRule{
			&domainExpiryRule{},
		},
		Interval: &happydns.CheckIntervalSpec{
			Min:     12 * time.Hour,
			Max:     7 * 24 * time.Hour,
			Default: 24 * time.Hour,
		},
		HasMetrics: true,
	})
}
