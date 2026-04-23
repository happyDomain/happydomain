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
	"strings"
	"testing"

	"git.happydns.org/happyDomain/model"
)

func contactsFixture() map[string]*happydns.ContactInfo {
	return map[string]*happydns.ContactInfo{
		"registrant": {
			Name:         "Alice Example",
			Organization: "Example Inc",
			Email:        "alice@example.com",
		},
		"admin": {
			Name:         "REDACTED FOR PRIVACY",
			Organization: "REDACTED FOR PRIVACY",
			Email:        "redacted@example.com",
		},
		"tech": {
			Name:         "Bob Tech",
			Organization: "Example Inc",
			Email:        "bob@example.com",
		},
	}
}

func TestDomainContactRule_Evaluate(t *testing.T) {
	rule := &domainContactRule{}
	obs := newWhoisObs(&WHOISData{Contacts: contactsFixture()})

	cases := []struct {
		name     string
		opts     happydns.CheckerOptions
		wantWorst happydns.Status
		// wantCodes: if non-nil, expect one state per entry with the listed code
		// (order matches roles). If nil, expect a single state and use wantCode.
		wantCodes []string
		wantCode  string
	}{
		{
			name:      "no expectations",
			opts:      nil,
			wantWorst: happydns.StatusUnknown,
			wantCode:  "contact_skipped",
		},
		{
			name: "registrant matches",
			opts: happydns.CheckerOptions{
				"expectedName":  "Alice Example",
				"expectedEmail": "alice@example.com",
			},
			wantWorst: happydns.StatusOK,
			wantCodes: []string{"contact_ok"},
		},
		{
			name: "registrant name mismatch",
			opts: happydns.CheckerOptions{
				"expectedName": "Carol Other",
			},
			wantWorst: happydns.StatusWarn,
			wantCodes: []string{"contact_mismatch"},
		},
		{
			name: "admin role is redacted",
			opts: happydns.CheckerOptions{
				"checkRoles":   "admin",
				"expectedName": "Alice Example",
			},
			wantWorst: happydns.StatusInfo,
			wantCodes: []string{"contact_redacted"},
		},
		{
			name: "missing role",
			opts: happydns.CheckerOptions{
				"checkRoles":   "billing",
				"expectedName": "Alice Example",
			},
			wantWorst: happydns.StatusWarn,
			wantCodes: []string{"contact_missing"},
		},
		{
			name: "multi-role mixed (worst wins)",
			opts: happydns.CheckerOptions{
				"checkRoles":   "registrant,admin",
				"expectedName": "Alice Example",
			},
			// registrant matches (OK), admin is redacted (Info). Info is worst.
			wantWorst: happydns.StatusInfo,
			wantCodes: []string{"contact_ok", "contact_redacted"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			states := rule.Evaluate(context.Background(), obs, tc.opts)
			if tc.wantCodes == nil {
				if len(states) != 1 {
					t.Fatalf("expected 1 state, got %d", len(states))
				}
				st := states[0]
				if st.Status != tc.wantWorst {
					t.Errorf("status = %v, want %v (msg=%q)", st.Status, tc.wantWorst, st.Message)
				}
				if st.Code != tc.wantCode {
					t.Errorf("code = %q, want %q", st.Code, tc.wantCode)
				}
				return
			}
			if len(states) != len(tc.wantCodes) {
				t.Fatalf("state count = %d, want %d", len(states), len(tc.wantCodes))
			}
			worst := happydns.StatusOK
			for i, st := range states {
				if st.Code != tc.wantCodes[i] {
					t.Errorf("state[%d].code = %q, want %q", i, st.Code, tc.wantCodes[i])
				}
				if st.Subject == "" {
					t.Errorf("state[%d].Subject is empty", i)
				}
				worst = worseStatus(worst, st.Status)
			}
			if worst != tc.wantWorst {
				t.Errorf("worst status = %v, want %v", worst, tc.wantWorst)
			}
		})
	}
}

func TestDomainContactRule_EvaluateObservationError(t *testing.T) {
	rule := &domainContactRule{}
	obs := &stubObservationGetter{key: ObservationKeyWhois, err: errString("nope")}
	states := rule.Evaluate(context.Background(), obs, happydns.CheckerOptions{"expectedName": "x"})
	if len(states) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states))
	}
	st := states[0]
	if st.Status != happydns.StatusError || st.Code != "contact_error" {
		t.Errorf("got %v / %q", st.Status, st.Code)
	}
}

func TestDomainContactRule_ValidateOptions(t *testing.T) {
	rule := &domainContactRule{}

	cases := []struct {
		name    string
		opts    happydns.CheckerOptions
		wantErr bool
	}{
		{"empty", nil, false},
		{"all valid", happydns.CheckerOptions{
			"expectedName":  "x",
			"checkRoles":    "registrant,tech",
		}, false},
		{"unknown role", happydns.CheckerOptions{"checkRoles": "billing"}, true},
		{"empty roles after split", happydns.CheckerOptions{"checkRoles": " , , "}, true},
		{"wrong type", happydns.CheckerOptions{"expectedEmail": 42}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := rule.ValidateOptions(tc.opts)
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestIsRedacted(t *testing.T) {
	cases := []struct {
		c    *happydns.ContactInfo
		want bool
	}{
		{&happydns.ContactInfo{Name: "REDACTED FOR PRIVACY"}, true},
		{&happydns.ContactInfo{Organization: "Contact Privacy Inc"}, true},
		{&happydns.ContactInfo{Email: "withheld@example.com"}, true},
		{&happydns.ContactInfo{Name: "Alice", Email: "alice@example.com"}, false},
	}
	for _, tc := range cases {
		if got := isRedacted(tc.c); got != tc.want {
			t.Errorf("isRedacted(%+v) = %v, want %v", tc.c, got, tc.want)
		}
	}
}

func TestWorseStatus(t *testing.T) {
	cases := []struct {
		a, b, want happydns.Status
	}{
		{happydns.StatusOK, happydns.StatusInfo, happydns.StatusInfo},
		{happydns.StatusInfo, happydns.StatusWarn, happydns.StatusWarn},
		{happydns.StatusCrit, happydns.StatusWarn, happydns.StatusCrit},
		{happydns.StatusOK, happydns.StatusUnknown, happydns.StatusOK},
		{happydns.StatusError, happydns.StatusCrit, happydns.StatusError},
	}
	for _, tc := range cases {
		if got := worseStatus(tc.a, tc.b); got != tc.want {
			t.Errorf("worseStatus(%v,%v) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

// Sanity: WHOIS data with no contacts must not panic when a role is requested.
func TestDomainContactRule_NilContacts(t *testing.T) {
	rule := &domainContactRule{}
	obs := newWhoisObs(&WHOISData{})
	states := rule.Evaluate(context.Background(), obs, happydns.CheckerOptions{
		"expectedName": "Alice",
	})
	if len(states) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states))
	}
	st := states[0]
	if st.Status != happydns.StatusWarn {
		t.Errorf("status = %v, want Warn", st.Status)
	}
	if !strings.Contains(st.Message, "not found") {
		t.Errorf("message = %q", st.Message)
	}
}
