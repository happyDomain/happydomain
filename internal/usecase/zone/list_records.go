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

package zone

import (
	"fmt"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
)

type ListRecordsUsecase struct {
	serviceListRecordsUC *service.ListRecordsUsecase
}

func NewListRecordsUsecase(serviceListRecordsUC *service.ListRecordsUsecase) *ListRecordsUsecase {
	return &ListRecordsUsecase{
		serviceListRecordsUC: serviceListRecordsUC,
	}
}

func (uc *ListRecordsUsecase) ToZoneFile(domain *happydns.Domain, zone *happydns.Zone) (string, error) {
	records, err := uc.List(domain, zone)
	if err != nil {
		return "", happydns.InternalError{
			Err: fmt.Errorf("unable to retrieve records for old zone: %w", err),
		}
	}

	var ret string

	for _, rr := range records {
		ret += rr.String() + "\n"
	}

	return ret, nil
}

func (uc *ListRecordsUsecase) List(domain *happydns.Domain, zone *happydns.Zone) (rrs []happydns.Record, err error) {
	var svc_rrs []happydns.Record

	for _, svcs := range zone.Services {
		for _, svc := range svcs {
			svc_rrs, err = uc.serviceListRecordsUC.List(svc, domain.Domain, zone.DefaultTtl)
			if err != nil {
				return
			}
			rrs = append(rrs, svc_rrs...)
		}

		// Ensure SOA is the first record
		for i, rr := range rrs {
			if rr.Header().Rrtype == dns.TypeSOA {
				rrs[0], rrs[i] = rrs[i], rrs[0]
				break
			}
		}
	}

	return
}
