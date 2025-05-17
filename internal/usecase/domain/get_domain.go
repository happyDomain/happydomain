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

package domain

import (
	"errors"
	"fmt"

	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

type GetDomainUsecase struct {
	store       DomainStorage
	zoneService *zoneUC.GetZoneUsecase
}

func NewGetDomainUsecase(store DomainStorage, zoneService *zoneUC.GetZoneUsecase) *GetDomainUsecase {
	return &GetDomainUsecase{
		store:       store,
		zoneService: zoneService,
	}
}

func (uc *GetDomainUsecase) ExtendsDomainWithZoneMeta(domain *happydns.Domain) (*happydns.DomainWithZoneMetadata, error) {
	var errs error
	ret := map[string]*happydns.ZoneMeta{}

	for _, zm := range domain.ZoneHistory {
		zoneMeta, err := uc.zoneService.GetMeta(zm)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("unable to retrieve zone meta history for %q: %w", domain.DomainName, err))
		} else {
			ret[zm.String()] = zoneMeta
		}
	}

	return &happydns.DomainWithZoneMetadata{
		Domain:   domain,
		ZoneMeta: ret,
	}, errs
}

func (uc *GetDomainUsecase) ByID(user *happydns.User, did happydns.Identifier) (*happydns.Domain, error) {
	domain, err := uc.store.GetDomain(did)
	if err != nil {
		return nil, err
	}

	if !user.Id.Equals(domain.Owner) {
		return nil, happydns.ErrDomainNotFound
	}

	return domain, nil
}

func (uc *GetDomainUsecase) ByFQDN(user *happydns.User, fqdn string) ([]*happydns.Domain, error) {
	return uc.store.GetDomainByDN(user, fqdn)
}
