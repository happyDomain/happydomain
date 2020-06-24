// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
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
	"fmt"
	"log"
	"reflect"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/sources"
)

func (s *LevelDBStorage) getSourceMeta(id []byte) (srcMeta *happydns.SourceMeta, err error) {
	srcMeta = &happydns.SourceMeta{}
	err = decodeData(id, srcMeta)
	return
}

func (s *LevelDBStorage) GetSourceMetas(u *happydns.User) (srcs []happydns.SourceMeta, err error) {
	iter := s.search("source-")
	defer iter.Release()

	for iter.Next() {
		var srcMeta happydns.SourceMeta
		err = decodeData(iter.Value(), &srcMeta)
		if err != nil {
			return
		}

		if srcMeta.OwnerId != u.Id {
			continue
		}

		srcs = append(srcs, srcMeta)
	}

	return
}

func (s *LevelDBStorage) GetSourceMeta(u *happydns.User, id int64) (srcMeta *happydns.SourceMeta, err error) {
	var v []byte
	v, err = s.db.Get([]byte(fmt.Sprintf("source-%d", id)), nil)
	if err != nil {
		return
	}

	srcMeta = new(happydns.SourceMeta)
	err = decodeData(v, &srcMeta)
	if err != nil {
		return
	}

	if srcMeta.OwnerId != u.Id {
		srcMeta = nil
		err = leveldb.ErrNotFound
	}

	return
}

func (s *LevelDBStorage) GetSource(u *happydns.User, id int64) (src *happydns.SourceCombined, err error) {
	var v []byte
	v, err = s.db.Get([]byte(fmt.Sprintf("source-%d", id)), nil)
	if err != nil {
		return
	}

	var srcMeta happydns.SourceMeta
	err = decodeData(v, &srcMeta)
	if err != nil {
		return
	}

	if srcMeta.OwnerId != u.Id {
		src = nil
		err = leveldb.ErrNotFound
	}

	var tsrc happydns.Source
	tsrc, err = sources.FindSource(srcMeta.Type)

	src = &happydns.SourceCombined{
		tsrc,
		srcMeta,
	}

	err = decodeData(v, src)

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
		happydns.SourceMeta{
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

func (s *LevelDBStorage) DeleteSource(src *happydns.SourceMeta) error {
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

func (s *LevelDBStorage) TidySources() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("source-")), nil)
	defer iter.Release()

	for iter.Next() {
		srcMeta, err := s.getSourceMeta(iter.Key())

		if err != nil {
			// Drop unreadable sources
			log.Printf("Deleting unreadable source (%w): %v\n", err, srcMeta)
			err = tx.Delete(iter.Key(), nil)
		} else {
			_, err = s.GetUser(srcMeta.OwnerId)
			if err == leveldb.ErrNotFound {
				// Drop sources of unexistant users
				log.Printf("Deleting orphan source (user %d not found): %v\n", srcMeta.OwnerId, srcMeta)
				err = tx.Delete(iter.Key(), nil)
			}
		}

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
