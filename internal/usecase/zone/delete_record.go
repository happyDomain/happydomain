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
	"reflect"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type DeleteRecordUsecase struct {
	serviceListRecordsUC  *service.ListRecordsUsecase
	serviceSearchRecordUC *service.SearchRecordUsecase
}

func NewDeleteRecordUsecase(serviceListRecordsUC *service.ListRecordsUsecase, serviceSearchRecordUC *service.SearchRecordUsecase) *DeleteRecordUsecase {
	return &DeleteRecordUsecase{
		serviceListRecordsUC:  serviceListRecordsUC,
		serviceSearchRecordUC: serviceSearchRecordUC,
	}
}

func (uc *DeleteRecordUsecase) delete(zone *happydns.Zone, origin string, record happydns.Record, svc *happydns.Service, dn happydns.Subdomain) error {
	// Export service related records
	svc_rrs, err := uc.serviceListRecordsUC.List(svc, origin, 0)
	if err != nil {
		return err
	}

	record = helpers.RRAbsolute(record, origin)

	// Drop given record
	rr_found := false
	for i, svc_rr := range svc_rrs {
		if svc_rr.String() == record.String() {
			svc_rrs = append(svc_rrs[:i], svc_rrs[i+1:]...)
			rr_found = true
			break
		}
	}

	if !rr_found {
		return fmt.Errorf("unable to find record")
	}

	var newsvc map[happydns.Subdomain][]*happydns.Service

	if len(svc_rrs) > 0 {
		// Recreate the service
		newsvc, _, err = svcs.AnalyzeZone(origin, svc_rrs)
		if err != nil {
			return err
		}
	}

	// Register in zone
	for i, s := range zone.Services[dn] {
		if s.Id.Equals(svc.Id) {
			zone.Services[dn] = append(zone.Services[dn][:i], append(newsvc[dn], zone.Services[dn][i+1:]...)...)
			break
		}
	}

	return nil
}

func (uc *DeleteRecordUsecase) Delete(zone *happydns.Zone, origin string, record happydns.Record) error {
	dn, svc, err := uc.serviceSearchRecordUC.Search(zone, record)
	if err != nil {
		return err
	}
	if svc == nil {
		return fmt.Errorf("unable to delete record: record not found")
	}

	err = uc.delete(zone, origin, record, svc, dn)
	if err != nil {
		return err
	}

	if svc.Type == "svcs.Orphan" {
		err = uc.ReanalyzeOrphan(zone, origin, dn)
		if err != nil {
			return err
		}
	}

	return nil
}

func (uc *DeleteRecordUsecase) ReanalyzeOrphan(zone *happydns.Zone, origin string, dn happydns.Subdomain) error {
	var records []happydns.Record

	// Found all orphan records
	for _, svc := range zone.Services[dn] {
		if svc.Type == "svcs.Orphan" {
			svc_rrs, err := uc.serviceListRecordsUC.List(svc, origin, 0)
			if err != nil {
				return err
			}

			records = append(records, svc_rrs...)
		}
	}

	if len(records) == 0 {
		return nil
	}

	// Redo analysis
	newsvcs, _, err := svcs.AnalyzeZone(origin, records)
	if err != nil {
		return err
	}

	for dn, nsvcs := range newsvcs {
		for _, svc := range nsvcs {
			if reflect.Indirect(reflect.ValueOf(svc)).Type().String() != reflect.ValueOf(svcs.Orphan{}).Type().String() {
				svc_rrs, err := uc.serviceListRecordsUC.List(svc, origin, 0)
				if err != nil {
					return err
				}

				for _, record := range svc_rrs {
					err = uc.delete(zone, origin, record, svc, dn)
					if err != nil {
						return err
					}
				}

				zone.Services[dn] = append(zone.Services[dn], svc)
			}
		}
	}

	return nil
}
