package happydns

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"time"
)

type User struct {
	Id               int64
	Email            string
	Password         []byte
	Salt             []byte
	RegistrationTime *time.Time
}

type Users []*User

func GenPassword(password string, salt []byte) []byte {
	return hmac.New(sha512.New512_224, []byte(password)).Sum(salt)
}

func NewUser(email string, password string) (u *User, err error) {
	t := time.Now()

	u = &User{
		Id:               0,
		Email:            email,
		RegistrationTime: &t,
	}

	err = u.DefinePassword(password)

	return
}

// DefinePassword computes the expected hash for the given password and also
// renew the User's Salt.
func (u *User) DefinePassword(password string) error {
	// Renew salt
	u.Salt = make([]byte, 64)
	if _, err := rand.Read(u.Salt); err != nil {
		return err
	}

	// Compute password hash
	u.Password = GenPassword(password, u.Salt)

	return nil
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
