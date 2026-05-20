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

	"git.happydns.org/happyDomain/model"
)

// PrecheckRules returns the per-rule precheck failure map for def given
// the merged opts: keys are rule names whose prerequisites are unmet,
// values are the human-readable reasons. Rules that pass — or do not
// implement RulePrecheck — are absent from the map.
//
// Dispatch is based on the "endpoint" AdminOpt added by
// RegisterExternalizableChecker: when set, the call is forwarded to the
// remote checker's POST /definition; otherwise the rules are inspected
// in-process. A non-nil error means the precheck itself could not run
// (typically a remote endpoint that is unreachable); callers should
// treat that as "no precheck information available" and leave the
// rule list interactive.
func PrecheckRules(ctx context.Context, def *happydns.CheckerDefinition, opts happydns.CheckerOptions) (map[string]string, error) {
	if def == nil || len(def.Rules) == 0 {
		return nil, nil
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
	return failures, nil
}
