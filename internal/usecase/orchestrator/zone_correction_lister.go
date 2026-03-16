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

package orchestrator

import (
	"context"

	adapter "git.happydns.org/happyDomain/internal/adapters"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

// ZoneCorrectionListerUsecase computes the list of corrections needed to
// synchronize a zone's desired state with the records currently published by
// the provider. It fetches provider records, expands the WIP zone to records,
// and computes a local diff without executable closures.
type ZoneCorrectionListerUsecase struct {
	providerService ProviderGetter
	listRecords     *zoneUC.ListRecordsUsecase
	zoneCorrector   ZoneCorrector
	zoneRetriever   ZoneRetriever
}

// NewZoneCorrectionListerUsecase creates a ZoneCorrectionListerUsecase with
// the given provider getter, record lister, zone corrector, and zone retriever.
func NewZoneCorrectionListerUsecase(
	providerService ProviderGetter,
	listRecords *zoneUC.ListRecordsUsecase,
	zoneCorrector ZoneCorrector,
	zoneRetriever ZoneRetriever,
) *ZoneCorrectionListerUsecase {
	return &ZoneCorrectionListerUsecase{
		providerService: providerService,
		listRecords:     listRecords,
		zoneCorrector:   zoneCorrector,
		zoneRetriever:   zoneRetriever,
	}
}

// listWithRecords is the internal implementation that returns the corrections
// along with the provider and WIP records used to compute them.
func (uc *ZoneCorrectionListerUsecase) listWithRecords(
	ctx context.Context,
	user *happydns.User,
	domain *happydns.Domain,
	zone *happydns.Zone,
) ([]*happydns.Correction, []happydns.Record, []happydns.Record, int, error) {
	provider, err := uc.providerService.GetUserProvider(user, domain.ProviderId)
	if err != nil {
		return nil, nil, nil, 0, err
	}

	providerRecords, err := uc.zoneRetriever.RetrieveZone(ctx, provider, domain.DomainName)
	if err != nil {
		return nil, nil, nil, 0, err
	}

	wipRecords, err := uc.listRecords.List(domain, zone)
	if err != nil {
		return nil, nil, nil, 0, err
	}

	corrections, nbDiffs, err := adapter.DNSControlDiffByRecord(providerRecords, wipRecords, domain.DomainName)
	if err != nil {
		return nil, nil, nil, nbDiffs, err
	}

	return corrections, providerRecords, wipRecords, nbDiffs, nil
}

// List returns the corrections required to bring the provider's live DNS
// records in line with the given zone. It fetches the current provider
// records, expands the zone into individual records, and computes the diff
// locally. The second return value is the total number of corrections.
func (uc *ZoneCorrectionListerUsecase) List(
	ctx context.Context,
	user *happydns.User,
	domain *happydns.Domain,
	zone *happydns.Zone,
) ([]*happydns.Correction, int, error) {
	corrections, _, _, nbDiffs, err := uc.listWithRecords(ctx, user, domain, zone)
	return corrections, nbDiffs, err
}
