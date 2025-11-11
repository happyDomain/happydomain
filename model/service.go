// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

package happydns

import (
	"encoding/json"
)

// ServiceBody represents a service provided by one or more DNS record.
type ServiceBody interface {
	// GetNbResources get the number of main Resources contains in the Service.
	GetNbResources() int

	// GenComment sum up the content of the Service, in a small usefull string.
	GenComment() string

	// GetRecords retrieves underlying RRs.
	GetRecords(domain string, ttl uint32, origin string) ([]Record, error)
}

func (svc *Service) Meta() (meta ServiceMeta) {
	meta.Type = svc.Type
	meta.UnderscoreId = svc.UnderscoreId
	meta.UnderscoreOwnerid = svc.UnderscoreOwnerid
	meta.Domain = svc.Domain
	meta.Ttl = svc.Ttl
	meta.Comment = svc.Comment
	meta.UserComment = svc.UserComment
	meta.Aliases = svc.Aliases
	meta.NbResources = svc.NbResources

	return
}

func (svc *Service) SetMeta(meta *ServiceMeta) {
	svc.Type = meta.Type
	svc.UnderscoreId = meta.UnderscoreId
	svc.UnderscoreOwnerid = meta.UnderscoreOwnerid
	svc.Domain = meta.Domain
	svc.Ttl = meta.Ttl
	svc.Comment = meta.Comment
	svc.UserComment = meta.UserComment
	svc.Aliases = meta.Aliases
	svc.NbResources = meta.NbResources
}

// ServiceCombined combined ServiceMeta + Service
type ServiceMessage struct {
	ServiceMeta
	Service json.RawMessage
}

func (msg *ServiceMessage) Meta() *ServiceMeta {
	return &msg.ServiceMeta
}

type ServiceRecord struct {
	Type   string `json:"type"`
	String string `json:"str"`
	RR     Record `json:"rr,omitempty"`
}

type ServiceUsecase interface {
	ListRecords(*Domain, *Zone, *Service) ([]Record, error)
	ValidateService(ServiceBody, string, Origin) ([]byte, error)
}
