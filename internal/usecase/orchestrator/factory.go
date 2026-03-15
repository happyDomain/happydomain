// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

// Package orchestrator wires together lower-level use-cases to implement the
// multi-step workflows that span provider access, zone storage, and domain
// history management.  It sits between the HTTP/API layer and the individual
// domain/zone use-cases, coordinating the sequence of operations required to
// import, diff, and publish DNS zones.
package orchestrator

import (
	"context"

	domainlogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

// DomainUpdater is an interface for updating domains.
type DomainUpdater interface {
	Update(domainID happydns.Identifier, user *happydns.User, updateFn func(*happydns.Domain)) error
}

// ProviderGetter is an interface for getting providers.
type ProviderGetter interface {
	GetUserProvider(user *happydns.User, providerID happydns.Identifier) (*happydns.Provider, error)
}

// ZoneRetriever is an interface for retrieving zones from providers.
type ZoneRetriever interface {
	RetrieveZone(ctx context.Context, provider *happydns.Provider, name string) ([]happydns.Record, error)
}

// ZoneCorrector is an interface for getting zone corrections.
type ZoneCorrector interface {
	ListZoneCorrections(ctx context.Context, provider *happydns.Provider, domain *happydns.Domain, records []happydns.Record) ([]*happydns.Correction, int, error)
}

// Orchestrator aggregates the use-cases that together implement the DNS zone
// lifecycle: importing zones from a provider, listing required corrections, and
// applying those corrections back to the provider.
type Orchestrator struct {
	// RemoteZoneImporter fetches a live zone from the provider and stores it.
	RemoteZoneImporter *RemoteZoneImporterUsecase
	// ZoneCorrectionApplier lists and applies the corrections needed to bring
	// the provider in sync with the desired zone state.
	ZoneCorrectionApplier *ZoneCorrectionApplierUsecase
	// ZoneImporter converts a flat list of DNS records into a happyDomain zone
	// and persists it in the domain history.
	ZoneImporter *ZoneImporterUsecase
}

// NewOrchestrator constructs an Orchestrator by wiring up all required
// dependencies.  It builds the shared ZoneImporterUsecase and
// ZoneCorrectionListerUsecase internally so callers do not need to manage
// those intermediate objects.
func NewOrchestrator(
	appendDomainLog domainlogUC.DomainLogAppender,
	domainUpdater DomainUpdater,
	providerService ProviderGetter,
	listRecords *zoneUC.ListRecordsUsecase,
	zoneCorrectorService ZoneCorrector,
	zoneCreator *zoneUC.CreateZoneUsecase,
	zoneGetter *zoneUC.GetZoneUsecase,
	zoneRetrieverService ZoneRetriever,
	zoneUpdater *zoneUC.UpdateZoneUsecase,
) *Orchestrator {
	zoneImporter := NewZoneImporterUsecase(domainUpdater, zoneCreator, zoneGetter)
	zoneCorrectionLister := NewZoneCorrectionListerUsecase(providerService, listRecords, zoneCorrectorService)
	return &Orchestrator{
		RemoteZoneImporter:    NewRemoteZoneImporterUsecase(appendDomainLog, providerService, zoneImporter, zoneRetrieverService),
		ZoneCorrectionApplier: NewZoneCorrectionApplierUsecase(appendDomainLog, domainUpdater, zoneCorrectionLister, zoneCreator, zoneUpdater),
		ZoneImporter:          zoneImporter,
	}
}
