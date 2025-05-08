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
	"git.happydns.org/happyDomain/model"
)

func (s *InMemoryStorage) ListAllUsers() (happydns.Iterator[happydns.User], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return NewInMemoryIterator[happydns.User](&s.users), nil
}

// ListUsers retrieves the list of known Users.
func (s *InMemoryStorage) ListUsers() (users []*happydns.User, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, user := range s.users {
		users = append(users, user)
	}

	return users, nil
}

// GetUser retrieves the User with the given identifier.
func (s *InMemoryStorage) GetUser(id happydns.Identifier) (*happydns.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[id.String()]
	if !exists {
		return nil, happydns.ErrUserNotFound
	}

	return user, nil
}

// GetUserByEmail retrieves the User with the given email address.
func (s *InMemoryStorage) GetUserByEmail(email string) (*happydns.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.usersByEmail[email]
	if !exists {
		return nil, happydns.ErrUserNotFound
	}

	return user, nil
}

// CreateOrUpdateUser updates the fields of the given User.
func (s *InMemoryStorage) CreateOrUpdateUser(user *happydns.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.users[user.Id.String()] = user
	s.usersByEmail[user.Email] = user

	return nil
}

// DeleteUser removes the given User from the database.
func (s *InMemoryStorage) DeleteUser(userid happydns.Identifier) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if user, exists := s.users[userid.String()]; exists {
		delete(s.users, userid.String())
		delete(s.usersByEmail, user.Email)
	}

	return nil
}

// ClearUsers deletes all Users present in the database.
func (s *InMemoryStorage) ClearUsers() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.users = make(map[string]*happydns.User)
	s.usersByEmail = make(map[string]*happydns.User)
	return nil
}
