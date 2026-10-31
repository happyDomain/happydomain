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

package zone

import (
	"fmt"

	"git.happydns.org/happyDomain/internal/helpers"
	intsvc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	"git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
)

// UpdateRecordUsecase handles replacing a single DNS record of an in-memory
// Zone by another one, keeping the service that owns it in place.
type UpdateRecordUsecase struct {
	addRecordUC           *AddRecordUsecase
	deleteRecordUC        *DeleteRecordUsecase
	serviceListRecordsUC  *service.ListRecordsUsecase
	serviceSearchRecordUC *service.SearchRecordUsecase
}

// NewUpdateRecordUsecase constructs an UpdateRecordUsecase. The add and
// delete use cases are used when the record moves to another subdomain.
func NewUpdateRecordUsecase(addRecordUC *AddRecordUsecase, deleteRecordUC *DeleteRecordUsecase, serviceListRecordsUC *service.ListRecordsUsecase, serviceSearchRecordUC *service.SearchRecordUsecase) *UpdateRecordUsecase {
	return &UpdateRecordUsecase{
		addRecordUC:           addRecordUC,
		deleteRecordUC:        deleteRecordUC,
		serviceListRecordsUC:  serviceListRecordsUC,
		serviceSearchRecordUC: serviceSearchRecordUC,
	}
}

// Update replaces oldRecord by newRecord in zone. The records of the service
// owning oldRecord are re-analysed with newRecord in its place, so that the
// service keeps its position and its metadata, even when its type changes
// (e.g. abstract.Origin). When newRecord lands on another subdomain, it is
// removed from its service and added among the services of its new name.
func (uc *UpdateRecordUsecase) Update(zone *happydns.Zone, origin string, oldRecord, newRecord happydns.Record) error {
	dn, svc, err := uc.serviceSearchRecordUC.Search(zone, oldRecord)
	if err != nil {
		return err
	}
	if svc == nil {
		return fmt.Errorf("unable to update record: record not found")
	}

	svc_rrs, err := uc.serviceListRecordsUC.List(svc, origin, 0)
	if err != nil {
		return err
	}

	oldAbs := helpers.RRAbsolute(helpers.CopyRecord(oldRecord), origin)
	newAbs := helpers.CopyRecord(newRecord)
	newAbs.Header().Name = helpers.DomainFQDN(newAbs.Header().Name, origin)

	rr_found := false
	for i, svc_rr := range svc_rrs {
		if svc_rr.String() == oldAbs.String() {
			svc_rrs[i] = newAbs
			rr_found = true
			break
		}
	}
	if !rr_found {
		return fmt.Errorf("unable to find record")
	}

	newsvcs, _, err := intsvc.AnalyzeZone(origin, svc_rrs)
	if err != nil {
		return err
	}

	for newdn := range newsvcs {
		if newdn != dn {
			// The record moves to another subdomain
			if err := uc.deleteRecordUC.Delete(zone, origin, oldRecord); err != nil {
				return err
			}
			return uc.addRecordUC.Add(zone, origin, newRecord)
		}
	}

	replaceService(zone, dn, svc, newsvcs[dn], origin)

	if svc.Type == "svcs.Orphan" {
		return uc.deleteRecordUC.ReanalyzeOrphan(zone, origin, dn)
	}

	return nil
}

// replaceService puts newsvcs in place of svc among the services of dn, after
// giving them the metadata of svc. A single new service takes them whatever
// its type: it holds the records of svc, a type change (e.g. an origin losing
// its SOA) does not make it another service.
func replaceService(zone *happydns.Zone, dn happydns.Subdomain, svc *happydns.Service, newsvcs []*happydns.Service, origin string) {
	if len(newsvcs) == 1 {
		transferMetadata(svc, newsvcs[0], origin, 0)
	} else {
		ReassociateMetadata(
			map[happydns.Subdomain][]*happydns.Service{dn: {svc}},
			map[happydns.Subdomain][]*happydns.Service{dn: newsvcs},
			origin,
			0,
		)
	}

	for i, s := range zone.Services[dn] {
		if s.Id.Equals(svc.Id) {
			zone.Services[dn] = append(zone.Services[dn][:i], append(newsvcs, zone.Services[dn][i+1:]...)...)
			break
		}
	}
}
