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

package provider_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
)

// stubActuator lists no zone, and records what it was asked.
type stubActuator struct {
	listed bool
	known  []string
	// guard is the guard handed to the instantiation, if any.
	guard *netguard.Guard
	// resolvers are the resolvers handed to the instantiation, if any.
	resolvers []string
}

func (a *stubActuator) CanCreateDomain() bool          { return false }
func (a *stubActuator) CanListZones() bool             { return false }
func (a *stubActuator) CreateDomain(fqdn string) error { return nil }
func (a *stubActuator) ListZones() ([]string, error)   { return nil, nil }
func (a *stubActuator) GetZoneCorrections(domain string, wantedRecords []happydns.Record) ([]*happydns.Correction, int, error) {
	return nil, 0, nil
}
func (a *stubActuator) GetZoneRecords(domain string) ([]happydns.Record, error) {
	a.listed = true
	return nil, nil
}

// discoveringActuator recomposes the zone instead of listing it.
type discoveringActuator struct {
	stubActuator
}

func (a *discoveringActuator) GetZoneRecordsWithContext(ctx context.Context, domain string, known []string) ([]happydns.Record, bool, error) {
	a.known = known
	return nil, true, nil
}

// stubBody hands out its actuator. The instantiation works on a copy of
// the body: what it learns is recorded on the actuator, shared by the copy.
type stubBody struct {
	actuator *stubActuator
}

func (b *stubBody) InstantiateProvider() (happydns.ProviderActuator, error) {
	return b.actuator, nil
}

type guardedStubBody struct {
	actuator *discoveringActuator
	// err fails the instantiation.
	err error
}

func (b *guardedStubBody) InstantiateProvider() (happydns.ProviderActuator, error) {
	return b.actuator, nil
}

func (b *guardedStubBody) InstantiateGuardedProvider(guard *netguard.Guard, resolvers []string) (happydns.ProviderActuator, error) {
	if b.err != nil {
		return nil, b.err
	}
	b.actuator.guard = guard
	b.actuator.resolvers = resolvers
	return b.actuator, nil
}

// stubProvider wraps body in a provider owned by someone, as opening its
// secrets requires.
func stubProvider(body happydns.ProviderBody) *happydns.Provider {
	return &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{
			Id:    happydns.Identifier("stub-provider"),
			Owner: happydns.Identifier("stub-owner"),
		},
		Provider: body,
	}
}

func Test_RetrieveZone_HandsKnownSubdomainsToDiscoverers(t *testing.T) {
	providerService, _ := newTestService(t)

	actuator := &discoveringActuator{}
	p := stubProvider(&guardedStubBody{actuator: actuator})

	_, partial, err := providerService.RetrieveZone(ctx, p, "example.com.", "www", "_dmarc")
	if err != nil {
		t.Fatalf("RetrieveZone() = %v", err)
	}
	if !partial {
		t.Error("RetrieveZone lost that the discovered zone is partial")
	}

	if actuator.listed {
		t.Error("RetrieveZone listed the zone of a provider discovering it")
	}
	if !slices.Equal(actuator.known, []string{"www", "_dmarc"}) {
		t.Errorf("known subdomains = %v, want [www _dmarc]", actuator.known)
	}
}

func Test_Instantiate_HandsTheGuardAndResolvers(t *testing.T) {
	db, _ := inmemory.Instantiate()
	guard, err := netguard.New("test", "-test-allowed-target", nil)
	if err != nil {
		t.Fatalf("guard: %v", err)
	}
	resolvers := []string{"192.0.2.53:53", "192.0.2.54:53"}
	providerService := provider.NewService(db, &mockValidator{}, guard, resolvers, plaintextSecrets(t))

	actuator := &discoveringActuator{}
	providerService.ListHostedDomains(ctx, stubProvider(&guardedStubBody{actuator: actuator}))
	if actuator.guard != guard {
		t.Errorf("the provider was instantiated with guard %p; want that of the service, %p", actuator.guard, guard)
	}
	if !slices.Equal(actuator.resolvers, resolvers) {
		t.Errorf("the provider was instantiated with resolvers %v; want those of the service, %v", actuator.resolvers, resolvers)
	}
}

func Test_Instantiate_GuardedProviderFails(t *testing.T) {
	providerService, _ := newTestService(t)

	failure := errors.New("unreachable endpoint")
	_, err := providerService.ListHostedDomains(ctx, stubProvider(&guardedStubBody{actuator: &discoveringActuator{}, err: failure}))
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), "unable to instantiate provider") {
		t.Errorf("ListHostedDomains error = %v; want the failure of the instantiation, wrapped", err)
	}
}

func Test_RetrieveZone_ListsOtherProviders(t *testing.T) {
	providerService, _ := newTestService(t)

	actuator := &stubActuator{}
	p := stubProvider(&stubBody{actuator: actuator})

	_, partial, err := providerService.RetrieveZone(ctx, p, "example.com.", "www")
	if err != nil {
		t.Fatalf("RetrieveZone() = %v", err)
	}
	if partial {
		t.Error("RetrieveZone tells a listed zone is partial")
	}

	if !actuator.listed {
		t.Error("RetrieveZone did not list the zone")
	}
}

// existenceActuator tells whether a domain exists without retrieving it.
type existenceActuator struct {
	stubActuator
	tested string
	err    error
}

func (a *existenceActuator) TestDomainExistence(ctx context.Context, domain string) error {
	a.tested = domain
	return a.err
}

func Test_TestDomainExistence_UsesTheCheapTest(t *testing.T) {
	providerService, _ := newTestService(t)

	actuator := &existenceActuator{}
	p := stubProvider(&existenceBody{actuator: actuator})

	if err := providerService.TestDomainExistence(ctx, p, "example.com."); err != nil {
		t.Fatalf("TestDomainExistence() = %v", err)
	}

	if actuator.listed {
		t.Error("TestDomainExistence retrieved the whole zone")
	}
	if actuator.tested != "example.com." {
		t.Errorf("tested = %q, want example.com.", actuator.tested)
	}
}

func Test_TestDomainExistence_ReportsTheCheapTestError(t *testing.T) {
	providerService, _ := newTestService(t)

	actuator := &existenceActuator{err: errors.New("no such domain")}
	err := providerService.TestDomainExistence(ctx, stubProvider(&existenceBody{actuator: actuator}), "example.com.")
	if !errors.Is(err, actuator.err) {
		t.Errorf("TestDomainExistence() = %v, want %v", err, actuator.err)
	}
	if actuator.listed {
		t.Error("TestDomainExistence retrieved the whole zone")
	}
}

func Test_TestDomainExistence_RetrievesOtherProviders(t *testing.T) {
	providerService, _ := newTestService(t)

	actuator := &stubActuator{}
	if err := providerService.TestDomainExistence(ctx, stubProvider(&stubBody{actuator: actuator}), "example.com."); err != nil {
		t.Fatalf("TestDomainExistence() = %v", err)
	}
	if !actuator.listed {
		t.Error("TestDomainExistence did not retrieve the zone")
	}
}

type existenceBody struct {
	actuator *existenceActuator
}

func (b *existenceBody) InstantiateProvider() (happydns.ProviderActuator, error) {
	return b.actuator, nil
}
