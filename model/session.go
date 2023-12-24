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
	"crypto/rand"
	"encoding/json"
	"fmt"
	mrand "math/rand"
	"time"
)

// Session holds informatin about a User's currently connected.
type Session struct {
	// Id is the Session's identifier.
	Id Identifier `json:"id" swaggertype:"string"`

	// IdUser is the User's identifier of the Session.
	IdUser Identifier `json:"login" swaggertype:"string"`

	// IssuedAt holds the creation date of the Session.
	IssuedAt time.Time `json:"time"`

	// Content stores data filled by other modules.
	Content map[string][]byte `json:"content,omitempty"`

	// changed indicates if Content has changed since its loading.
	changed bool
}

// NewSession fills a new Session structure.
func NewSession(user *User) (s *Session, err error) {
	session_id := make([]byte, 16)
	_, err = rand.Read(session_id)
	if err == nil {
		s = &Session{
			Id:       session_id,
			IdUser:   user.Id,
			IssuedAt: time.Now(),
		}
	}

	return
}

// HasChanged tells if the Session has changed since its last loading.
func (s *Session) HasChanged() bool {
	return s.changed
}

// FindNewKey returns a key and an identifier appended to the given
// prefix, that is available in the User's Session.
func (s *Session) FindNewKey(prefix string) (key string, id int64) {
	for {
		// max random id is 2^53 to fit on float64 without loosing precision (JSON limitation)
		id = mrand.Int63n(1 << 53)
		key = fmt.Sprintf("%s%d", prefix, id)

		if _, ok := s.Content[key]; !ok {
			return
		}
	}
}

// SetValue defines, erase or delete a content to stores at the given
// key. If the key is already defined, it erases its content. If the
// given value is nil, it deletes the key.
func (s *Session) SetValue(key string, value interface{}) {
	if s.Content == nil && value != nil {
		s.Content = map[string][]byte{}
	}

	if value == nil {
		if s.Content == nil {
			return
		} else if _, ok := s.Content[key]; !ok {
			return
		} else {
			delete(s.Content, key)
			s.changed = true
		}
	} else {
		s.Content[key], _ = json.Marshal(value)
		s.changed = true
	}
}

// GetValue retrieves data stored at the given key. Returns true if
// the key exists and if the value has been filled correctly.
func (s *Session) GetValue(key string, value interface{}) bool {
	if v, ok := s.Content[key]; !ok {
		return false
	} else if json.Unmarshal(v, value) != nil {
		return false
	} else {
		return true
	}
}

// DropKey removes the given key from the Session's Content.
func (s *Session) DropKey(key string) {
	s.SetValue(key, nil)
}

// ClearSession removes all content from the Session.
func (s *Session) ClearSession() {
	s.Content = nil
	s.changed = true
}
