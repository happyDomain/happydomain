package database

import (
	"fmt"

	"git.happydns.org/happydns/model"

	"github.com/syndtr/goleveldb/leveldb/util"
)

func (s *LevelDBStorage) GetUsers() (users happydns.Users, err error) {
	iter := s.search("user-")
	defer iter.Release()

	for iter.Next() {
		var u happydns.User

		err = decodeData(iter.Value(), &u)
		if err != nil {
			return
		}
		users = append(users, &u)
	}

	return
}

func (s *LevelDBStorage) GetUser(id int) (u *happydns.User, err error) {
	u = &happydns.User{}
	err = s.get(fmt.Sprintf("user-%d", id), &u)
	return
}

func (s *LevelDBStorage) GetUserByEmail(email string) (u *happydns.User, err error) {
	var users happydns.Users

	users, err = s.GetUsers()
	if err != nil {
		return
	}

	for _, user := range users {
		if user.Email == email {
			u = user
			return
		}
	}

	return nil, fmt.Errorf("Unable to find user with email address '%s'.", email)
}

func (s *LevelDBStorage) UserExists(email string) bool {
	users, err := s.GetUsers()
	if err != nil {
		return false
	}

	for _, user := range users {
		if user.Email == email {
			return true
		}
	}

	return false
}

func (s *LevelDBStorage) CreateUser(u *happydns.User) error {
	key, id, err := s.findInt63Key("user-")
	if err != nil {
		return err
	}

	u.Id = id
	return s.put(key, u)
}

func (s *LevelDBStorage) UpdateUser(u *happydns.User) error {
	return s.put(fmt.Sprintf("user-%d", u.Id), u)
}

func (s *LevelDBStorage) DeleteUser(u *happydns.User) error {
	return s.delete(fmt.Sprintf("user-%d", u.Id))
}

func (s *LevelDBStorage) ClearUsers() error {
	if err := s.ClearSessions(); err != nil {
		return err
	}

	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("user-")), nil)
	defer iter.Release()

	for iter.Next() {
		err = tx.Delete(iter.Key(), nil)
		if err != nil {
			tx.Discard()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		tx.Discard()
		return err
	}

	return nil
}
