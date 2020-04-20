package database // import "happydns.org/storage/leveldb"

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	mrand "math/rand"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type LevelDBStorage struct {
	db *leveldb.DB
}

// NewMySQLStorage establishes the connection to the database
func NewLevelDBStorage(path string) (*LevelDBStorage, error) {
	if db, err := leveldb.OpenFile(path, nil); err != nil {
		return nil, err
	} else {
		return &LevelDBStorage{db}, nil
	}
}

func (s *LevelDBStorage) DoMigration() error {
	return nil
}

func (s *LevelDBStorage) Close() error {
	return s.db.Close()
}

func decodeData(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (s *LevelDBStorage) get(key string, v interface{}) error {
	data, err := s.db.Get([]byte(key), nil)
	if err != nil {
		return err
	}

	return decodeData(data, v)
}

func (s *LevelDBStorage) put(key string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return s.db.Put([]byte(key), data, nil)
}

func (s *LevelDBStorage) findInt63Key(prefix string) (key string, id int64, err error) {
	found := true
	for found {
		id = mrand.Int63()
		key = fmt.Sprintf("%s%d", prefix, id)

		found, err = s.db.Has([]byte(key), nil)
		if err != nil {
			return
		}
	}
	return
}

func (s *LevelDBStorage) findBytesKey(prefix string, len int) (key string, id []byte, err error) {
	id = make([]byte, len)
	found := true
	for found {
		if _, err = crand.Read(id); err != nil {
			return
		}
		key = fmt.Sprintf("%s%x", prefix, id)

		found, err = s.db.Has([]byte(key), nil)
		if err != nil {
			return
		}
	}
	return
}

func (s *LevelDBStorage) delete(key string) error {
	return s.db.Delete([]byte(key), nil)
}

func (s *LevelDBStorage) search(prefix string) iterator.Iterator {
	return s.db.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
}
