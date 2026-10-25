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

package providers // import "git.happydns.org/happyDomain/providers"

import (
	"encoding/json"
	"slices"
	"testing"

	"git.happydns.org/happyDomain/internal/discovery"
	"git.happydns.org/happyDomain/internal/forms"
	"git.happydns.org/happyDomain/model"
)

func TestPublicDNSDiscoveryIsReadOnly(t *testing.T) {
	var body happydns.ProviderBody = &PublicDNSDiscovery{}

	ro, ok := body.(happydns.ReadOnlyProvider)
	if !ok || !ro.IsReadOnly() {
		t.Fatal("PublicDNSDiscovery is not a read-only provider")
	}

	actuator, err := body.InstantiateProvider()
	if err != nil {
		t.Fatalf("InstantiateProvider: %v", err)
	}
	if _, ok := actuator.(happydns.DiscoveryZoneGetter); !ok {
		t.Error("the actuator does not take the known subdomains")
	}
	if _, _, err := actuator.GetZoneCorrections("example.com.", nil); err == nil {
		t.Error("GetZoneCorrections succeeded on a read-only provider")
	}
}

func TestPublicDNSDiscoveryFields(t *testing.T) {
	// The settings of the techniques are embedded together: two
	// techniques sharing a field name would shadow each other.
	fields := forms.GenStructFields(&PublicDNSDiscovery{})
	seen := map[string]bool{}
	for _, field := range fields {
		if seen[field.Id] {
			t.Errorf("field %q is declared twice", field.Id)
		}
		seen[field.Id] = true
	}

	// The servers queried come from DNS answers, vetted by the guard: the
	// user names no destination.
	if endpoints := forms.Endpoints(&PublicDNSDiscovery{}); len(endpoints) != 0 {
		t.Errorf("Endpoints = %v; want none", endpoints)
	}
}

func TestPublicDNSDiscoveryJSON(t *testing.T) {
	var body PublicDNSDiscovery
	if err := json.Unmarshal([]byte(`{"unknown":true}`), &body); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	b, err := json.Marshal(&body)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var flat map[string]any
	if err := json.Unmarshal(b, &flat); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := flat["unknown"]; ok {
		t.Errorf("marshalled %s; want the unknown field dropped", b)
	}

	// Through the interface, as providers are stored.
	var stored happydns.ProviderBody = &body
	if b2, err := json.Marshal(stored); err != nil || string(b2) != string(b) {
		t.Errorf("marshalled through the interface %s, %v; want %s", b2, err, b)
	}
}

func TestPublicDNSDiscoverySources(t *testing.T) {
	ids := func(sources []discovery.NameSource) (ret []string) {
		for _, source := range sources {
			ret = append(ret, source.ID())
		}
		return
	}

	// With no settings, every technique enabled by default is used, in
	// priority order.
	var want []string
	for _, f := range discovery.SourceFactories() {
		if source := f.New(new(discovery.Settings)); source != nil {
			want = append(want, source.ID())
		}
	}
	if got := ids((&PublicDNSDiscovery{}).Sources()); !slices.Equal(got, want) {
		t.Errorf("sources with no settings = %v; want %v", got, want)
	}
}
