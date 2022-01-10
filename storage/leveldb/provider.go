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
// As a counterpart to the access to the provider code and rights to copy, modify
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
	"bytes"
	"fmt"
	"log"
	"reflect"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"

	"git.happydns.org/happydomain/model"
	"git.happydns.org/happydomain/providers"
)

func (s *LevelDBStorage) getProviderMeta(id []byte) (srcMeta *happydns.ProviderMeta, err error) {
	var v []byte
	v, err = s.db.Get(id, nil)
	if err != nil {
		return
	}

	srcMeta = &happydns.ProviderMeta{}
	err = decodeData(v, srcMeta)
	return
}

func (s *LevelDBStorage) GetProviderMetas(u *happydns.User) (srcs []happydns.ProviderMeta, err error) {
	iter := s.search("provider-")
	defer iter.Release()

	for iter.Next() {
		var srcMeta happydns.ProviderMeta
		err = decodeData(iter.Value(), &srcMeta)
		if err != nil {
			return
		}

		if !bytes.Equal(srcMeta.OwnerId, u.Id) {
			continue
		}

		srcs = append(srcs, srcMeta)
	}

	return
}

func (s *LevelDBStorage) GetProviderMeta(u *happydns.User, id int64) (srcMeta *happydns.ProviderMeta, err error) {
	var v []byte
	v, err = s.db.Get([]byte(fmt.Sprintf("provider-%d", id)), nil)
	if err != nil {
		return
	}

	srcMeta = new(happydns.ProviderMeta)
	err = decodeData(v, &srcMeta)
	if err != nil {
		return
	}

	if !bytes.Equal(srcMeta.OwnerId, u.Id) {
		srcMeta = nil
		err = leveldb.ErrNotFound
	}

	return
}

func (s *LevelDBStorage) GetProvider(u *happydns.User, id int64) (src *happydns.ProviderCombined, err error) {
	var v []byte
	v, err = s.db.Get([]byte(fmt.Sprintf("provider-%d", id)), nil)
	if err != nil {
		return
	}

	var srcMeta happydns.ProviderMeta
	err = decodeData(v, &srcMeta)
	if err != nil {
		return
	}

	if !bytes.Equal(srcMeta.OwnerId, u.Id) {
		src = nil
		err = leveldb.ErrNotFound
	}

	var tsrc happydns.Provider
	tsrc, err = providers.FindProvider(srcMeta.Type)

	src = &happydns.ProviderCombined{
		tsrc,
		srcMeta,
	}

	err = decodeData(v, src)

	return
}

func (s *LevelDBStorage) CreateProvider(u *happydns.User, src happydns.Provider, comment string) (*happydns.ProviderCombined, error) {
	key, id, err := s.findInt63Key("provider-")
	if err != nil {
		return nil, err
	}

	sType := reflect.Indirect(reflect.ValueOf(src)).Type()

	st := &happydns.ProviderCombined{
		src,
		happydns.ProviderMeta{
			Type:    sType.Name(),
			Id:      id,
			OwnerId: u.Id,
			Comment: comment,
		},
	}
	return st, s.put(key, st)
}

func (s *LevelDBStorage) UpdateProvider(src *happydns.ProviderCombined) error {
	return s.put(fmt.Sprintf("provider-%d", src.Id), src)
}

func (s *LevelDBStorage) UpdateProviderOwner(src *happydns.ProviderCombined, newOwner *happydns.User) error {
	src.OwnerId = newOwner.Id
	return s.UpdateProvider(src)
}

func (s *LevelDBStorage) DeleteProvider(src *happydns.ProviderMeta) error {
	return s.delete(fmt.Sprintf("provider-%d", src.Id))
}

func (s *LevelDBStorage) ClearProviders() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("provider-")), nil)
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

func (s *LevelDBStorage) TidyProviders() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("provider-")), nil)
	defer iter.Release()

	for iter.Next() {
		srcMeta, err := s.getProviderMeta(iter.Key())

		if err != nil {
			// Drop unreadable providers
			log.Printf("Deleting unreadable provider (%w): %v\n", err, srcMeta)
			err = tx.Delete(iter.Key(), nil)
		} else {
			_, err = s.GetUser(srcMeta.OwnerId)
			if err == leveldb.ErrNotFound {
				// Drop providers of unexistant users
				log.Printf("Deleting orphan provider (user %d not found): %v\n", srcMeta.OwnerId, srcMeta)
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
