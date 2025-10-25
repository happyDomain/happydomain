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

package orchestrator

import (
	domainlogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

// DomainUpdater is an interface for updating domains.
type DomainUpdater interface {
	Update(domainID happydns.Identifier, user *happydns.User, updateFn func(*happydns.Domain)) error
}

type Orchestrator struct {
	RemoteZoneImporter    *RemoteZoneImporterUsecase
	ZoneCorrectionApplier *ZoneCorrectionApplierUsecase
	ZoneImporter          *ZoneImporterUsecase
}

func NewOrchestrator(
	appendDomainLog domainlogUC.DomainLogAppender,
	domainUpdater DomainUpdater,
	getProvider *providerUC.GetProviderUsecase,
	listRecords *zoneUC.ListRecordsUsecase,
	zoneCorrector *providerUC.ZoneCorrectorUsecase,
	zoneCreator *zoneUC.CreateZoneUsecase,
	zoneRetriever *providerUC.ZoneRetrieverUsecase,
	zoneUpdater *zoneUC.UpdateZoneUsecase,
) *Orchestrator {
	zoneImporter := NewZoneImporterUsecase(domainUpdater, zoneCreator)
	return &Orchestrator{
		RemoteZoneImporter:    NewRemoteZoneImporterUsecase(appendDomainLog, getProvider, zoneImporter, zoneRetriever),
		ZoneCorrectionApplier: NewZoneCorrectionApplierUsecase(appendDomainLog, domainUpdater, getProvider, listRecords, zoneCorrector, zoneCreator, zoneUpdater),
		ZoneImporter:          zoneImporter,
	}
}
