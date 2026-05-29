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
	"errors"
	"fmt"
	"time"

	"git.happydns.org/happyDomain/internal/dnschecker"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/pkg/domaininfo"
)

// ObservationKeyAvailability is the observation key for domain availability data.
const ObservationKeyAvailability happydns.ObservationKey = "availability"

// AvailabilityData represents domain availability observation data.
type AvailabilityData struct {
	Available  bool       `json:"available"`
	Registrar  string     `json:"registrar,omitempty"`
	ExpiryDate *time.Time `json:"expiryDate,omitempty"`
}

// availabilityProvider collects whether a domain name is currently registered.
type availabilityProvider struct{}

func (p *availabilityProvider) Key() happydns.ObservationKey {
	return ObservationKeyAvailability
}

func (p *availabilityProvider) Collect(ctx context.Context, opts happydns.CheckerOptions) (any, error) {
	domainName, _ := opts["domainName"].(string)
	if domainName == "" {
		return nil, fmt.Errorf("domainName is required")
	}

	info, err := domaininfo.GetDomainInfo(ctx, happydns.Origin(domainName))
	if err != nil {
		if errors.Is(err, happydns.ErrDomainDoesNotExist) {
			return &AvailabilityData{Available: true}, nil
		}
		return nil, fmt.Errorf("failed to retrieve domain info: %w", err)
	}

	return &AvailabilityData{
		Available:  false,
		Registrar:  info.Registrar,
		ExpiryDate: info.ExpirationDate,
	}, nil
}

// domainAvailabilityRule emits a notify-worthy status when a watched domain
// becomes available for registration. The status is inverted relative to the
// usual convention (Crit when available) so the registered->available
// transition crosses the notification threshold and the dispatcher fires once.
type domainAvailabilityRule struct{}

func (r *domainAvailabilityRule) Name() string {
	return "domain_availability_check"
}

func (r *domainAvailabilityRule) Description() string {
	return "Checks whether a watched domain name has become available for registration"
}

func (r *domainAvailabilityRule) ValidateOptions(opts happydns.CheckerOptions) error {
	return nil
}

func (r *domainAvailabilityRule) Evaluate(ctx context.Context, obs happydns.ObservationGetter, opts happydns.CheckerOptions) []happydns.CheckState {
	var data AvailabilityData
	if err := obs.Get(ctx, ObservationKeyAvailability, &data); err != nil {
		return []happydns.CheckState{{
			Status:  happydns.StatusError,
			Message: fmt.Sprintf("Failed to get availability data: %v", err),
			Code:    "availability_error",
		}}
	}

	domainName, _ := opts["domainName"].(string)

	if data.Available {
		return []happydns.CheckState{{
			Status:  happydns.StatusCrit,
			Message: fmt.Sprintf("Domain %s is now available for registration", domainName),
			Code:    "available",
		}}
	}

	meta := map[string]any{}
	if data.Registrar != "" {
		meta["registrar"] = data.Registrar
	}
	if data.ExpiryDate != nil {
		meta["expiry_date"] = data.ExpiryDate
	}

	return []happydns.CheckState{{
		Status:  happydns.StatusOK,
		Message: fmt.Sprintf("Domain %s is still registered", domainName),
		Code:    "registered",
		Meta:    meta,
	}}
}

func init() {
	dnschecker.RegisterObservationProvider(&availabilityProvider{})

	dnschecker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   happydns.DomainAvailabilityCheckerID,
		Name: "Domain Availability",
		// All Availability flags are left false so IsAutoScheduled never
		// schedules this checker on managed domains. It is scheduled only via
		// the dedicated availability-watch enumeration in the scheduler.
		Availability:    happydns.CheckerAvailability{},
		ObservationKeys: []happydns.ObservationKey{ObservationKeyAvailability},
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{
					Id:       "domainName",
					Type:     "string",
					AutoFill: happydns.AutoFillDomainName,
					Hide:     true,
				},
			},
		},
		Rules: []happydns.CheckRule{
			&domainAvailabilityRule{},
		},
		Interval: &happydns.CheckIntervalSpec{
			Min:     1 * time.Hour,
			Max:     24 * time.Hour,
			Default: 6 * time.Hour,
		},
	})
}
