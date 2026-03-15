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

// Package service implements use cases that operate on individual DNS services
// (the logical groupings of records within a zone subdomain).  It provides:
//   - ListRecordsUsecase – expands a Service into its constituent DNS records.
//   - SearchRecordUsecase – locates the Service and subdomain that owns a given
//     record within a Zone.
//   - ValidateServiceUsecase – verifies that a ServiceBody can generate at least
//     one record and returns a SHA-1 hash of the resulting RDATA.
//   - ParseService – deserialises a ServiceMessage into a typed Service value.
//
// The Service facade wires these together and is the main entry point consumed
// by higher-level zone use cases.
package service

import (
	"git.happydns.org/happyDomain/model"
)

// Service is the facade for all service-level use cases.  Callers should use
// its methods rather than reaching into the embedded use-case structs directly.
type Service struct {
	ListRecordsUC     *ListRecordsUsecase
	SearchRecordUC    *SearchRecordUsecase
	ValidateServiceUC *ValidateServiceUsecase
}

// NewServiceUsecases wires and returns a ready-to-use Service facade.
func NewServiceUsecases() *Service {
	ListRecordsUC := NewListRecordsUsecase()

	return &Service{
		ListRecordsUC:     ListRecordsUC,
		SearchRecordUC:    NewSearchRecordUsecase(ListRecordsUC),
		ValidateServiceUC: NewValidateServiceUsecase(),
	}
}

// ListRecords expands the given service into its constituent DNS records,
// qualifying names relative to domain and applying the zone's default TTL.
func (s *Service) ListRecords(domain *happydns.Domain, zone *happydns.Zone, service *happydns.Service) ([]happydns.Record, error) {
	return s.ListRecordsUC.List(service, domain.DomainName, zone.DefaultTTL)
}

// ValidateService verifies that body generates at least one DNS record and
// returns a SHA-1 hash of the resulting RDATA for change-detection purposes.
func (s *Service) ValidateService(body happydns.ServiceBody, subdomain happydns.Subdomain, origin happydns.Origin) ([]byte, error) {
	return s.ValidateServiceUC.Validate(body, subdomain, origin)
}
