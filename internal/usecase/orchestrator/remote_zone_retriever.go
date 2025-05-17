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
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
)

type RemoteZoneImporterUsecase struct {
	appendDomainLog *domainlogUC.CreateDomainLogUsecase
	getProvider     *providerUC.GetProviderUsecase
	zoneImporter    *ZoneImporterUsecase
	zoneRetriever   *providerUC.ZoneRetrieverUsecase
}

func NewRemoteZoneImporterUsecase(
	appendDomainLog *domainlogUC.CreateDomainLogUsecase,
	getProvider *providerUC.GetProviderUsecase,
	zoneImporter *ZoneImporterUsecase,
	zoneRetriever *providerUC.ZoneRetrieverUsecase,
) *RemoteZoneImporterUsecase {
	return &RemoteZoneImporterUsecase{
		appendDomainLog: appendDomainLog,
		getProvider:     getProvider,
		zoneImporter:    zoneImporter,
		zoneRetriever:   zoneRetriever,
	}
}

func (uc *RemoteZoneImporterUsecase) Import(user *happydns.User, domain *happydns.Domain) (*happydns.Zone, error) {
	provider, err := uc.getProvider.Get(user, domain.ProviderId)
	if err != nil {
		return nil, err
	}

	zone, err := uc.zoneRetriever.RetrieveCurrentZone(provider, domain.DomainName)
	if err != nil {
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("unable to retrieve the zone from server: %s", err.Error())}
	}

	// import
	myZone, err := uc.zoneImporter.Import(user, domain, zone)
	if err != nil {
		return nil, err
	}

	if uc.appendDomainLog != nil {
		uc.appendDomainLog.Create(domain, happydns.NewDomainLog(user, happydns.LOG_INFO, fmt.Sprintf("Zone imported from provider API: %s", myZone.Id.String())))
	}

	return myZone, nil
}
