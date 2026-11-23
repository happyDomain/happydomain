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

package providerregistry

import (
	"slices"
	"testing"

	"git.happydns.org/happyDomain/model"
)

type firstProvider struct{ happydns.ProviderBody }
type secondProvider struct{ happydns.ProviderBody }
type muteProvider struct{ happydns.ProviderBody }
type brokenProvider struct{ happydns.ProviderBody }

// withRegistry runs the test on a registry of its own.
func withRegistry(t *testing.T) {
	t.Helper()
	providers, nameservers := providerRegistry, nameserverMatchers
	providerRegistry = map[string]happydns.ProviderCreator{}
	nameserverMatchers = map[string]nameserverMatcher{}
	t.Cleanup(func() { providerRegistry, nameserverMatchers = providers, nameservers })

	RegisterProvider(func() happydns.ProviderBody { return &firstProvider{} }, happydns.ProviderInfos{
		Nameservers: []string{`d?ns\d*\.first\.example`, `shared\d\.example`, `a\.alt\.example|b\.alt\.example`},
	})
	RegisterProvider(func() happydns.ProviderBody { return &secondProvider{} }, happydns.ProviderInfos{
		Nameservers: []string{`ns\d\.second\.example`, `shared\d\.example`},
	})
	RegisterProvider(func() happydns.ProviderBody { return &muteProvider{} }, happydns.ProviderInfos{})
}

func TestMatchNameservers(t *testing.T) {
	withRegistry(t)

	for name, tt := range map[string]struct {
		nameservers []string
		want        []string
	}{
		"no name server":      {nil, []string{}},
		"unknown":             {[]string{"ns1.example.com."}, []string{}},
		"final dot":           {[]string{"dns103.first.example."}, []string{"firstProvider"}},
		"no final dot":        {[]string{"ns.first.example"}, []string{"firstProvider"}},
		"upper case":          {[]string{"NS1.Second.Example."}, []string{"secondProvider"}},
		"anchored start":      {[]string{"evil-ns1.second.example."}, []string{}},
		"anchored subdomain":  {[]string{"x.ns1.second.example."}, []string{}},
		"anchored end":        {[]string{"ns1.second.example.evil.example."}, []string{}},
		"anchored choice":     {[]string{"x.b.alt.example.", "a.alt.example.x."}, []string{}},
		"choice":              {[]string{"b.alt.example."}, []string{"firstProvider"}},
		"one of them matches": {[]string{"ns.example.com.", "ns1.second.example."}, []string{"secondProvider"}},
		"shared":              {[]string{"shared1.example."}, []string{"firstProvider", "secondProvider"}},
		"mixed, sorted":       {[]string{"ns1.second.example.", "ns1.first.example."}, []string{"firstProvider", "secondProvider"}},
	} {
		t.Run(name, func(t *testing.T) {
			got := MatchNameservers(tt.nameservers)
			if got == nil || !slices.Equal(got, tt.want) {
				t.Errorf("MatchNameservers(%q) = %#v, want %#v", tt.nameservers, got, tt.want)
			}
		})
	}
}

func TestRegisterProviderRefusesBrokenPattern(t *testing.T) {
	withRegistry(t)

	defer func() {
		if recover() == nil {
			t.Error("a pattern that does not compile has been accepted")
		}
	}()
	RegisterProvider(func() happydns.ProviderBody { return &brokenProvider{} }, happydns.ProviderInfos{
		Nameservers: []string{`ns(\d`},
	})
}
