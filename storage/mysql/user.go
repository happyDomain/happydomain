// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
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
	"git.happydns.org/happydomain/model"
)

func (s *MySQLStorage) GetUsers() (users happydns.Users, err error) {
	if rows, errr := s.db.Query("SELECT id_user, email, password, registration_time FROM users"); errr != nil {
		return nil, errr
	} else {
		defer rows.Close()

		for rows.Next() {
			var u happydns.User
			if err = rows.Scan(&u.Id, &u.Email, &u.Password, &u.RegistrationTime); err != nil {
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

func (s *MySQLStorage) GetUser(id int64) (u *happydns.User, err error) {
	u = &happydns.User{}
	err = s.db.QueryRow("SELECT id_user, email, password, registration_time FROM users WHERE id_user=?", id).Scan(&u.Id, &u.Email, &u.Password, &u.RegistrationTime)
	return
}

func (s *MySQLStorage) GetUserByEmail(email string) (u *happydns.User, err error) {
	u = &happydns.User{}
	err = s.db.QueryRow("SELECT id_user, email, password, registration_time FROM users WHERE email=?", email).Scan(&u.Id, &u.Email, &u.Password, &u.RegistrationTime)
	return
}

func (s *MySQLStorage) UserExists(email string) bool {
	var z int
	err := s.db.QueryRow("SELECT 1 FROM users WHERE email=?", email).Scan(&z)
	return err == nil && z == 1
}

func (s *MySQLStorage) CreateUser(u *happydns.User) error {
	if res, err := s.db.Exec("INSERT INTO users (email, password, registration_time) VALUES (?, ?, ?, ?)", u.Email, u.Password, u.RegistrationTime); err != nil {
		return err
	} else if sid, err := res.LastInsertId(); err != nil {
		return err
	} else {
		u.Id = sid
		return err
	}
}

func (s *MySQLStorage) UpdateUser(u *happydns.User) error {
	_, err := s.db.Exec("UPDATE users SET email = ?, password = ?, registration_time = ? WHERE id_user = ?", u.Email, u.Password, u.RegistrationTime, u.Id)
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
