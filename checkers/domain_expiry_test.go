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
	"testing"
	"time"

	sdk "git.happydns.org/checker-sdk-go/checker"
	"git.happydns.org/happyDomain/model"
)

func TestDomainExpiryRule_Evaluate(t *testing.T) {
	rule := &domainExpiryRule{}
	now := time.Now()

	cases := []struct {
		name      string
		expiresIn time.Duration
		opts      happydns.CheckerOptions
		want      happydns.Status
		code      string
	}{
		{"already expired", -5 * 24 * time.Hour, nil, happydns.StatusCrit, "expiry_critical"},
		{"critical default", 3 * 24 * time.Hour, nil, happydns.StatusCrit, "expiry_critical"},
		{"warning default", 15 * 24 * time.Hour, nil, happydns.StatusWarn, "expiry_warning"},
		{"ok default", 90 * 24 * time.Hour, nil, happydns.StatusOK, "expiry_ok"},
		{"ok with custom thresholds", 10 * 24 * time.Hour, happydns.CheckerOptions{
			"warning_days":  float64(5),
			"critical_days": float64(2),
		}, happydns.StatusOK, "expiry_ok"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obs := newWhoisObs(&WHOISData{
				ExpiryDate: now.Add(tc.expiresIn),
				Registrar:  "Test Registrar",
			})
			st := rule.Evaluate(context.Background(), obs, tc.opts)
			if st.Status != tc.want {
				t.Errorf("status = %v, want %v (msg=%q)", st.Status, tc.want, st.Message)
			}
			if st.Code != tc.code {
				t.Errorf("code = %q, want %q", st.Code, tc.code)
			}
		})
	}
}

func TestDomainExpiryRule_EvaluateObservationError(t *testing.T) {
	rule := &domainExpiryRule{}
	obs := &stubObservationGetter{key: ObservationKeyWhois, err: errString("boom")}
	st := rule.Evaluate(context.Background(), obs, nil)
	if st.Status != happydns.StatusError {
		t.Fatalf("expected StatusError, got %v", st.Status)
	}
	if st.Code != "whois_error" {
		t.Errorf("code = %q, want whois_error", st.Code)
	}
}

func TestDomainExpiryRule_ValidateOptions(t *testing.T) {
	rule := &domainExpiryRule{}

	cases := []struct {
		name    string
		opts    happydns.CheckerOptions
		wantErr bool
	}{
		{"defaults", nil, false},
		{"valid custom", happydns.CheckerOptions{"warning_days": 30.0, "critical_days": 7.0}, false},
		{"crit >= warn", happydns.CheckerOptions{"warning_days": 5.0, "critical_days": 5.0}, true},
		{"warn wrong type", happydns.CheckerOptions{"warning_days": "thirty"}, true},
		{"warn negative", happydns.CheckerOptions{"warning_days": -1.0}, true},
		{"crit negative", happydns.CheckerOptions{"critical_days": -3.0}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := rule.ValidateOptions(tc.opts)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateOptions err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestWhoisProvider_ExtractMetrics(t *testing.T) {
	p := &whoisProvider{}
	collected := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	data := WHOISData{
		ExpiryDate: collected.Add(10 * 24 * time.Hour),
		Registrar:  "Acme",
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	metrics, err := p.ExtractMetrics(sdk.StaticReportContext(raw), collected)
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	m := metrics[0]
	if m.Name != "domain_expiry_days_remaining" {
		t.Errorf("name = %q", m.Name)
	}
	if m.Value < 9.99 || m.Value > 10.01 {
		t.Errorf("value = %v, want ~10", m.Value)
	}
	if m.Labels["registrar"] != "Acme" {
		t.Errorf("registrar label = %q", m.Labels["registrar"])
	}
}
