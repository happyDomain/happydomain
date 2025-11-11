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

// ZoneMessage is the intermediate struct for parsing zones.
type ZoneMessage struct {
	ZoneMeta
	Services map[string][]*ServiceMessage `json:"services"`
}

func (z *Zone) Meta() (meta ZoneMeta) {
	meta.Id = z.Id
	meta.IdAuthor = z.IdAuthor
	meta.Parent = z.Parent
	meta.CommitDate = z.CommitDate
	meta.CommitMessage = z.CommitMessage
	meta.DefaultTtl = z.DefaultTtl
	meta.LastModified = z.LastModified
	meta.Published = z.Published

	return
}

func (z *Zone) SetMeta(meta ZoneMeta) {
	z.Id = meta.Id
	z.IdAuthor = meta.IdAuthor
	z.Parent = meta.Parent
	z.CommitDate = meta.CommitDate
	z.CommitMessage = meta.CommitMessage
	z.DefaultTtl = meta.DefaultTtl
	z.LastModified = meta.LastModified
	z.Published = meta.Published
}

// DerivateNew creates a new Zone from the current one, by copying all fields.
func (z *Zone) DerivateNew() *Zone {
	newZone := new(Zone)

	newZone.Parent = &z.Id
	newZone.IdAuthor = z.IdAuthor
	newZone.DefaultTtl = z.DefaultTtl
	newZone.LastModified = time.Now()
	newZone.Services = map[string][]*Service{}

	for subdomain, svcs := range z.Services {
		newZone.Services[subdomain] = svcs
	}

	return newZone
}

func (zone *Zone) eraseService(subdomain string, old *Service, idx int, new *Service) error {
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
func (zone *Zone) EraseService(subdomain string, id []byte, s *Service) error {
	idx, svc := zone.FindSubdomainService(subdomain, id)
	if svc == nil {
		return fmt.Errorf("service not found")
	}

	return zone.eraseService(subdomain, svc, idx, s)
}

func (zone *Zone) EraseServiceWithoutMeta(subdomain string, id []byte, s ServiceBody) error {
	idx, svc := zone.FindSubdomainService(subdomain, id)
	if svc == nil {
		return fmt.Errorf("service not found")
	}

	return zone.eraseService(subdomain, svc, idx, &Service{
		UnderscoreId:      svc.UnderscoreId,
		Aliases:           svc.Aliases,
		Type:              svc.Type,
		Comment:           svc.Comment,
		Domain:            svc.Domain,
		UserComment:       svc.UserComment,
		UnderscoreOwnerid: svc.UnderscoreOwnerid,
		Ttl:               svc.Ttl,
		NbResources:       svc.NbResources,
		Service:           s,
	})
}

// FindService finds the Service identified by the given id.
func (z *Zone) FindService(id []byte) (string, *Service) {
	for subdomain := range z.Services {
		if _, svc := z.FindSubdomainService(subdomain, id); svc != nil {
			return subdomain, svc
		}
	}

	return "", nil
}

// FindSubdomainService finds the Service identified by the given id, only under the given subdomain.
func (z *Zone) FindSubdomainService(subdomain string, id []byte) (int, *Service) {
	if subdomain == "@" {
		subdomain = ""
	}

	if services, ok := z.Services[subdomain]; ok {
		for k, svc := range services {
			if bytes.Equal(svc.UnderscoreId, id) {
				return k, svc
			}
		}
	}

	return -1, nil
}

type ZoneUsecase interface {
	AddRecord(*Zone, string, Record) error
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
