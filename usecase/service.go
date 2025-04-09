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

package usecase

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type serviceUsecase struct{}

func NewServiceUsecase() happydns.ServiceUsecase {
	return &serviceUsecase{}
}

func (su *serviceUsecase) GetRecords(domain *happydns.Domain, zone *happydns.Zone, svc *happydns.Service) ([]happydns.Record, error) {
	ttl := zone.DefaultTTL
	if svc.Ttl != 0 {
		ttl = svc.Ttl
	}

	return svc.Service.GetRecords(svc.Domain, ttl, domain.DomainName)
}

func ParseService(msg *happydns.ServiceMessage) (svc *happydns.Service, err error) {
	svc = &happydns.Service{}

	svc.ServiceMeta = msg.ServiceMeta
	svc.Service, err = svcs.FindService(msg.Type)
	if err != nil {
		return
	}

	err = json.Unmarshal(msg.Service, &svc.Service)
	return
}

func (su *serviceUsecase) ValidateService(svc happydns.ServiceBody, subdomain, origin string) ([]byte, error) {
	rrs, err := svc.GetRecords(subdomain, 0, origin)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve records: %w", err)
	}

	if len(rrs) == 0 {
		return nil, fmt.Errorf("no record can be generated from your service.")
	} else {
		hash := sha1.New()
		for _, rr := range rrs {
			io.WriteString(hash, rr.String())
		}

		return hash.Sum(nil), nil
	}
}
