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
	"git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
)

type Service struct {
	AddRecordUC    *AddRecordUsecase
	CreateZoneUC   *CreateZoneUsecase
	DeleteRecordUC *DeleteRecordUsecase
	DeleteZoneUC   *DeleteZoneUsecase
	DiffZoneUC     *ZoneDifferUsecase
	GetZoneUC      *GetZoneUsecase
	ListRecordsUC  *ListRecordsUsecase
	UpdateZoneUC   *UpdateZoneUsecase
}

func NewZoneUsecases(store ZoneStorage, serviceUC *service.Service) *Service {
	getZone := NewGetZoneUsecase(store)
	listRecords := NewListRecordsUsecase(serviceUC.ListRecordsUC)

	return &Service{
		AddRecordUC:    NewAddRecordUsecase(serviceUC.ListRecordsUC),
		CreateZoneUC:   NewCreateZoneUsecase(store),
		DeleteRecordUC: NewDeleteRecordUsecase(serviceUC.ListRecordsUC, serviceUC.SearchRecordUC),
		DeleteZoneUC:   NewDeleteZoneUsecase(store),
		DiffZoneUC:     NewZoneDifferUsecase(getZone, listRecords),
		GetZoneUC:      getZone,
		ListRecordsUC:  listRecords,
		UpdateZoneUC:   NewUpdateZoneUsease(store, getZone),
	}
}

func (s *Service) AddRecord(zone *happydns.Zone, origin string, record happydns.Record) error {
	return s.AddRecordUC.Add(zone, origin, record)
}

func (s *Service) CreateZone(zone *happydns.Zone) error {
	return s.CreateZoneUC.Create(zone)
}

func (s *Service) DeleteZone(zoneid happydns.Identifier) error {
	return s.DeleteZoneUC.Delete(zoneid)
}

func (s *Service) DeleteRecord(zone *happydns.Zone, origin string, record happydns.Record) error {
	return s.DeleteRecordUC.Delete(zone, origin, record)
}

func (s *Service) DiffZones(domain *happydns.Domain, newZone *happydns.Zone, oldZoneID happydns.Identifier) ([]*happydns.Correction, error) {
	return s.DiffZoneUC.Diff(domain, newZone, oldZoneID)
}

func (s *Service) FlattenZoneFile(domain *happydns.Domain, zone *happydns.Zone) (string, error) {
	return s.ListRecordsUC.ToZoneFile(domain, zone)
}

func (s *Service) GenerateRecords(domain *happydns.Domain, zone *happydns.Zone) ([]happydns.Record, error) {
	return s.ListRecordsUC.List(domain, zone)
}

func (s *Service) GetZone(zoneid happydns.Identifier) (*happydns.Zone, error) {
	return s.GetZoneUC.Get(zoneid)
}

func (s *Service) GetZoneMeta(zoneid happydns.Identifier) (*happydns.ZoneMeta, error) {
	return s.GetZoneUC.GetMeta(zoneid)
}

func (s *Service) LoadZoneFromId(domain *happydns.Domain, id happydns.Identifier) (*happydns.Zone, error) {
	return s.GetZoneUC.GetInDomain(id, domain)
}

func (s *Service) UpdateZone(zoneID happydns.Identifier, updateFn func(*happydns.Zone)) error {
	return s.UpdateZoneUC.Update(zoneID, updateFn)
}
