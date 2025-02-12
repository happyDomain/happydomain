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
	"crypto/sha1"
	"fmt"
	"io"

	"github.com/StackExchange/dnscontrol/v4/models"
)

// Service represents a service provided by one or more DNS record.
type Service interface {
	// GetNbResources get the number of main Resources contains in the Service.
	GetNbResources() int

	// GenComment sum up the content of the Service, in a small usefull string.
	GenComment(origin string) string

	// genRRs generates corresponding RRs.
	GenRRs(domain string, ttl uint32, origin string) (models.Records, error)
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
type ServiceCombined struct {
	Service
	ServiceMeta
}

// UnmarshalServiceJSON stores a functor defined in services/interfaces.go that
// can't be defined here due to cyclic imports.
var UnmarshalServiceJSON func(*ServiceCombined, []byte) error

// UnmarshalJSON points to the implementation of the UnmarshalJSON function for
// the encoding/json module.
func (svc *ServiceCombined) UnmarshalJSON(b []byte) error {
	return UnmarshalServiceJSON(svc, b)
}

func ValidateService(svc Service, subdomain, origin string) ([]byte, error) {
	records, err := svc.GenRRs(subdomain, 0, origin)
	if err != nil {
		return nil, fmt.Errorf("unable to generate records: %w", err)
	} else if len(records) == 0 {
		return nil, fmt.Errorf("no record can be generated from your service.")
	} else {
		hash := sha1.New()
		io.WriteString(hash, records[0].String())

		return hash.Sum(nil), nil
	}
}
