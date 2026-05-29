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

package dnschecker

import (
	"context"

	sdk "git.happydns.org/checker-sdk-go/checker"
	"git.happydns.org/happyDomain/model"
)

// PrecheckResult is the combined config-time verdict for a checker against
// a target's effective options. Failures is the per-rule precheck map (keys
// are rule names whose prerequisites are unmet, values the human-readable
// reasons; rules that pass — or do not implement RulePrecheck — are absent).
//
// Eligible / EligibilityReason carry the whole-checker eligibility verdict
// from the provider's optional CheckEnabler, with the SDK's fail-open
// semantics:
//   - Eligible == nil: the checker does not implement CheckEnabler, or
//     IsEligible could not determine eligibility (transient error). The host
//     keeps the checker. EligibilityReason, when set in this state, is a
//     diagnostic for the undetermined lookup, not a definitive skip.
//   - Eligible == &true: applicable to this target.
//   - Eligible == &false: not applicable; EligibilityReason explains why and
//     the host should hide/skip the checker.
type PrecheckResult struct {
	Failures          map[string]string
	Eligible          *bool
	EligibilityReason string
}

// CheckerHasEligibilityGate reports whether def could ever yield a definitive
// eligibility verdict (Eligible == &false) from EvaluateChecker. That happens
// only when the checker is delegated to a remote endpoint that is actually
// configured (the remote /definition can report eligibility), or when one of
// its observation providers implements CheckEnabler. When this returns false,
// EvaluateChecker always leaves Eligible nil (fail open), so callers can skip
// the evaluation, and the option building it requires, entirely.
//
// hasEndpoint must report whether the checker's "endpoint" admin option is
// actually set to a non-empty value (see CheckerOptionsUsecase.HasAdminEndpoint).
// Merely carrying the "endpoint" AdminOpt is not enough: RegisterExternalizableChecker
// adds it to nearly every built-in checker regardless of whether an
// administrator configured a remote URL.
func CheckerHasEligibilityGate(def *happydns.CheckerDefinition, hasEndpoint bool) bool {
	if def == nil {
		return false
	}

	if hasEndpoint {
		return true
	}

	for _, key := range def.ObservationKeys {
		if _, ok := sdk.FindObservationProvider(key).(sdk.CheckEnabler); ok {
			return true
		}
	}

	return false
}

// EvaluateChecker returns the per-rule precheck failures and the
// whole-checker eligibility verdict for def given the merged opts. The opts
// must be autofilled the same way Collect receives them (domain_name / zone /
// service), since both rule prechecks and CheckEnabler.IsEligible read those.
//
// Dispatch is based on the "endpoint" AdminOpt added by
// RegisterExternalizableChecker: when set, the call is forwarded to the
// remote checker's POST /definition (which returns failures and eligibility
// in a single round trip); otherwise the rules and providers are inspected
// in-process. A non-nil error means the evaluation itself could not run
// (typically a remote endpoint that is unreachable); callers should treat
// that as "no precheck information available", leave the rule list
// interactive, and fail open on eligibility.
func EvaluateChecker(ctx context.Context, def *happydns.CheckerDefinition, opts happydns.CheckerOptions) (PrecheckResult, error) {
	if def == nil {
		return PrecheckResult{}, nil
	}

	if endpoint, ok := opts["endpoint"].(string); ok && endpoint != "" {
		// Observation key is not used by the precheck HTTP call itself,
		// but HTTPObservationProvider's error messages reference it; pick
		// the first registered key when available so logs stay
		// identifiable.
		var key happydns.ObservationKey
		if len(def.ObservationKeys) > 0 {
			key = def.ObservationKeys[0]
		}
		return NewHTTPObservationProvider(key, endpoint).Precheck(ctx, opts)
	}

	result := PrecheckResult{}

	failures := map[string]string{}
	for _, rule := range def.Rules {
		pc, ok := rule.(happydns.RulePrecheck)
		if !ok {
			continue
		}
		if err := pc.Precheck(ctx, opts); err != nil {
			failures[rule.Name()] = err.Error()
		}
	}
	if len(failures) > 0 {
		result.Failures = failures
	}

	// Whole-checker eligibility: the first observation provider implementing
	// CheckEnabler decides. It is rule-independent, so it runs even for
	// checkers with no rules.
	for _, key := range def.ObservationKeys {
		provider := sdk.FindObservationProvider(key)
		enabler, ok := provider.(sdk.CheckEnabler)
		if !ok {
			continue
		}
		eligible, reason, err := enabler.IsEligible(ctx, opts)
		if err != nil {
			// Undetermined: fail open (Eligible stays nil), keep the
			// reason as a diagnostic.
			result.EligibilityReason = err.Error()
		} else {
			result.Eligible = &eligible
			result.EligibilityReason = reason
		}
		break
	}

	return result, nil
}
