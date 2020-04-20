package database

import (
	"fmt"

	"github.com/syndtr/goleveldb/leveldb/util"

	"git.happydns.org/happydns/model"
)

func (s *LevelDBStorage) GetSession(id []byte) (session *happydns.Session, err error) {
	session = &happydns.Session{}
	err = s.get(fmt.Sprintf("user.session-%x", id), &session)
	return
}

func (s *LevelDBStorage) CreateSession(session *happydns.Session) error {
	key, id, err := s.findBytesKey("user.session-", 255)
	if err != nil {
		return err
	}

	session.Id = id
	return s.put(key, session)
}

func (s *LevelDBStorage) UpdateSession(session *happydns.Session) error {
	return s.put(fmt.Sprintf("user.session-%x", session.Id), session)
}

func (s *LevelDBStorage) DeleteSession(session *happydns.Session) error {
	return s.delete(fmt.Sprintf("user.session-%x", session.Id))
}

func (s *LevelDBStorage) ClearSessions() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("user.session-")), nil)
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
