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

package app

import (
	"net/url"
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

func TestResolveHostingHost(t *testing.T) {
	external := url.URL{Scheme: "https", Host: "external.example.com:8443"}

	for name, tc := range map[string]struct {
		cfg  happydns.Options
		want string
	}{
		"service hosting host wins": {
			cfg:  happydns.Options{ServiceHostingHost: "hosting.example.com", MailAutoconfigHost: "legacy.example.com", ExternalURL: external},
			want: "hosting.example.com",
		},
		"deprecated autoconfig host": {
			cfg:  happydns.Options{MailAutoconfigHost: "legacy.example.com", ExternalURL: external},
			want: "legacy.example.com",
		},
		"external URL host without its port": {
			cfg:  happydns.Options{ExternalURL: external},
			want: "external.example.com",
		},
		"nothing configured": {
			want: "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := resolveHostingHost(&tc.cfg); got != tc.want {
				t.Errorf("resolveHostingHost() = %q; want %q", got, tc.want)
			}
		})
	}
}
