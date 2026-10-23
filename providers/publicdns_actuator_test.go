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
	"context"
	"slices"
	"testing"

	"git.happydns.org/happyDomain/internal/discovery"
	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

type testTechnique struct{}

func (testTechnique) ID() string { return "test-publicdns" }

func (testTechnique) Discover(ctx context.Context, t discovery.Target) ([]discovery.Name, error) {
	return nil, nil
}

func init() {
	discovery.RegisterSource("test-publicdns", 999, func(s *discovery.Settings) discovery.NameSource {
		return testTechnique{}
	})
}

func TestPublicDNSDiscoveryTechniques(t *testing.T) {
	if !slices.ContainsFunc((&PublicDNSDiscovery{}).Sources(), func(s discovery.NameSource) bool { return s.ID() == "test-publicdns" }) {
		t.Error("a registered technique is not used")
	}
}

func TestPublicDNSDiscoveryActuator(t *testing.T) {
	body := &PublicDNSDiscovery{Resolver: "9.9.9.9"}

	guard, err := netguard.New("test", "-test-allowed-target", nil)
	if err != nil {
		t.Fatalf("guard: %v", err)
	}
	instance, err := body.InstantiateGuardedProvider(guard)
	if err != nil {
		t.Fatalf("InstantiateGuardedProvider: %v", err)
	}
	actuator := instance.(*publicDNSActuator)
	if actuator.opts.Guard != guard || actuator.opts.Resolver != "9.9.9.9" {
		t.Errorf("options = %+v; want the guard and the resolver handed", actuator.opts)
	}

	// Without a guard, the exploration refuses every non-public server.
	if instance, err = body.InstantiateProvider(); err != nil || instance.(*publicDNSActuator).opts.Guard != nil {
		t.Errorf("InstantiateProvider = %v, %v; want an actuator without guard", instance, err)
	}

	if actuator.CanCreateDomain() || actuator.CanListZones() {
		t.Error("the discovery claims to create domains or list zones")
	}
	if err := actuator.CreateDomain("example.com."); err == nil {
		t.Error("CreateDomain succeeded")
	}
	if _, err := actuator.ListZones(); err == nil {
		t.Error("ListZones succeeded")
	}
	if _, _, err := actuator.GetZoneCorrections("example.com.", nil); err == nil {
		t.Error("GetZoneCorrections succeeded")
	}
}

func TestPublicDNSDiscoveryTestDomainExistence(t *testing.T) {
	instance, err := (&PublicDNSDiscovery{}).InstantiateProvider()
	if err != nil {
		t.Fatalf("InstantiateProvider: %v", err)
	}

	// The existence is tested with the cheap lookup of the name servers,
	// not with the full exploration GetZoneRecords makes.
	if _, ok := instance.(happydns.DomainExistenceChecker); !ok {
		t.Error("the discovery tests the existence of a domain by exploring it")
	}
}
