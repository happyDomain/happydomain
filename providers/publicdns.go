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

package providers

import (
	"context"
	"errors"

	"git.happydns.org/happyDomain/internal/discovery"
	"git.happydns.org/happyDomain/internal/netguard"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/model"
)

// errPublicDNSReadOnly is returned by every operation changing the zone.
var errPublicDNSReadOnly = errors.New("the public DNS discovery only reads the zone: change it where it is hosted")

// explore is discovery.Explore, swapped in tests: the exploration asks the
// authoritative servers on port 53, which tests cannot serve.
var explore = discovery.Explore

// PublicDNSDiscovery recomposes the zone of a domain hosted where
// happyDomain has no access, from what its authoritative servers publicly
// answer. It is meant to monitor the domain: it never changes the zone.
//
// Its settings are those of every enumeration technique, embedded through
// discovery.Settings. Their zero value is the recommended setting: a
// provider created with no settings at all explores fully.
type PublicDNSDiscovery struct {
	discovery.Settings
}

func (s *PublicDNSDiscovery) IsReadOnly() bool {
	return true
}

func (s *PublicDNSDiscovery) InstantiateProvider() (happydns.ProviderActuator, error) {
	return s.InstantiateGuardedProvider(nil, nil)
}

// InstantiateGuardedProvider hands guard to the exploration, as the
// authoritative servers it queries come from DNS answers, controlled by
// whoever runs the domain, and resolvers, the recursive resolvers asked to
// find them.
func (s *PublicDNSDiscovery) InstantiateGuardedProvider(guard *netguard.Guard, resolvers []string) (happydns.ProviderActuator, error) {
	return &publicDNSActuator{
		opts: discovery.Options{
			Guard:     guard,
			Resolvers: resolvers,
		},
		sources: s.Sources(),
	}, nil
}

// Sources returns the enumeration techniques the settings enable, in
// priority order.
func (s *PublicDNSDiscovery) Sources() (sources []discovery.NameSource) {
	for _, f := range discovery.SourceFactories() {
		if source := f.New(&s.Settings); source != nil {
			sources = append(sources, source)
		}
	}
	return
}

// The provider usecase recognises the discovery by these interfaces.
var (
	_ happydns.DiscoveryZoneGetter    = (*publicDNSActuator)(nil)
	_ happydns.DomainExistenceChecker = (*publicDNSActuator)(nil)
)

type publicDNSActuator struct {
	opts    discovery.Options
	sources []discovery.NameSource
}

func (a *publicDNSActuator) CanCreateDomain() bool {
	return false
}

func (a *publicDNSActuator) CanListZones() bool {
	return false
}

func (a *publicDNSActuator) CreateDomain(fqdn string) error {
	return errPublicDNSReadOnly
}

func (a *publicDNSActuator) GetZoneRecords(domain string) ([]happydns.Record, error) {
	records, _, err := a.GetZoneRecordsWithContext(context.Background(), domain, nil)
	return records, err
}

func (a *publicDNSActuator) GetZoneRecordsWithContext(ctx context.Context, domain string, known []string) ([]happydns.Record, bool, error) {
	records, complete, err := explore(ctx, domain, known, a.sources, a.opts)
	return records, !complete, err
}

// TestDomainExistence only looks for the name servers of the domain: the
// exploration is left to the retrieval of the zone.
func (a *publicDNSActuator) TestDomainExistence(ctx context.Context, domain string) error {
	_, err := discovery.Locate(ctx, domain, a.opts)
	return err
}

func (a *publicDNSActuator) GetZoneCorrections(domain string, wantedRecords []happydns.Record) ([]*happydns.Correction, int, error) {
	return nil, 0, errPublicDNSReadOnly
}

func (a *publicDNSActuator) ListZones() ([]string, error) {
	return nil, errPublicDNSReadOnly
}

func init() {
	providerReg.RegisterProvider(func() happydns.ProviderBody {
		return &PublicDNSDiscovery{}
	}, happydns.ProviderInfos{
		Name:        "Public DNS discovery",
		Description: "For domains hosted elsewhere: happyDomain recomposes the zone from what its name servers publicly answer, to monitor it. The zone is read-only, and may miss the names that nothing points to.",
	})
}
