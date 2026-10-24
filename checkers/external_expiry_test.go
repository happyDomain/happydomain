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
	"errors"
	"sync/atomic"
	"testing"
	"time"

	dangling "git.happydns.org/checker-dangling/contract"
	sdk "git.happydns.org/checker-sdk-go/checker"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/pkg/domaininfo"
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

func TestRegistrableFacts(t *testing.T) {
	expiry := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
	got := registrableFacts("a.example", domaininfo.Result{Info: &domaininfo.DomainInfo{
		ExpirationDate: &expiry,
		Registrar:      "Acme",
		Status:         []string{"pendingDelete"},
	}})
	if got.Registrable != "a.example" || !got.ExpiryDate.Equal(expiry) || got.Registrar != "Acme" || len(got.Status) != 1 || got.Error != "" {
		t.Errorf("got %+v", got)
	}

	got = registrableFacts("b.example", domaininfo.Result{Err: errString("timeout")})
	if got.Registrable != "b.example" || got.Error != "timeout" {
		t.Errorf("got %+v", got)
	}
}

func externalTargetEntry(t *testing.T, ref, registrable string) sdk.DiscoveryEntry {
	t.Helper()
	payload, err := json.Marshal(dangling.ExternalTarget{Registrable: registrable})
	if err != nil {
		t.Fatal(err)
	}
	return sdk.DiscoveryEntry{Type: dangling.ExternalTargetType, Ref: ref, Payload: payload}
}

func TestExternalWhoisProvider_Collect(t *testing.T) {
	expiry := time.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC)
	var lookups atomic.Int32
	p := &externalWhoisProvider{getInfo: func(ctx context.Context, name string) (*domaininfo.DomainInfo, error) {
		lookups.Add(1)
		if name == "broken.example" {
			return nil, errors.New("rdap unreachable")
		}
		return &domaininfo.DomainInfo{ExpirationDate: &expiry, Registrar: "Acme"}, nil
	}}

	entries := []sdk.DiscoveryEntry{
		externalTargetEntry(t, "cname", "shared.example"),
		externalTargetEntry(t, "mx", "shared.example"),
		externalTargetEntry(t, "ns", "broken.example"),
		externalTargetEntry(t, "no-registrable", ""),
		{Type: "other.v1", Ref: "other", Payload: json.RawMessage(`{"registrable":"other.example"}`)},
		{Type: dangling.ExternalTargetType, Ref: "bad-payload", Payload: json.RawMessage(`[`)},
	}
	got, err := p.Collect(context.Background(), happydns.CheckerOptions{"discovery_entries": entries})
	if err != nil {
		t.Fatal(err)
	}
	facts := got.(*ExternalWhoisData).Facts

	if n := lookups.Load(); n != 2 {
		t.Errorf("got %d lookups, want 2: the Refs sharing a registrable share its lookup", n)
	}
	if len(facts) != 3 {
		t.Fatalf("got facts for %d Refs, want 3: %v", len(facts), facts)
	}
	for _, ref := range []string{"cname", "mx"} {
		if f := facts[ref]; f.Registrable != "shared.example" || !f.ExpiryDate.Equal(expiry) || f.Error != "" {
			t.Errorf("facts[%q] = %+v", ref, f)
		}
	}
	if f := facts["ns"]; f.Error != "rdap unreachable" {
		t.Errorf("facts[ns] = %+v, want the lookup error", f)
	}
}

func TestExternalWhoisProvider_CollectCancelled(t *testing.T) {
	p := &externalWhoisProvider{getInfo: func(ctx context.Context, name string) (*domaininfo.DomainInfo, error) {
		t.Errorf("%s looked up after the cancellation", name)
		return nil, nil
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := p.Collect(ctx, happydns.CheckerOptions{"discovery_entries": []sdk.DiscoveryEntry{
		externalTargetEntry(t, "cname", "a.example"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if facts := got.(*ExternalWhoisData).Facts; len(facts) != 0 {
		t.Errorf("got facts %v, want none for the registrables not looked up", facts)
	}
}
