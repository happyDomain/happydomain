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
	"strings"

	"git.happydns.org/happyDomain/model"
)

type SearchRecordUsecase struct {
	serviceListRecordsUC *ListRecordsUsecase
}

func NewSearchRecordUsecase(serviceListRecordsUC *ListRecordsUsecase) *SearchRecordUsecase {
	return &SearchRecordUsecase{
		serviceListRecordsUC: serviceListRecordsUC,
	}
}

func (uc *SearchRecordUsecase) ExistsInService(svc *happydns.Service, record happydns.Record) (bool, error) {
	records, err := uc.serviceListRecordsUC.List(svc, "", 0)
	if err != nil {
		return false, err
	}

	for _, rr := range records {
		if record.Header().Name == rr.Header().Name &&
			record.Header().Rrtype == rr.Header().Rrtype &&
			record.Header().Class == rr.Header().Class &&
			strings.TrimPrefix(record.String(), record.Header().String()) == strings.TrimPrefix(rr.String(), rr.Header().String()) {
			return true, nil
		}
	}

	return false, nil
}

func (uc *SearchRecordUsecase) Search(zone *happydns.Zone, record happydns.Record) (happydns.Subdomain, *happydns.Service, error) {
	for dn, _ := range zone.Services {
		svc, err := uc.SearchInSubdomain(zone, dn, record)
		if err != nil || svc != nil {
			return dn, svc, err
		}
	}

	return "", nil, nil
}

func (uc *SearchRecordUsecase) SearchInSubdomain(zone *happydns.Zone, subdomain happydns.Subdomain, record happydns.Record) (*happydns.Service, error) {
	services, ok := zone.Services[subdomain]
	if !ok {
		return nil, nil
	}

	for _, svc := range services {
		exists, err := uc.ExistsInService(svc, record)
		if err != nil || exists {
			return svc, err
		}
	}

	return nil, nil
}
