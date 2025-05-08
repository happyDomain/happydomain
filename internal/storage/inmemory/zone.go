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

package inmemory

import (
	"encoding/json"

	"git.happydns.org/happyDomain/model"
)

func (s *InMemoryStorage) ListAllZones() (happydns.Iterator[happydns.ZoneMessage], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return NewInMemoryIterator[happydns.ZoneMessage](&s.zones), nil
}

// GetZoneMeta retrieves metadata of the Zone with the given identifier.
func (s *InMemoryStorage) GetZoneMeta(id happydns.Identifier) (*happydns.ZoneMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	zone, exists := s.zones[id.String()]
	if !exists {
		return nil, happydns.ErrZoneNotFound
	}
	return &zone.ZoneMeta, nil
}

// GetZone retrieves the full Zone (including Services and metadata) which have the given identifier.
func (s *InMemoryStorage) GetZone(id happydns.Identifier) (*happydns.ZoneMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	zone, exists := s.zones[id.String()]
	if !exists {
		return nil, happydns.ErrZoneNotFound
	}

	return zone, nil
}

// CreateZone creates a record in the database for the given Zone.
func (s *InMemoryStorage) CreateZone(zone *happydns.Zone) (err error) {
	zone.ZoneMeta.Id, err = happydns.NewRandomIdentifier()
	if err != nil {
		return
	}

	return s.UpdateZone(zone)
}

// UpdateZone updates the fields of the given Zone.
func (s *InMemoryStorage) UpdateZone(zone *happydns.Zone) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	zmsg := &happydns.ZoneMessage{
		ZoneMeta: zone.ZoneMeta,
		Services: map[happydns.Subdomain][]*happydns.ServiceMessage{},
	}

	for subdn, services := range zone.Services {
		for _, service := range services {
			message, err := json.Marshal(service.Service)
			if err != nil {
				return err
			}

			zmsg.Services[subdn] = append(zmsg.Services[subdn], &happydns.ServiceMessage{
				ServiceMeta: service.ServiceMeta,
				Service:     message,
			})
		}
	}

	s.zones[zone.Id.String()] = zmsg

	return nil
}

// DeleteZone removes the given Zone from the database.
func (s *InMemoryStorage) DeleteZone(id happydns.Identifier) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.zones, id.String())
	return nil
}

// ClearZones deletes all Zones present in the database.
func (s *InMemoryStorage) ClearZones() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.zones = make(map[string]*happydns.ZoneMessage)
	return nil
}
