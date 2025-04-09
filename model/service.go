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

type Service struct {
	ServiceMeta
	Service ServiceBody
}

func (msg *Service) Meta() *ServiceMeta {
	return &msg.ServiceMeta
}

// ServiceBody represents a service provided by one or more DNS record.
type ServiceBody interface {
	// GetNbResources get the number of main Resources contains in the Service.
	GetNbResources() int

	// GenComment sum up the content of the Service, in a small usefull string.
	GenComment() string

	// GetRecords retrieves underlying RRs.
	GetRecords(domain string, ttl uint32, origin string) ([]Record, error)
}

// ServiceMeta holds the metadata associated to a Service.
type ServiceMeta struct {
	// Type is the string representation of the Service's type.
	Type string `json:"_svctype"`

	// Id is the Service's identifier.
	Id Identifier `json:"_id,omitempty" swaggertype:"string"`

	// OwnerId is the User's identifier for the current Service.
	OwnerId Identifier `json:"_ownerid,omitempty" swaggertype:"string"`

	// Domain contains the abstract domain where this Service relates.
	Domain string `json:"_domain"`

	// Ttl contains the specific TTL for the underlying Resources.
	Ttl uint32 `json:"_ttl"`

	// Comment is a string that helps user to distinguish the Service.
	Comment string `json:"_comment,omitempty"`

	// UserComment is a supplementary string defined by the user to
	// distinguish the Service.
	UserComment string `json:"_mycomment,omitempty"`

	// Aliases exposes the aliases defined on this Service.
	Aliases []string `json:"_aliases,omitempty"`

	// NbResources holds the number of Resources stored inside this Service.
	NbResources int `json:"_tmp_hint_nb"`
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
	GetRecords(*Domain, *Zone, *Service) ([]Record, error)
	ValidateService(ServiceBody, string, string) ([]byte, error)
}
