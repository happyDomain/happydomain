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

package database

import (
	"crypto/rand"

	"git.happydns.org/happydns/model"
)

func (s *MySQLStorage) GetSession(id []byte) (session *happydns.Session, err error) {
	session = &happydns.Session{}
	err = s.db.QueryRow("SELECT id_session, id_user, time FROM user_sessions WHERE id_session=?", id).Scan(&session.Id, &session.IdUser, &session.Time)
	return
}

func (s *MySQLStorage) CreateSession(session *happydns.Session) error {
	session_id := make([]byte, 255)
	if _, err := rand.Read(session_id); err != nil {
		return err
	} else if _, err := s.db.Exec("INSERT INTO user_sessions (id_session, id_user, time) VALUES (?, ?, ?)", session_id, session.IdUser, session.Time); err != nil {
		return err
	} else {
		session.Id = session_id
		return nil
	}
}

func (s *MySQLStorage) UpdateSession(session *happydns.Session) error {
	_, err := s.db.Exec(`UPDATE user_sessions SET id_user = ?, time = ? WHERE id_session = ?`, session.IdUser, session.Time, session.Id)
	return err
}

func (s *MySQLStorage) DeleteSession(session *happydns.Session) error {
	_, err := s.db.Exec("DELETE FROM user_sessions WHERE id_session = ?", session.Id)
	return err
}

func (s *MySQLStorage) ClearSessions() error {
	_, err := s.db.Exec("DELETE FROM user_sessions")
	return err
}
