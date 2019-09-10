package libredns

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"time"
)

type User struct {
	Id               int64  `json:"id"`
	Email            string `json:"email"`
	password         []byte
	salt             []byte
	RegistrationTime *time.Time `json:"registration_time"`
}

func GetUsers() (users []User, err error) {
	if rows, errr := DBQuery("SELECT id_user, email, password, salt, registration_time FROM users"); errr != nil {
		return nil, errr
	} else {
		defer rows.Close()

		for rows.Next() {
			var u User
			if err = rows.Scan(&u.Id, &u.Email, &u.password, &u.salt, &u.RegistrationTime); err != nil {
				return
			}
			users = append(users, u)
		}
		if err = rows.Err(); err != nil {
			return
		}

		return
	}
}

func GetUser(id int) (u User, err error) {
	err = DBQueryRow("SELECT id_user, email, password, salt, registration_time WHERE id_user=?", id).Scan(&u.Id, &u.Email, &u.password, &u.salt, &u.RegistrationTime)
	return
}

func GetUserByEmail(email string) (u User, err error) {
	err = DBQueryRow("SELECT id_user, email, password, salt, registration_time WHERE email=?", email).Scan(&u.Id, &u.Email, &u.password, &u.salt, &u.RegistrationTime)
	return
}

func UserExists(email string) bool {
	var z int
	err := DBQueryRow("SELECT 1 FROM users WHERE email=?", email).Scan(&z)
	return err == nil && z == 1
}

func GenPassword(password string, salt []byte) []byte {
	return hmac.New(sha512.New512_224, []byte(password)).Sum([]byte(salt))
}

func NewUser(email string, password string) (User, error) {
	salt := make([]byte, 64)
	if _, err := rand.Read(salt); err != nil {
		return User{}, err
	}
	t := time.Now()
	hashedpass := GenPassword(password, salt)
	if res, err := DBExec("INSERT INTO users (email, password, salt, registration_time) VALUES (?, ?, ?, ?)", email, hashedpass, salt, t); err != nil {
		return User{}, err
	} else if sid, err := res.LastInsertId(); err != nil {
		return User{}, err
	} else {
		return User{sid, email, hashedpass, salt, &t}, nil
	}
}

func (u User) Update() (int64, error) {
	if res, err := DBExec("UPDATE users SET email = ?, password = ?, salt = ?, registration_time = ? WHERE id_user = ?", u.Email, u.password, u.salt, u.RegistrationTime, u.Id); err != nil {
		return 0, err
	} else if nb, err := res.RowsAffected(); err != nil {
		return 0, err
	} else {
		return nb, err
	}
}

func (u User) Delete() (int64, error) {
	if res, err := DBExec("DELETE FROM users WHERE id_user = ?", u.Id); err != nil {
		return 0, err
	} else if nb, err := res.RowsAffected(); err != nil {
		return 0, err
	} else {
		return nb, err
	}
}

func ClearUsers() (int64, error) {
	if res, err := DBExec("DELETE FROM users"); err != nil {
		return 0, err
	} else if nb, err := res.RowsAffected(); err != nil {
		return 0, err
	} else {
		return nb, err
	}
}
