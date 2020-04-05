package database

import (
	"git.happydns.org/happydns/model"
)

func (s *MySQLStorage) GetUsers() (users happydns.Users, err error) {
	if rows, errr := s.db.Query("SELECT id_user, email, password, salt, registration_time FROM users"); errr != nil {
		return nil, errr
	} else {
		defer rows.Close()

		for rows.Next() {
			var u happydns.User
			if err = rows.Scan(&u.Id, &u.Email, &u.Password, &u.Salt, &u.RegistrationTime); err != nil {
				return
			}
			users = append(users, &u)
		}
		if err = rows.Err(); err != nil {
			return
		}

		return
	}
}

func (s *MySQLStorage) GetUser(id int) (u *happydns.User, err error) {
	u = &happydns.User{}
	err = s.db.QueryRow("SELECT id_user, email, password, salt, registration_time FROM users WHERE id_user=?", id).Scan(&u.Id, &u.Email, &u.Password, &u.Salt, &u.RegistrationTime)
	return
}

func (s *MySQLStorage) GetUserByEmail(email string) (u *happydns.User, err error) {
	u = &happydns.User{}
	err = s.db.QueryRow("SELECT id_user, email, password, salt, registration_time FROM users WHERE email=?", email).Scan(&u.Id, &u.Email, &u.Password, &u.Salt, &u.RegistrationTime)
	return
}

func (s *MySQLStorage) UserExists(email string) bool {
	var z int
	err := s.db.QueryRow("SELECT 1 FROM users WHERE email=?", email).Scan(&z)
	return err == nil && z == 1
}

func (s *MySQLStorage) CreateUser(u *happydns.User) error {
	if res, err := s.db.Exec("INSERT INTO users (email, password, salt, registration_time) VALUES (?, ?, ?, ?)", u.Email, u.Password, u.Salt, u.RegistrationTime); err != nil {
		return err
	} else if sid, err := res.LastInsertId(); err != nil {
		return err
	} else {
		u.Id = sid
		return err
	}
}

func (s *MySQLStorage) UpdateUser(u *happydns.User) error {
	_, err := s.db.Exec("UPDATE users SET email = ?, password = ?, salt = ?, registration_time = ? WHERE id_user = ?", u.Email, u.Password, u.Salt, u.RegistrationTime, u.Id)
	return err
}

func (s *MySQLStorage) DeleteUser(u *happydns.User) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id_user = ?", u.Id)
	return err
}

func (s *MySQLStorage) ClearUsers() error {
	_, err := s.db.Exec("DELETE FROM users")
	return err
}
