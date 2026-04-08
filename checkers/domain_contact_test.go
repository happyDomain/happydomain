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
		name string
		opts happydns.CheckerOptions
		want happydns.Status
		code string
	}{
		{
			name: "no expectations",
			opts: nil,
			want: happydns.StatusUnknown,
			code: "contact_skipped",
		},
		{
			name: "registrant matches",
			opts: happydns.CheckerOptions{
				"expectedName":  "Alice Example",
				"expectedEmail": "alice@example.com",
			},
			want: happydns.StatusOK,
			code: "contact_result",
		},
		{
			name: "registrant name mismatch",
			opts: happydns.CheckerOptions{
				"expectedName": "Carol Other",
			},
			want: happydns.StatusWarn,
			code: "contact_result",
		},
		{
			name: "admin role is redacted",
			opts: happydns.CheckerOptions{
				"checkRoles":   "admin",
				"expectedName": "Alice Example",
			},
			want: happydns.StatusInfo,
			code: "contact_result",
		},
		{
			name: "missing role",
			opts: happydns.CheckerOptions{
				"checkRoles":   "billing",
				"expectedName": "Alice Example",
			},
			want: happydns.StatusWarn,
			code: "contact_result",
		},
		{
			name: "multi-role mixed (worst wins)",
			opts: happydns.CheckerOptions{
				"checkRoles":   "registrant,admin",
				"expectedName": "Alice Example",
			},
			// admin is redacted (Info) — Info is worse than OK from registrant.
			want: happydns.StatusInfo,
			code: "contact_result",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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

func TestDomainContactRule_EvaluateObservationError(t *testing.T) {
	rule := &domainContactRule{}
	obs := &stubObservationGetter{key: ObservationKeyWhois, err: errString("nope")}
	st := rule.Evaluate(context.Background(), obs, happydns.CheckerOptions{"expectedName": "x"})
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
	st := rule.Evaluate(context.Background(), obs, happydns.CheckerOptions{
		"expectedName": "Alice",
	})
	if st.Status != happydns.StatusWarn {
		t.Errorf("status = %v, want Warn", st.Status)
	}
	if !strings.Contains(st.Message, "not found") {
		t.Errorf("message = %q", st.Message)
	}
}
