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
	"fmt"

	domainlogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	"git.happydns.org/happyDomain/model"
)

// RemoteZoneImporterUsecase fetches the live DNS records for a domain directly
// from the provider and delegates to ZoneImporterUsecase to persist them.  It
// also appends a domain log entry on success.
type RemoteZoneImporterUsecase struct {
	appendDomainLog domainlogUC.DomainLogAppender
	providerService ProviderGetter
	zoneImporter    *ZoneImporterUsecase
	zoneRetriever   ZoneRetriever
}

// NewRemoteZoneImporterUsecase creates a RemoteZoneImporterUsecase wired to
// the given log appender, provider getter, zone importer, and zone retriever.
func NewRemoteZoneImporterUsecase(
	appendDomainLog domainlogUC.DomainLogAppender,
	providerService ProviderGetter,
	zoneImporter *ZoneImporterUsecase,
	zoneRetriever ZoneRetriever,
) *RemoteZoneImporterUsecase {
	return &RemoteZoneImporterUsecase{
		appendDomainLog: appendDomainLog,
		providerService: providerService,
		zoneImporter:    zoneImporter,
		zoneRetriever:   zoneRetriever,
	}
}

// Import resolves the provider for the domain, retrieves its current records,
// and imports them via ZoneImporterUsecase.  A domain log entry is appended on
// success.  Returns the newly created zone or an error.
func (uc *RemoteZoneImporterUsecase) Import(user *happydns.User, domain *happydns.Domain) (*happydns.Zone, error) {
	provider, err := uc.providerService.GetUserProvider(user, domain.ProviderId)
	if err != nil {
		return nil, err
	}

	zone, err := uc.zoneRetriever.RetrieveZone(provider, domain.DomainName)
	if err != nil {
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("unable to retrieve the zone from server: %s", err.Error())}
	}

	// import
	myZone, err := uc.zoneImporter.Import(user, domain, zone)
	if err != nil {
		return nil, err
	}

	if uc.appendDomainLog != nil {
		uc.appendDomainLog.AppendDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_INFO, fmt.Sprintf("Zone imported from provider API: %s", myZone.Id.String())))
	}

	return myZone, nil
}
