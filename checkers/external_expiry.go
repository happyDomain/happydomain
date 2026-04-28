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
	"encoding/json"
	"fmt"
	"sync"
	"time"

	dangling "git.happydns.org/checker-dangling/contract"
	sdk "git.happydns.org/checker-sdk-go/checker"
	"git.happydns.org/happyDomain/internal/checker"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/pkg/domaininfo"
)

// ObservationKeyExternalWhois is the observation key produced by the
// external_whois checker. It mirrors the constant by the same name in
// checker-dangling/checker so consumers and producers agree.
const ObservationKeyExternalWhois happydns.ObservationKey = "external_whois"

// maxConcurrentRDAP caps parallel RDAP/WHOIS lookups per Collect call
// so a wide zone with many distinct registrables does not overwhelm
// the upstream registry. The value matches checker-tls's analogous
// MaxConcurrentProbes; it can be tuned via configuration later.
const maxConcurrentRDAP = 8

// ExternalWhoisData is the per-Ref index of WHOIS facts the
// external_whois checker publishes. The shape mirrors checker-tls's
// TLSData.Probes: a single observation with one entry per discovery
// Ref so the host can pivot it into RelatedObservation.Ref.
type ExternalWhoisData struct {
	// Facts is keyed by the DiscoveryEntry.Ref of the dangling
	// external-target.v1 entry the WHOIS lookup covers.
	Facts       map[string]ExternalWhoisFacts `json:"facts"`
	CollectedAt time.Time                     `json:"collected_at"`
}

// ExternalWhoisFacts is the subset of WHOIS data checker-dangling
// consumes. Kept narrow on purpose — the canonical, full WHOIS
// observation lives under domain_expiry's own ObservationKeyWhois.
type ExternalWhoisFacts struct {
	Registrable  string    `json:"registrable"`
	ExpiryDate   time.Time `json:"expiryDate"`
	CreationDate time.Time `json:"creationDate,omitzero"`
	Registrar    string    `json:"registrar,omitempty"`
	Status       []string  `json:"status,omitempty"`
	Error        string    `json:"error,omitempty"`
}

// externalWhoisProvider subscribes to dangling.external-target.v1
// DiscoveryEntries and publishes one WHOIS fact set per registrable
// domain referenced by those entries. Two entries pointing at the
// same registrable share one lookup.
type externalWhoisProvider struct{}

func (p *externalWhoisProvider) Key() happydns.ObservationKey {
	return ObservationKeyExternalWhois
}

func (p *externalWhoisProvider) Collect(ctx context.Context, opts happydns.CheckerOptions) (any, error) {
	raw, _ := sdk.GetOption[[]sdk.DiscoveryEntry](opts, "discovery_entries")

	jobs := make([]rdapJob, 0, len(raw))
	// Cache one job per (Ref, registrable) pair so we still produce a
	// per-Ref entry in Facts even when several Refs share the same
	// registrable (we resolve the registrable once below).
	for _, e := range raw {
		if e.Type != dangling.ExternalTargetType {
			continue
		}
		var t dangling.ExternalTarget
		if err := json.Unmarshal(e.Payload, &t); err != nil {
			continue
		}
		if t.Registrable == "" {
			continue
		}
		jobs = append(jobs, rdapJob{ref: e.Ref, registrable: t.Registrable})
	}

	if len(jobs) == 0 {
		return &ExternalWhoisData{
			Facts:       map[string]ExternalWhoisFacts{},
			CollectedAt: time.Now().UTC(),
		}, nil
	}

	// Resolve unique registrables once, reuse the result for every Ref.
	cache := map[string]ExternalWhoisFacts{}
	var cacheMu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentRDAP)

	uniq := uniqueRegistrables(jobs)
dispatch:
	for _, reg := range uniq {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break dispatch
		}
		wg.Add(1)
		go func(reg string) {
			defer wg.Done()
			defer func() { <-sem }()
			facts := lookupRegistrable(ctx, reg)
			cacheMu.Lock()
			cache[reg] = facts
			cacheMu.Unlock()
		}(reg)
	}
	wg.Wait()

	out := make(map[string]ExternalWhoisFacts, len(jobs))
	for _, j := range jobs {
		if f, ok := cache[j.registrable]; ok {
			out[j.ref] = f
		}
	}

	return &ExternalWhoisData{
		Facts:       out,
		CollectedAt: time.Now().UTC(),
	}, nil
}

