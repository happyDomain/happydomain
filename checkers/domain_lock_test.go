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

func TestDomainLockRule_Evaluate(t *testing.T) {
	rule := &domainLockRule{}

	cases := []struct {
		name    string
		status  []string
		opts    happydns.CheckerOptions
		want    happydns.Status
		code    string
	}{
		{
			name:   "default required present",
			status: []string{"clientTransferProhibited", "ok"},
			opts:   nil,
			want:   happydns.StatusOK,
			code:   "lock_ok",
		},
		{
			name:   "default required missing",
			status: []string{"ok"},
			opts:   nil,
			want:   happydns.StatusCrit,
			code:   "lock_missing",
		},
		{
			name:   "multiple required all present",
			status: []string{"clientTransferProhibited", "clientUpdateProhibited", "clientDeleteProhibited"},
			opts: happydns.CheckerOptions{
				"requiredStatuses": "clientTransferProhibited,clientUpdateProhibited,clientDeleteProhibited",
			},
			want: happydns.StatusOK,
			code: "lock_ok",
		},
		{
			name:   "multiple required some missing",
			status: []string{"clientTransferProhibited"},
			opts: happydns.CheckerOptions{
				"requiredStatuses": "clientTransferProhibited,clientUpdateProhibited",
			},
			want: happydns.StatusCrit,
			code: "lock_missing",
		},
		{
			name:   "case insensitive match",
			status: []string{"clienttransferprohibited"},
			opts:   nil,
			want:   happydns.StatusOK,
			code:   "lock_ok",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obs := newWhoisObs(&WHOISData{Status: tc.status})
			states := rule.Evaluate(context.Background(), obs, tc.opts)
			if len(states) != 1 {
				t.Fatalf("expected 1 state, got %d", len(states))
			}
			st := states[0]
			if st.Status != tc.want {
				t.Errorf("status = %v, want %v (msg=%q)", st.Status, tc.want, st.Message)
			}
			if st.Code != tc.code {
				t.Errorf("code = %q, want %q", st.Code, tc.code)
			}
		})
	}
}

// Sanity: WHOIS data with nil/empty status must report missing locks, not panic.
func TestDomainLockRule_NilStatus(t *testing.T) {
	rule := &domainLockRule{}

	cases := []struct {
		name   string
		status []string
	}{
		{"nil status", nil},
		{"empty status", []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obs := newWhoisObs(&WHOISData{Status: tc.status})
			states := rule.Evaluate(context.Background(), obs, nil)
			if len(states) != 1 {
				t.Fatalf("expected 1 state, got %d", len(states))
			}
			st := states[0]
			if st.Status != happydns.StatusCrit {
				t.Errorf("status = %v, want Crit", st.Status)
			}
			if st.Code != "lock_missing" {
				t.Errorf("code = %q, want lock_missing", st.Code)
			}
		})
	}
}

func TestDomainLockRule_EvaluateObservationError(t *testing.T) {
	rule := &domainLockRule{}
	obs := &stubObservationGetter{key: ObservationKeyWhois, err: errString("nope")}
	states := rule.Evaluate(context.Background(), obs, nil)
	if len(states) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states))
	}
	st := states[0]
	if st.Status != happydns.StatusError || st.Code != "lock_error" {
		t.Errorf("got %v / %q", st.Status, st.Code)
	}
}

func TestDomainLockRule_ValidateOptions(t *testing.T) {
	rule := &domainLockRule{}
	cases := []struct {
		name    string
		opts    happydns.CheckerOptions
		wantErr bool
	}{
		{"default", nil, false},
		{"valid", happydns.CheckerOptions{"requiredStatuses": "clientTransferProhibited"}, false},
		{"empty after split", happydns.CheckerOptions{"requiredStatuses": " , , "}, true},
		{"wrong type", happydns.CheckerOptions{"requiredStatuses": 123}, true},
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
