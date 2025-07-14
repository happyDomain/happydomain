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
	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type AddRecordUsecase struct {
	serviceListRecordsUC *service.ListRecordsUsecase
}

func NewAddRecordUsecase(serviceListRecordsUC *service.ListRecordsUsecase) *AddRecordUsecase {
	return &AddRecordUsecase{
		serviceListRecordsUC: serviceListRecordsUC,
	}
}

func (uc *AddRecordUsecase) Add(zone *happydns.Zone, origin string, record happydns.Record) error {
	record = helpers.CopyRecord(record)

	record.Header().Name = helpers.DomainFQDN(record.Header().Name, origin)

	// Research the service in which the record should be found
	newsvc, _, err := svcs.AnalyzeZone(origin, []happydns.Record{record})
	if err != nil {
		return err
	}

	for dn := range newsvc {
		for _, newsvctype := range newsvc[dn] {
			// Is there such kind of service in the subdomain?
			var foundsamesvc *happydns.Service
			for i, s := range zone.Services[dn] {
				if s.Type == newsvctype.Type {
					foundsamesvc = s

					// Export service related records
					svc_rrs, err := uc.serviceListRecordsUC.List(foundsamesvc, origin, 0)
					if err != nil {
						return err
					}

					svc_rrs = append([]happydns.Record{record}, svc_rrs...)

					// Recreate the service
					newsvc, _, err := svcs.AnalyzeZone(origin, svc_rrs)
					if err != nil {
						return err
					}

					// Replace in zone
					zone.Services[dn] = append(zone.Services[dn][:i], append(newsvc[dn], zone.Services[dn][i+1:]...)...)

					break
				}
			}

			if foundsamesvc == nil {
				// Register in zone
				zone.Services[dn] = append(zone.Services[dn], newsvc[dn]...)
			}
		}
	}

	return nil
}
