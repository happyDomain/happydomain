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

package abstract_test

import (
	"testing"

	"git.happydns.org/happyDomain/services/abstract"
)

func TestServiceHostingHost(t *testing.T) {
	t.Cleanup(func() { abstract.SetServiceHostingHost("") })

	for _, tc := range []struct {
		set, want string
	}{
		{"happydomain.example.com", "happydomain.example.com."},
		// A trailing dot given by the operator must not be doubled.
		{"happydomain.example.com.", "happydomain.example.com."},
		{"", ""},
	} {
		abstract.SetServiceHostingHost(tc.set)
		if got := abstract.GetServiceHostingHost(); got != tc.want {
			t.Errorf("after Set(%q), Get() = %q; want %q", tc.set, got, tc.want)
		}
	}
}
