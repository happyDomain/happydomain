package happydns

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"time"
)

type User struct {
	Id               int64  `json:"id"`
	Email            string `json:"email"`
	Password         []byte
	Salt             []byte
	RegistrationTime *time.Time `json:"registration_time"`
}

type Users []*User

func GenPassword(password string, salt []byte) []byte {
	return hmac.New(sha512.New512_224, []byte(password)).Sum([]byte(salt))
}

func NewUser(email string, password string) (*User, error) {
	salt := make([]byte, 64)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	hashedpass := GenPassword(password, salt)
	t := time.Now()
	return &User{
		Id:               0,
		Email:            email,
		Password:         hashedpass,
		Salt:             salt,
		RegistrationTime: &t,
	}, nil
}

func (u *User) CheckAuth(password string) bool {
	pass := GenPassword(password, u.Salt)
	if len(pass) != len(u.Password) {
		return false
	} else {
		for k := range pass {
			if pass[k] != u.Password[k] {
				return false
			}
		}
		return true
	}
}