type rdapJob struct {
	ref         string
	registrable string
}

func uniqueRegistrables(jobs []rdapJob) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(jobs))
	for _, j := range jobs {
		if seen[j.registrable] {
			continue
		}
		seen[j.registrable] = true
		out = append(out, j.registrable)
	}
	return out
}

// lookupRegistrable wraps domaininfo.GetDomainInfo so a single failed
// lookup is captured in ExternalWhoisFacts.Error instead of propagating
// (we want partial results when most lookups succeed).
func lookupRegistrable(ctx context.Context, registrable string) ExternalWhoisFacts {
	out := ExternalWhoisFacts{Registrable: registrable}
	info, err := domaininfo.GetDomainInfo(ctx, happydns.Origin(registrable))
	if err != nil {
		out.Error = err.Error()
		return out
	}
	if info.ExpirationDate != nil {
		out.ExpiryDate = *info.ExpirationDate
	}
	if info.CreationDate != nil {
		out.CreationDate = *info.CreationDate
	}
	out.Registrar = info.Registrar
	out.Status = info.Status
	return out
}

// externalWhoisRule emits a coarse status purely so the host has a
// rule to bind: the actionable verdicts live in checker-dangling,
// which fetches our facts via GetRelated. We surface the count of
// successful lookups vs failures here.
type externalWhoisRule struct{}

func (r *externalWhoisRule) Name() string { return "external_whois_collected" }
func (r *externalWhoisRule) Description() string {
	return "Reports how many external pointer targets had their registrable WHOIS/RDAP record successfully retrieved. Verdicts about expiration, redemption, or recent re-registration are surfaced by checker-dangling."
}
func (r *externalWhoisRule) Evaluate(ctx context.Context, obs happydns.ObservationGetter, opts happydns.CheckerOptions) []happydns.CheckState {
	var data ExternalWhoisData
	if err := obs.Get(ctx, ObservationKeyExternalWhois, &data); err != nil {
		return []happydns.CheckState{{
			Status:  happydns.StatusError,
			Message: fmt.Sprintf("Failed to read external_whois: %v", err),
			Code:    "external_whois_error",
		}}
	}
	total := len(data.Facts)
	failed := 0
	for _, f := range data.Facts {
		if f.Error != "" {
			failed++
		}
	}
	if total == 0 {
		return []happydns.CheckState{{
			Status:  happydns.StatusInfo,
			Message: "No external pointer target was reported by checker-dangling.",
			Code:    "external_whois_empty",
		}}
	}
	if failed == total {
		return []happydns.CheckState{{
			Status:  happydns.StatusWarn,
			Message: fmt.Sprintf("WHOIS lookup failed for all %d external target(s).", total),
			Code:    "external_whois_all_failed",
			Meta:    map[string]any{"total": total, "failed": failed},
		}}
	}
	if failed > 0 {
		return []happydns.CheckState{{
			Status:  happydns.StatusInfo,
			Message: fmt.Sprintf("Collected WHOIS for %d/%d external target(s); %d lookup(s) failed.", total-failed, total, failed),
			Code:    "external_whois_partial",
			Meta:    map[string]any{"total": total, "failed": failed},
		}}
	}
	return []happydns.CheckState{{
		Status:  happydns.StatusOK,
		Message: fmt.Sprintf("Collected WHOIS for %d external target(s).", total),
		Code:    "external_whois_ok",
		Meta:    map[string]any{"total": total},
	}}
}

func init() {
	checker.RegisterObservationProvider(&externalWhoisProvider{})

	checker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   "external_whois",
		Name: "External target WHOIS",
		Availability: happydns.CheckerAvailability{
			ApplyToZone: true,
		},
		ObservationKeys: []happydns.ObservationKey{ObservationKeyExternalWhois},
		Options: happydns.CheckerOptionsDocumentation{
			RunOpts: []happydns.CheckerOptionDocumentation{
				{
					Id:       "discovery_entries",
					Type:     "array",
					Label:    "Discovery entries",
					AutoFill: happydns.AutoFillDiscoveryEntries,
					Hide:     true,
				},
			},
		},
		Rules: []happydns.CheckRule{
			&externalWhoisRule{},
		},
		Interval: &happydns.CheckIntervalSpec{
			Min:     12 * time.Hour,
			Max:     7 * 24 * time.Hour,
			Default: 24 * time.Hour,
		},
	})
}
