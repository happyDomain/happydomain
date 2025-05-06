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

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

func (s *InMemoryStorage) ListAllProviders() (storage.Iterator[happydns.ProviderMessage], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return NewInMemoryIterator[happydns.ProviderMessage](&s.providers), nil
}

// ListProviders retrieves all providers owned by the given User.
func (s *InMemoryStorage) ListProviders(u *happydns.User) (happydns.ProviderMessages, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var providers happydns.ProviderMessages
	for _, provider := range s.providers {
		if provider.Owner.Equals(u.Id) {
			providers = append(providers, provider)
		}
	}

	return providers, nil
}

// GetProvider retrieves the full Provider with the given identifier and owner.
func (s *InMemoryStorage) GetProvider(id happydns.Identifier) (*happydns.ProviderMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	provider, exists := s.providers[id.String()]
	if !exists {
		return nil, happydns.ErrProviderNotFound
	}

	return provider, nil
}

// CreateProvider creates a record in the database for the given Provider.
func (s *InMemoryStorage) CreateProvider(p *happydns.Provider) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	message, err := json.Marshal(p.Provider)
	if err != nil {
		return err
	}

	p.ProviderMeta.Id, err = happydns.NewRandomIdentifier()
	s.providers[p.ProviderMeta.Id.String()] = &happydns.ProviderMessage{
		ProviderMeta: p.ProviderMeta,
		Provider:     message,
	}

	return nil
}

// UpdateProvider updates the fields of the given Provider.
func (s *InMemoryStorage) UpdateProvider(p *happydns.Provider) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	message, err := json.Marshal(p.Provider)
	if err != nil {
		return err
	}

	s.providers[p.ProviderMeta.Id.String()] = &happydns.ProviderMessage{
		ProviderMeta: p.ProviderMeta,
		Provider:     message,
	}

	return nil
}

// DeleteProvider removes the given Provider from the database.
func (s *InMemoryStorage) DeleteProvider(id happydns.Identifier) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.providers, id.String())
	return nil
}

// ClearProviders deletes all Providers present in the database.
func (s *InMemoryStorage) ClearProviders() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.providers = make(map[string]*happydns.ProviderMessage)
	return nil
}
