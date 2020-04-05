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
