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
