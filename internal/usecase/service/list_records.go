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

package service

import (
	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

type ListRecordsUsecase struct{}

func NewListRecordsUsecase() *ListRecordsUsecase {
	return &ListRecordsUsecase{}
}

func (uc *ListRecordsUsecase) List(domain *happydns.Domain, zone *happydns.Zone, svc *happydns.Service) ([]happydns.Record, error) {
	ttl := zone.DefaultTTL
	if svc.Ttl != 0 {
		ttl = svc.Ttl
	}

	records, err := svc.Service.GetRecords(svc.Domain, ttl, domain.DomainName)
	if err != nil {
		return nil, err
	}

	for i, record := range records {
		records[i].Header().Name = helpers.DomainJoin(record.Header().Name, svc.Domain, domain.DomainName)
		if record.Header().Ttl == 0 {
			record.Header().Ttl = ttl
		}
	}

	return records, nil
}
