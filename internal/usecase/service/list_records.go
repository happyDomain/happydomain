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

func (uc *ListRecordsUsecase) List(svc *happydns.Service, origin string, defaultTTL uint32) ([]happydns.Record, error) {
	if svc.Ttl != 0 {
		defaultTTL = svc.Ttl
	}

	records, err := svc.Service.GetRecords(svc.Domain, defaultTTL, origin)
	if err != nil {
		return nil, err
	}

	for i, record := range records {
		records[i] = helpers.CopyRecord(record)

		records[i].Header().Name = helpers.DomainJoin(records[i].Header().Name, svc.Domain)
		records[i] = helpers.RRAbsolute(records[i], origin)

		if records[i].Header().Ttl == 0 {
			records[i].Header().Ttl = defaultTTL
		}
	}

	return records, nil
}
