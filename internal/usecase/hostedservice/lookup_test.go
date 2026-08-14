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

package hostedservice

import "testing"

func TestStripPrefix(t *testing.T) {
	for _, tc := range []struct {
		in       string
		prefixes []string
		want     string
		wantOK   bool
	}{
		{"mta-sts.example.com.", []string{"mta-sts."}, "example.com.", true},
		{"mta-sts.example.com", []string{"mta-sts."}, "example.com.", true},
		{"MTA-STS.example.com.", []string{"mta-sts."}, "example.com.", true},
		{"example.com.", []string{"mta-sts."}, "example.com.", false},
		// A domain that merely starts with the same letters is not a policy host.
		{"mta-stsx.example.com.", []string{"mta-sts."}, "mta-stsx.example.com.", false},
		{"autoconfig.example.com", []string{"autoconfig.", "autodiscover."}, "example.com.", true},
		{"autodiscover.example.com.", []string{"autoconfig.", "autodiscover."}, "example.com.", true},
		{"www.example.com", []string{"autoconfig.", "autodiscover."}, "www.example.com.", false},
	} {
		got, ok := StripPrefix(tc.in, tc.prefixes...)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("StripPrefix(%q, %v) = (%q, %v); want (%q, %v)", tc.in, tc.prefixes, got, ok, tc.want, tc.wantOK)
		}
	}
}
