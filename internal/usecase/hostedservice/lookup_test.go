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

import (
	"errors"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/usecase/hostedservice/hostedservicetest"
	happydns "git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/abstract"
)

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

// zoneWith builds a zone holding the given services at the apex.
func zoneWith(id byte, services ...happydns.ServiceBody) (*happydns.Zone, happydns.Identifier) {
	zoneID := happydns.Identifier{id}
	zone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{Id: zoneID},
		Services: map[happydns.Subdomain][]*happydns.Service{},
	}
	for _, s := range services {
		zone.Services[""] = append(zone.Services[""], &happydns.Service{Service: s})
	}
	return zone, zoneID
}

func TestFindService(t *testing.T) {
	want := &abstract.EmailAutoConfig{}
	other := &abstract.EmailAutoConfig{}

	zone1, id1 := zoneWith(1, &abstract.MTASTS{}, want)
	zone2, id2 := zoneWith(2, other)

	domains := &hostedservicetest.FakeDomains{ByName: map[string][]*happydns.Domain{
		"example.com.": {
			// Skipped: no zone published yet.
			{DomainName: "example.com."},
			// Skipped: its zone cannot be loaded.
			{DomainName: "example.com.", ZoneHistory: []happydns.Identifier{{0x99}}},
			{DomainName: "example.com.", ZoneHistory: []happydns.Identifier{id1}},
			{DomainName: "example.com.", ZoneHistory: []happydns.Identifier{id2}},
		},
		"onlympta.example.": {
			{DomainName: "onlympta.example.", ZoneHistory: []happydns.Identifier{id1}},
		},
	}}
	zones := &hostedservicetest.FakeZones{ByID: map[string]*happydns.Zone{
		id1.String(): zone1,
		id2.String(): zone2,
	}}
	store := NewStore(domains, zones)

	t.Run("first usable domain wins", func(t *testing.T) {
		got, err := FindService[*abstract.EmailAutoConfig](store, "example.com.")
		if err != nil {
			t.Fatalf("FindService: %v", err)
		}
		if got != want {
			t.Errorf("FindService returned %p; want the service of the first usable domain %p", got, want)
		}
	})

	t.Run("picks the requested type", func(t *testing.T) {
		if _, err := FindService[*abstract.MTASTS](store, "example.com."); err != nil {
			t.Errorf("FindService[MTASTS]: %v", err)
		}
	})

	t.Run("service of another type", func(t *testing.T) {
		zone, id := zoneWith(3, &abstract.MTASTS{})
		store := NewStore(
			&hostedservicetest.FakeDomains{ByName: map[string][]*happydns.Domain{"mta.example.": {{ZoneHistory: []happydns.Identifier{id}}}}},
			&hostedservicetest.FakeZones{ByID: map[string]*happydns.Zone{id.String(): zone}},
		)

		if _, err := FindService[*abstract.EmailAutoConfig](store, "mta.example."); !errors.Is(err, happydns.ErrNotFound) {
			t.Errorf("FindService error = %v; want ErrNotFound", err)
		}
	})

	t.Run("unknown domain", func(t *testing.T) {
		if _, err := FindService[*abstract.EmailAutoConfig](store, "unknown.example."); !errors.Is(err, happydns.ErrNotFound) {
			t.Errorf("FindService error = %v; want ErrNotFound", err)
		}
	})

	t.Run("storage error", func(t *testing.T) {
		boom := errors.New("storage is down")
		failing := NewStore(&hostedservicetest.FakeDomains{Err: boom}, zones)

		if _, err := FindService[*abstract.EmailAutoConfig](failing, "example.com."); !errors.Is(err, boom) {
			t.Errorf("FindService error = %v; want the storage error", err)
		}
	})
}

func TestIsManaged(t *testing.T) {
	hostedPolicy := &abstract.MTASTS{PolicyCNAME: &dns.CNAME{
		Hdr:    dns.RR_Header{Name: "mta-sts", Rrtype: dns.TypeCNAME, Class: dns.ClassINET},
		Target: "happydomain.example.com.",
	}}

	store := func(name string, service happydns.ServiceBody) Store {
		domains, zones := hostedservicetest.NewStore(t, name, service)
		return NewStore(domains, zones)
	}

	t.Run("service without a hosted notion", func(t *testing.T) {
		got, err := IsManaged[*abstract.EmailAutoConfig](store("example.com.", &abstract.EmailAutoConfig{}), "autoconfig.example.com.", "autoconfig.")
		if err != nil || !got {
			t.Errorf("IsManaged = (%v, %v); want (true, nil)", got, err)
		}
	})

	t.Run("hosted service", func(t *testing.T) {
		got, err := IsManaged[*abstract.MTASTS](store("example.com.", hostedPolicy), "mta-sts.example.com.", "mta-sts.")
		if err != nil || !got {
			t.Errorf("IsManaged = (%v, %v); want (true, nil)", got, err)
		}
	})

	t.Run("service configured for the DNS half only", func(t *testing.T) {
		got, err := IsManaged[*abstract.MTASTS](store("example.com.", &abstract.MTASTS{}), "mta-sts.example.com.", "mta-sts.")
		if err != nil || got {
			t.Errorf("IsManaged = (%v, %v); want (false, nil)", got, err)
		}
	})

	t.Run("name outside the prefixes", func(t *testing.T) {
		got, err := IsManaged[*abstract.MTASTS](store("example.com.", hostedPolicy), "example.com.", "mta-sts.")
		if err != nil || got {
			t.Errorf("IsManaged = (%v, %v); want (false, nil)", got, err)
		}
	})

	t.Run("unknown domain", func(t *testing.T) {
		got, err := IsManaged[*abstract.MTASTS](store("example.com.", hostedPolicy), "mta-sts.unknown.example.", "mta-sts.")
		if err != nil || got {
			t.Errorf("IsManaged = (%v, %v); want (false, nil)", got, err)
		}
	})

	t.Run("storage error", func(t *testing.T) {
		boom := errors.New("storage is down")
		_, zones := hostedservicetest.NewStore(t, "example.com.", hostedPolicy)
		failing := NewStore(&hostedservicetest.FakeDomains{Err: boom}, zones)

		got, err := IsManaged[*abstract.MTASTS](failing, "mta-sts.example.com.", "mta-sts.")
		if !errors.Is(err, boom) || got {
			t.Errorf("IsManaged = (%v, %v); want (false, the storage error)", got, err)
		}
	})
}
