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
	intsvc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	"git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
)

// AddRecordUsecase handles adding a single DNS record to an in-memory Zone,
// merging it into an existing compatible service or registering a new one.
type AddRecordUsecase struct {
	serviceListRecordsUC *service.ListRecordsUsecase
}

// NewAddRecordUsecase constructs an AddRecordUsecase with the given service
// record-listing dependency.
func NewAddRecordUsecase(serviceListRecordsUC *service.ListRecordsUsecase) *AddRecordUsecase {
	return &AddRecordUsecase{
		serviceListRecordsUC: serviceListRecordsUC,
	}
}

// Add inserts record into zone under origin. The record name is first
// qualified to a FQDN. If a service of the same type already exists under the
// target subdomain, the record is merged into that service and the service is
// re-analysed. Otherwise, it is merged into the first service of the subdomain
// that absorbs it (e.g. a NS record into an abstract.Origin, a SOA into an
// abstract.NSOnlyOrigin); failing that, a new service is appended.
func (uc *AddRecordUsecase) Add(zone *happydns.Zone, origin string, record happydns.Record) error {
	record = helpers.CopyRecord(record)

	record.Header().Name = helpers.DomainFQDN(record.Header().Name, origin)

	// Research the service in which the record should be found
	newsvc, _, err := intsvc.AnalyzeZone(origin, []happydns.Record{record})
	if err != nil {
		return err
	}

	for dn := range newsvc {
		for _, newsvctype := range newsvc[dn] {
			merged := false

			// Is there such kind of service in the subdomain?
			for _, s := range zone.Services[dn] {
				if s.Type == newsvctype.Type {
					mergedsvc, err := uc.merge(s, origin, record)
					if err != nil {
						return err
					}

					replaceService(zone, dn, s, mergedsvc[dn], origin)
					merged = true
					break
				}
			}

			// Is there a service that the record completes?
			for _, s := range zone.Services[dn] {
				if merged {
					break
				}

				mergedsvc, err := uc.merge(s, origin, record)
				if err != nil {
					return err
				}

				if len(mergedsvc) == 1 && len(mergedsvc[dn]) == 1 {
					replaceService(zone, dn, s, mergedsvc[dn], origin)
					merged = true
				}
			}

			if !merged {
				// Register in zone
				zone.Services[dn] = append(zone.Services[dn], newsvctype)
			}
		}
	}

	return nil
}

// merge analyses the records of svc along with record.
func (uc *AddRecordUsecase) merge(svc *happydns.Service, origin string, record happydns.Record) (map[happydns.Subdomain][]*happydns.Service, error) {
	// Export service related records
	svc_rrs, err := uc.serviceListRecordsUC.List(svc, origin, 0)
	if err != nil {
		return nil, err
	}

	svc_rrs = append([]happydns.Record{record}, svc_rrs...)

	// Recreate the service
	mergedsvc, _, err := intsvc.AnalyzeZone(origin, svc_rrs)
	return mergedsvc, err
}
