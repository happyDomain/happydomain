package database

import (
	"fmt"
	"reflect"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/sources"
)

func (s *LevelDBStorage) GetSourceTypes(u *happydns.User) (srcs []happydns.SourceType, err error) {
	iter := s.search("source-")
	defer iter.Release()

	for iter.Next() {
		var srcType happydns.SourceType
		err = decodeData(iter.Value(), &srcType)
		if err != nil {
			return
		}

		if srcType.OwnerId != u.Id {
			continue
		}

		srcs = append(srcs, srcType)
	}

	return
}

func (s *LevelDBStorage) GetSource(u *happydns.User, id int64) (src *happydns.SourceCombined, err error) {
	var v []byte
	v, err = s.db.Get([]byte(fmt.Sprintf("source-%d", id)), nil)
	if err != nil {
		return
	}

	var srcType happydns.SourceType
	err = decodeData(v, &srcType)
	if err != nil {
		return
	}

	if srcType.OwnerId != u.Id {
		src = nil
		err = leveldb.ErrNotFound
	}

	var tsrc happydns.Source
	tsrc, err = sources.FindSource(srcType.Type)

	src = &happydns.SourceCombined{
		tsrc,
		srcType,
	}

	err = decodeData(v, src)
	if err != nil {
		return
	}

	return
}

func (s *LevelDBStorage) CreateSource(u *happydns.User, src happydns.Source, comment string) (*happydns.SourceCombined, error) {
	key, id, err := s.findInt63Key("source-")
	if err != nil {
		return nil, err
	}

	sType := reflect.Indirect(reflect.ValueOf(src)).Type()

	st := &happydns.SourceCombined{
		src,
		happydns.SourceType{
			Type:    sType.PkgPath() + "/" + sType.Name(),
			Id:      id,
			OwnerId: u.Id,
			Comment: comment,
		},
	}
	return st, s.put(key, st)
}

func (s *LevelDBStorage) UpdateSource(src *happydns.SourceCombined) error {
	return s.put(fmt.Sprintf("source-%d", src.Id), src)
}

func (s *LevelDBStorage) UpdateSourceOwner(src *happydns.SourceCombined, newOwner *happydns.User) error {
	src.OwnerId = newOwner.Id
	return s.UpdateSource(src)
}

func (s *LevelDBStorage) DeleteSource(src *happydns.SourceType) error {
	return s.delete(fmt.Sprintf("source-%d", src.Id))
}

func (s *LevelDBStorage) ClearSources() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("source-")), nil)
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
