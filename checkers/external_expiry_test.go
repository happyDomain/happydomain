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
	"testing"

	"git.happydns.org/happyDomain/model"
)

func TestExternalWhoisRule_NoFacts(t *testing.T) {
	rule := &externalWhoisRule{}
	obs := &stubObservationGetter{key: ObservationKeyExternalWhois, data: &ExternalWhoisData{
		Facts: map[string]ExternalWhoisFacts{},
	}}
	states := rule.Evaluate(context.Background(), obs, nil)
	if len(states) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states))
	}
	if states[0].Status != happydns.StatusInfo || states[0].Code != "external_whois_empty" {
		t.Errorf("got %+v, want info/external_whois_empty", states[0])
	}
}

func TestExternalWhoisRule_AllFailed(t *testing.T) {
	rule := &externalWhoisRule{}
	obs := &stubObservationGetter{key: ObservationKeyExternalWhois, data: &ExternalWhoisData{
		Facts: map[string]ExternalWhoisFacts{
			"a": {Registrable: "a.example", Error: "rdap unreachable"},
			"b": {Registrable: "b.example", Error: "timeout"},
		},
	}}
	states := rule.Evaluate(context.Background(), obs, nil)
	if states[0].Status != happydns.StatusWarn || states[0].Code != "external_whois_all_failed" {
		t.Errorf("got %+v, want warn/external_whois_all_failed", states[0])
	}
}

func TestExternalWhoisRule_AllOK(t *testing.T) {
	rule := &externalWhoisRule{}
	obs := &stubObservationGetter{key: ObservationKeyExternalWhois, data: &ExternalWhoisData{
		Facts: map[string]ExternalWhoisFacts{
			"a": {Registrable: "a.example", Registrar: "Acme"},
		},
	}}
	states := rule.Evaluate(context.Background(), obs, nil)
	if states[0].Status != happydns.StatusOK || states[0].Code != "external_whois_ok" {
		t.Errorf("got %+v, want ok/external_whois_ok", states[0])
	}
}

func TestUniqueRegistrables_Dedupes(t *testing.T) {
	got := uniqueRegistrables([]rdapJob{
		{ref: "r1", registrable: "a.example"},
		{ref: "r2", registrable: "b.example"},
		{ref: "r3", registrable: "a.example"},
	})
	if len(got) != 2 {
		t.Errorf("expected 2 unique registrables, got %v", got)
	}
}
