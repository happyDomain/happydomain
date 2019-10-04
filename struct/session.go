package happydns

import (
	"crypto/rand"
	"time"
)

type Session struct {
	Id     []byte    `json:"id"`
	IdUser int64     `json:"login"`
	Time   time.Time `json:"time"`
}

func GetSession(id []byte) (s Session, err error) {
	err = DBQueryRow("SELECT id_session, id_user, time FROM user_sessions WHERE id_session=?", id).Scan(&s.Id, &s.IdUser, &s.Time)
	return
}

func (user User) NewSession() (Session, error) {
	session_id := make([]byte, 255)
	if _, err := rand.Read(session_id); err != nil {
		return Session{}, err
	} else if _, err := DBExec("INSERT INTO user_sessions (id_session, id_user, time) VALUES (?, ?, ?)", session_id, user.Id, time.Now()); err != nil {
		return Session{}, err
	} else {
		return Session{session_id, user.Id, time.Now()}, nil
	}
}

func (s Session) Update() (int64, error) {
	if res, err := DBExec("UPDATE user_sessions SET id_user = ?, time = ? WHERE id_session = ?", s.IdUser, s.Time, s.Id); err != nil {
		return 0, err
	} else if nb, err := res.RowsAffected(); err != nil {
		return 0, err
	} else {
		return nb, err
	}
}

func (s Session) Delete() (int64, error) {
	if res, err := DBExec("DELETE FROM user_sessions WHERE id_session = ?", s.Id); err != nil {
		return 0, err
	} else if nb, err := res.RowsAffected(); err != nil {
		return 0, err
	} else {
		return nb, err
	}
}

func ClearSession() (int64, error) {
	if res, err := DBExec("DELETE FROM user_sessions"); err != nil {
		return 0, err
	} else if nb, err := res.RowsAffected(); err != nil {
		return 0, err
	} else {
		return nb, err
	}
}
