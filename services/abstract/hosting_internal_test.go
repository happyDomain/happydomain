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

package abstract

import "testing"

func TestHostingTargetMatches(t *testing.T) {
	t.Cleanup(func() { SetServiceHostingHost("") })

	SetServiceHostingHost("happydomain.example.com")
	for target, want := range map[string]bool{
		"happydomain.example.com.":     true,
		"happydomain.example.com":      true,
		"HappyDomain.Example.COM.":     true,
		"third-party.example.net.":     false,
		"sub.happydomain.example.com.": false,
		"":                             false,
	} {
		if got := hostingTargetMatches(target); got != want {
			t.Errorf("hostingTargetMatches(%q) = %v; want %v", target, got, want)
		}
	}

	// With no host configured, nothing can be confirmed as pointing at us.
	SetServiceHostingHost("")
	for _, target := range []string{"happydomain.example.com.", ""} {
		if hostingTargetMatches(target) {
			t.Errorf("hostingTargetMatches(%q) = true with no host configured; want false", target)
		}
	}
}
