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
	"errors"
	"testing"

	"git.happydns.org/happyDomain/model"
)

// enablerProvider is an ObservationProvider that also implements CheckEnabler,
// returning a fixed eligibility verdict for the tests.
type enablerProvider struct {
	key      happydns.ObservationKey
	eligible bool
	reason   string
	err      error
}

func (p *enablerProvider) Key() happydns.ObservationKey { return p.key }

func (p *enablerProvider) Collect(_ context.Context, _ happydns.CheckerOptions) (any, error) {
	return map[string]string{}, nil
}

func (p *enablerProvider) IsEligible(_ context.Context, _ happydns.CheckerOptions) (bool, string, error) {
	return p.eligible, p.reason, p.err
}

// plainProvider implements ObservationProvider but not CheckEnabler.
type plainProvider struct {
	key happydns.ObservationKey
}

func (p *plainProvider) Key() happydns.ObservationKey { return p.key }

func (p *plainProvider) Collect(_ context.Context, _ happydns.CheckerOptions) (any, error) {
	return map[string]string{}, nil
}

func TestEvaluateChecker_Eligibility(t *testing.T) {
	tests := []struct {
		name         string
		provider     happydns.ObservationProvider
		wantEligible *bool
		wantReason   string
	}{
		{
			name:         "eligible",
			provider:     &enablerProvider{key: "evalchk-elig", eligible: true},
			wantEligible: boolPtr(true),
		},
		{
			name:         "ineligible",
			provider:     &enablerProvider{key: "evalchk-inelig", reason: "not a reverse zone"},
			wantEligible: boolPtr(false),
			wantReason:   "not a reverse zone",
		},
		{
			name:         "undetermined",
			provider:     &enablerProvider{key: "evalchk-undet", err: errors.New("lookup failed")},
			wantEligible: nil,
			wantReason:   "lookup failed",
		},
		{
			name:         "not implemented",
			provider:     &plainProvider{key: "evalchk-plain"},
			wantEligible: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			RegisterObservationProvider(tc.provider)
			def := &happydns.CheckerDefinition{
				ID:              "evalchk-" + tc.name,
				ObservationKeys: []happydns.ObservationKey{tc.provider.Key()},
			}

			res, err := EvaluateChecker(context.Background(), def, happydns.CheckerOptions{})
			if err != nil {
				t.Fatalf("EvaluateChecker() error: %v", err)
			}

			if (res.Eligible == nil) != (tc.wantEligible == nil) {
				t.Fatalf("Eligible = %v, want %v", res.Eligible, tc.wantEligible)
			}
			if res.Eligible != nil && *res.Eligible != *tc.wantEligible {
				t.Fatalf("Eligible = %v, want %v", *res.Eligible, *tc.wantEligible)
			}
			if res.EligibilityReason != tc.wantReason {
				t.Fatalf("EligibilityReason = %q, want %q", res.EligibilityReason, tc.wantReason)
			}
		})
	}
}

func TestCheckerHasEligibilityGate(t *testing.T) {
	RegisterObservationProvider(&enablerProvider{key: "gate-enabler"})
	RegisterObservationProvider(&plainProvider{key: "gate-plain"})

	tests := []struct {
		name        string
		def         *happydns.CheckerDefinition
		hasEndpoint bool
		want        bool
	}{
		{
			name: "nil definition",
			def:  nil,
			want: false,
		},
		{
			name: "no observation provider with CheckEnabler",
			def:  &happydns.CheckerDefinition{ObservationKeys: []happydns.ObservationKey{"gate-plain"}},
			want: false,
		},
		{
			name: "observation provider implements CheckEnabler",
			def:  &happydns.CheckerDefinition{ObservationKeys: []happydns.ObservationKey{"gate-plain", "gate-enabler"}},
			want: true,
		},
		{
			name: "externalizable checker without a configured endpoint",
			def: &happydns.CheckerDefinition{
				ObservationKeys: []happydns.ObservationKey{"gate-plain"},
				Options: happydns.CheckerOptionsDocumentation{
					AdminOpts: []happydns.CheckerOptionDocumentation{{Id: "endpoint"}},
				},
			},
			hasEndpoint: false,
			want:        false,
		},
		{
			name: "externalizable checker with a configured endpoint",
			def: &happydns.CheckerDefinition{
				ObservationKeys: []happydns.ObservationKey{"gate-plain"},
				Options: happydns.CheckerOptionsDocumentation{
					AdminOpts: []happydns.CheckerOptionDocumentation{{Id: "endpoint"}},
				},
			},
			hasEndpoint: true,
			want:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CheckerHasEligibilityGate(tc.def, tc.hasEndpoint); got != tc.want {
				t.Fatalf("CheckerHasEligibilityGate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }
