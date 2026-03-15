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

// Package zone implements the use cases for DNS zone management in
// happyDomain.  It covers the full zone lifecycle:
//
//   - CreateZoneUsecase / DeleteZoneUsecase – storage-level zone creation and
//     removal.
//   - GetZoneUsecase – retrieval by ID, with optional domain-history validation.
//   - UpdateZoneUsecase – functional update pattern (fetch → mutate → save).
//   - ListRecordsUsecase – flattens the service tree of a zone into a list of
//     raw DNS records; ToZoneFile renders them as a standard zone file.
//   - AddRecordUsecase / DeleteRecordUsecase – individual record-level mutations
//     that re-analyse affected services and keep the zone consistent.
//   - ZoneDifferUsecase – computes the corrections between two zone snapshots.
//   - ReassociateMetadata – transfers user-supplied metadata (comments, IDs,
//     aliases, TTL overrides) from old services to newly re-analysed ones.
//
// The Service facade wires all of the above together and is the primary entry
// point consumed by the orchestrator and HTTP handler layers.
package zone

import (
	"git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
)

// Service is the facade that wires all zone-level use cases together. It is
// the primary entry point consumed by the orchestrator and HTTP handler layers.
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

// NewZoneUsecases constructs a Service by wiring the storage and service-level
// use cases into all zone-level use cases.
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

// AddRecord adds a single DNS record to zone, merging it into the appropriate
// existing service or creating a new one when no compatible service is found.
func (s *Service) AddRecord(zone *happydns.Zone, origin string, record happydns.Record) error {
	return s.AddRecordUC.Add(zone, origin, record)
}

// CreateZone persists a new zone in storage.
func (s *Service) CreateZone(zone *happydns.Zone) error {
	return s.CreateZoneUC.Create(zone)
}

// DeleteZone removes the zone identified by zoneid from storage.
func (s *Service) DeleteZone(zoneid happydns.Identifier) error {
	return s.DeleteZoneUC.Delete(zoneid)
}

// DeleteRecord removes a single DNS record from zone, re-analysing affected
// services to keep the zone consistent.
func (s *Service) DeleteRecord(zone *happydns.Zone, origin string, record happydns.Record) error {
	return s.DeleteRecordUC.Delete(zone, origin, record)
}

// DiffZones computes the corrections needed to go from the zone identified by
// oldZoneID to newZone for the given domain.
func (s *Service) DiffZones(domain *happydns.Domain, newZone *happydns.Zone, oldZoneID happydns.Identifier) ([]*happydns.Correction, error) {
	return s.DiffZoneUC.Diff(domain, newZone, oldZoneID)
}

// FlattenZoneFile renders all records of the zone as a standard zone-file
// string, with the SOA record first.
func (s *Service) FlattenZoneFile(domain *happydns.Domain, zone *happydns.Zone) (string, error) {
	return s.ListRecordsUC.ToZoneFile(domain, zone)
}

// GenerateRecords expands the service tree of zone into a flat list of raw DNS
// records, merging SPF contributions.
func (s *Service) GenerateRecords(domain *happydns.Domain, zone *happydns.Zone) ([]happydns.Record, error) {
	return s.ListRecordsUC.List(domain, zone)
}

// GetZone retrieves the fully parsed Zone for the given identifier.
func (s *Service) GetZone(zoneid happydns.Identifier) (*happydns.Zone, error) {
	return s.GetZoneUC.Get(zoneid)
}

// GetZoneMeta retrieves only the metadata portion of the zone identified by
// zoneid, without deserialising its service tree.
func (s *Service) GetZoneMeta(zoneid happydns.Identifier) (*happydns.ZoneMeta, error) {
	return s.GetZoneUC.GetMeta(zoneid)
}

// LoadZoneFromId retrieves the zone identified by id and validates that it
// belongs to the given domain's history before returning it.
func (s *Service) LoadZoneFromId(domain *happydns.Domain, id happydns.Identifier) (*happydns.Zone, error) {
	return s.GetZoneUC.GetInDomain(id, domain)
}

// UpdateZone fetches the zone identified by zoneID, applies updateFn to it,
// and persists the result. The zone identifier must remain unchanged.
func (s *Service) UpdateZone(zoneID happydns.Identifier, updateFn func(*happydns.Zone)) error {
	return s.UpdateZoneUC.Update(zoneID, updateFn)
}
