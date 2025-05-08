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

func (s *InMemoryStorage) ListAllSessions() (happydns.Iterator[happydns.Session], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return NewInMemoryIterator[happydns.Session](&s.sessions), nil
}

// GetSession retrieves the Session with the given identifier.
func (s *InMemoryStorage) GetSession(id string) (*happydns.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[id]
	if !exists {
		return nil, happydns.ErrSessionNotFound
	}

	return session, nil
}

// ListAuthUserSessions retrieves all Session for the given AuthUser.
func (s *InMemoryStorage) ListAuthUserSessions(user *happydns.UserAuth) (sessions []*happydns.Session, ess error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, session := range s.sessions {
		if user.Id.Equals(session.IdUser) {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// ListUserSessions retrieves all Session for the given User.
func (s *InMemoryStorage) ListUserSessions(userid happydns.Identifier) (sessions []*happydns.Session, ess error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, session := range s.sessions {
		if userid.Equals(session.IdUser) {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// UpdateSession updates the fields of the given Session.
func (s *InMemoryStorage) UpdateSession(session *happydns.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.Id] = session
	return nil
}

// DeleteSession removes the given Session from the database.
func (s *InMemoryStorage) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)
	return nil
}

// ClearSessions deletes all Sessions present in the database.
func (s *InMemoryStorage) ClearSessions() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions = make(map[string]*happydns.Session)
	return nil
}
