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

package domaininfo

import (
	"testing"
)

func TestNormalizeStatus(t *testing.T) {
	for in, want := range map[string]string{
		"client transfer prohibited": "clientTransferProhibited",
		"pending delete":             "pendingDelete",
		"redemption period":          "redemptionPeriod",
		"pending restore":            "pendingRestore",
		"auto renew period":          "autoRenewPeriod",
		"Server Hold":                "serverHold",
		"  pending   delete ":        "pendingDelete",
		"pendingDelete":              "pendingDelete",
		"client-transfer-prohibited": "client-transfer-prohibited",
		"active":                     "active",
		"ok":                         "ok",
		"":                           "",
	} {
		if got := NormalizeStatus(in); got != want {
			t.Errorf("NormalizeStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStatusKey(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"clientTransferProhibited", "clienttransferprohibited"},
		{"client transfer prohibited", "clienttransferprohibited"},
		{"client-transfer-prohibited", "clienttransferprohibited"},
		{"client_transfer_prohibited", "clienttransferprohibited"},
		{"  Client Transfer Prohibited  ", "clienttransferprohibited"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := StatusKey(tc.in); got != tc.want {
			t.Errorf("StatusKey(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeStatuses(t *testing.T) {
	if got := normalizeStatuses(nil); got != nil {
		t.Errorf("normalizeStatuses(nil) = %#v, want nil", got)
	}

	got := normalizeStatuses([]string{"client transfer prohibited", "ok"})
	if len(got) != 2 || got[0] != "clientTransferProhibited" || got[1] != "ok" {
		t.Errorf("normalizeStatuses() = %#v", got)
	}
}

func TestHasStatus(t *testing.T) {
	statuses := []string{"client transfer prohibited", "redemptionPeriod"}
	cases := []struct {
		statuses []string
		status   string
		want     bool
	}{
		{statuses, "clientTransferProhibited", true},
		{statuses, "redemption period", true},
		{statuses, "pendingDelete", false},
		{nil, "ok", false},
	}
	for _, tc := range cases {
		if got := HasStatus(tc.statuses, tc.status); got != tc.want {
			t.Errorf("HasStatus(%q, %q) = %v, want %v", tc.statuses, tc.status, got, tc.want)
		}
	}
}
