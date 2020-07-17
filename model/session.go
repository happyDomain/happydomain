// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package happydns

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	mrand "math/rand"
	"time"
)

type Session struct {
	Id      []byte            `json:"id"`
	IdUser  int64             `json:"login"`
	Time    time.Time         `json:"time"`
	Content map[string][]byte `json:"content,omitempty"`
	changed bool
}

func NewSession(user *User) (s *Session, err error) {
	session_id := make([]byte, 255)
	_, err = rand.Read(session_id)
	if err == nil {
		s = &Session{
			Id:     session_id,
			IdUser: user.Id,
			Time:   time.Now(),
		}
	}

	return
}

func (s *Session) HasChanged() bool {
	return s.changed
}

func (s *Session) FindNewKey(prefix string) (key string, id int64) {
	for {
		// max random id is 2^53 to fit on float64 without loosing precision (JSON limitation)
		id = mrand.Int63n(1 << 53)
		key = fmt.Sprintf("%s%d", prefix, id)

		if _, ok := s.Content[key]; !ok {
			return
		}
	}
	return
}

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

func (s *Session) GetValue(key string, value interface{}) bool {
	if v, ok := s.Content[key]; !ok {
		return false
	} else if json.Unmarshal(v, value) != nil {
		return false
	} else {
		return true
	}
}

func (s *Session) DropKey(key string) {
	s.SetValue(key, nil)
}
