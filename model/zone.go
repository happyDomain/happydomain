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
	"bytes"
	"fmt"
	"time"
)

// ZoneMeta holds the metadata associated to a Zone.
type ZoneMeta struct {
	// Id is the Zone's identifier.
	Id Identifier `json:"id" swaggertype:"string"`

	// IdAuthor is the User's identifier for the current Zone.
	IdAuthor Identifier `json:"id_author" swaggertype:"string"`

	// ParentZone identifies the parental zone of this one.
	ParentZone *Identifier `json:"parent,omitempty" swaggertype:"string"`

	// DefaultTTL is the TTL to use when no TTL has been defined for a record in this Zone.
	DefaultTTL uint32 `json:"default_ttl"`

	// LastModified holds the time when the last modification has been made on this Zone.
	LastModified time.Time `json:"last_modified,omitempty"`

	// CommitMsg is a message defined by the User to give a label to this Zone revision.
	CommitMsg *string `json:"commit_message,omitempty"`

	// CommitDate is the time when the commit has been made.
	CommitDate *time.Time `json:"commit_date,omitempty"`

	// Published indicates whether the Zone has already been published or not.
	Published *time.Time `json:"published,omitempty"`
}

// ZoneMessage is the intermediate struct for parsing zones.
type ZoneMessage struct {
	ZoneMeta
	Services map[Subdomain][]*ServiceMessage `json:"services"`
}

// Zone contains ZoneMeta + map of services by subdomains.
type Zone struct {
	ZoneMeta
	Services map[Subdomain][]*Service `json:"services"`
}

// DerivateNew creates a new Zone from the current one, by copying all fields.
func (z *Zone) DerivateNew() *Zone {
	newZone := new(Zone)

	newZone.ZoneMeta.ParentZone = &z.ZoneMeta.Id
	newZone.ZoneMeta.IdAuthor = z.ZoneMeta.IdAuthor
	newZone.ZoneMeta.DefaultTTL = z.ZoneMeta.DefaultTTL
	newZone.ZoneMeta.LastModified = time.Now()
	newZone.Services = map[Subdomain][]*Service{}

	for subdomain, svcs := range z.Services {
		newZone.Services[subdomain] = svcs
	}

	return newZone
}

func (zone *Zone) eraseService(subdomain Subdomain, old *Service, idx int, new *Service) error {
	if new == nil {
		// Disallow removing SOA
		if subdomain == "" && old.Type == "abstract.Origin" {
			return fmt.Errorf("You cannot delete this service. It is mandatory.")
		}

		if len(zone.Services[subdomain]) <= 1 {
			delete(zone.Services, subdomain)
		} else {
			zone.Services[subdomain] = append(zone.Services[subdomain][:idx], zone.Services[subdomain][idx+1:]...)
		}
	} else {
		new.Comment = new.Service.GenComment()
		new.NbResources = new.Service.GetNbResources()
		zone.Services[subdomain][idx] = new
	}

	return nil
}

// EraseService overwrites the Service identified by the given id, under the given subdomain.
// The the new service is nil, it removes the existing Service instead of overwrite it.
func (zone *Zone) EraseService(subdomain Subdomain, id []byte, s *Service) error {
	idx, svc := zone.FindSubdomainService(subdomain, id)
	if svc == nil {
		return fmt.Errorf("service not found")
	}

	return zone.eraseService(subdomain, svc, idx, s)
}

func (zone *Zone) EraseServiceWithoutMeta(subdomain Subdomain, id []byte, s ServiceBody) error {
	idx, svc := zone.FindSubdomainService(subdomain, id)
	if svc == nil {
		return fmt.Errorf("service not found")
	}

	return zone.eraseService(subdomain, svc, idx, &Service{Service: s, ServiceMeta: svc.ServiceMeta})
}

// FindService finds the Service identified by the given id.
func (z *Zone) FindService(id []byte) (Subdomain, *Service) {
	for subdomain := range z.Services {
		if _, svc := z.FindSubdomainService(subdomain, id); svc != nil {
			return subdomain, svc
		}
	}

	return "", nil
}

// FindSubdomainService finds the Service identified by the given id, only under the given subdomain.
func (z *Zone) FindSubdomainService(subdomain Subdomain, id []byte) (int, *Service) {
	if subdomain == "@" {
		subdomain = ""
	}

	if services, ok := z.Services[subdomain]; ok {
		for k, svc := range services {
			if bytes.Equal(svc.Id, id) {
				return k, svc
			}
		}
	}

	return -1, nil
}

type ZoneServices struct {
	Services []*Service `json:"services"`
}

type ZoneUsecase interface {
	CreateZone(*Zone) error
	DeleteRecord(*Zone, string, Record) error
	DeleteZone(Identifier) error
	DiffZones(*Domain, *Zone, Identifier) ([]*Correction, error)
	FlattenZoneFile(*Domain, *Zone) (string, error)
	GenerateRecords(*Domain, *Zone) ([]Record, error)
	GetZone(Identifier) (*Zone, error)
	GetZoneMeta(Identifier) (*ZoneMeta, error)
	LoadZoneFromId(domain *Domain, id Identifier) (*Zone, error)
	UpdateZone(Identifier, func(*Zone)) error
}

type ApplyZoneForm struct {
	WantedCorrections []Identifier `json:"wantedCorrections"`
	CommitMsg         string       `json:"commitMessage"`
}
