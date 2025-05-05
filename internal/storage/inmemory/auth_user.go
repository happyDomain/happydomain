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
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

func (s *InMemoryStorage) ListAllAuthUsers() (storage.Iterator[happydns.UserAuth], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return NewInMemoryIterator[happydns.UserAuth](&s.authUsers), nil
}

// ListAuthUsers retrieves the list of known Users.
func (s *InMemoryStorage) ListAuthUsers() (happydns.UserAuths, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var users happydns.UserAuths
	for _, user := range s.authUsers {
		users = append(users, user)
	}

	return users, nil
}

// GetAuthUser retrieves the User with the given identifier.
func (s *InMemoryStorage) GetAuthUser(id happydns.Identifier) (*happydns.UserAuth, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.authUsers[id.String()]
	if !exists {
		return nil, storage.ErrNotFound
	}

	return user, nil
}

// GetAuthUserByEmail retrieves the User with the given email address.
func (s *InMemoryStorage) GetAuthUserByEmail(email string) (*happydns.UserAuth, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userid, exists := s.authUsersByEmail[email]
	if !exists {
		return nil, storage.ErrNotFound
	}

	user, exists := s.authUsers[userid.String()]
	if !exists {
		return nil, storage.ErrNotFound
	}

	return user, nil
}

// AuthUserExists checks if the given email address is already associated to an User.
func (s *InMemoryStorage) AuthUserExists(email string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.authUsersByEmail[email]
	return exists, nil
}

// CreateAuthUser creates a record in the database for the given User.
func (s *InMemoryStorage) CreateAuthUser(user *happydns.UserAuth) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user.Id, err = happydns.NewRandomIdentifier()
	s.authUsers[user.Id.String()] = user
	s.authUsersByEmail[user.Email] = user.Id
	return
}

// UpdateAuthUser updates the fields of the given User.
func (s *InMemoryStorage) UpdateAuthUser(user *happydns.UserAuth) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.authUsers[user.Id.String()] = user
	s.authUsersByEmail[user.Email] = user.Id
	return nil
}

// DeleteAuthUser removes the given User from the database.
func (s *InMemoryStorage) DeleteAuthUser(user *happydns.UserAuth) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.authUsers, user.Id.String())
	delete(s.authUsersByEmail, user.Email)

	return nil
}

// ClearAuthUsers deletes all AuthUsers present in the database.
func (s *InMemoryStorage) ClearAuthUsers() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.authUsers = make(map[string]*happydns.UserAuth)
	s.authUsersByEmail = make(map[string]happydns.Identifier)

	return nil
}
