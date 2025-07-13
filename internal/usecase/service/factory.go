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
	"git.happydns.org/happyDomain/model"
)

type Service struct {
	ListRecordsUC     *ListRecordsUsecase
	ValidateServiceUC *ValidateServiceUsecase
}

func NewServiceUsecases() *Service {
	return &Service{
		ListRecordsUC:     NewListRecordsUsecase(),
		ValidateServiceUC: NewValidateServiceUsecase(),
	}
}

func (s *Service) ListRecords(domain *happydns.Domain, zone *happydns.Zone, service *happydns.Service) ([]happydns.Record, error) {
	return s.ListRecordsUC.List(service, domain.DomainName, zone.DefaultTTL)
}

func (s *Service) ValidateService(body happydns.ServiceBody, subdomain happydns.Subdomain, origin happydns.Origin) ([]byte, error) {
	return s.ValidateServiceUC.Validate(body, subdomain, origin)
}
