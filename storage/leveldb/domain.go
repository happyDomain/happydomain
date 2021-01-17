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

	"git.happydns.org/happydns/model"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func (s *LevelDBStorage) GetDomains(u *happydns.User) (domains happydns.Domains, err error) {
	iter := s.search("domain-")
	defer iter.Release()

	for iter.Next() {
		var z happydns.Domain

		err = decodeData(iter.Value(), &z)
		if err != nil {
			return
		}

		if z.IdUser == u.Id {
			domains = append(domains, &z)
		}
	}

	return
}

func (s *LevelDBStorage) getDomain(id string) (z *happydns.Domain, err error) {
	z = &happydns.Domain{}
	err = s.get(id, z)
	return
}

func (s *LevelDBStorage) GetDomain(u *happydns.User, id int64) (z *happydns.Domain, err error) {
	z, err = s.getDomain(fmt.Sprintf("domain-%d", id))

	if err != nil {
		return
	}

	if z.IdUser != u.Id {
		z = nil
		err = leveldb.ErrNotFound
	}

	return
}

func (s *LevelDBStorage) GetDomainByDN(u *happydns.User, dn string) (*happydns.Domain, error) {
	domains, err := s.GetDomains(u)
	if err != nil {
		return nil, err
	}

	for _, domain := range domains {
		if domain.DomainName == dn {
			return domain, nil
		}
	}

	return nil, leveldb.ErrNotFound
}

func (s *LevelDBStorage) DomainExists(dn string) bool {
	iter := s.search("domain-")
	defer iter.Release()

	for iter.Next() {
		var z happydns.Domain

		err := decodeData(iter.Value(), &z)
		if err != nil {
			continue
		}

		if z.DomainName == dn {
			return true
		}
	}

	return false
}

func (s *LevelDBStorage) CreateDomain(u *happydns.User, z *happydns.Domain) error {
	key, id, err := s.findInt63Key("domain-")
	if err != nil {
		return err
	}

	z.Id = id
	z.IdUser = u.Id
	return s.put(key, z)
}

func (s *LevelDBStorage) UpdateDomain(z *happydns.Domain) error {
	return s.put(fmt.Sprintf("domain-%d", z.Id), z)
}

func (s *LevelDBStorage) UpdateDomainOwner(z *happydns.Domain, newOwner *happydns.User) error {
	z.IdUser = newOwner.Id
	return s.put(fmt.Sprintf("domain-%d", z.Id), z)
}

func (s *LevelDBStorage) DeleteDomain(z *happydns.Domain) error {
	return s.delete(fmt.Sprintf("domain-%d", z.Id))
}

func (s *LevelDBStorage) ClearDomains() error {
	err := s.ClearZones()
	if err != nil {
		return err
	}

	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("domain-")), nil)
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

func (s *LevelDBStorage) TidyDomains() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("domain-")), nil)
	defer iter.Release()

	for iter.Next() {
		domain, err := s.getDomain(string(iter.Key()))

		if err == nil {
			var u *happydns.User
			u, err = s.GetUser(domain.IdUser)
			if err == leveldb.ErrNotFound {
				// Drop domain of unexistant users
				err = tx.Delete(iter.Key(), nil)
				log.Printf("Deleting orphan domain (user %d not found): %v\n", domain.IdUser, domain)
			}

			_, err = s.GetSource(u, domain.IdSource)
			if err == leveldb.ErrNotFound {
				// Drop domain of unexistant source
				err = tx.Delete(iter.Key(), nil)
				log.Printf("Deleting orphan domain (source %d not found): %v\n", domain.IdSource, domain)
			}
		} else {
			// Drop unreadable domains
			log.Printf("Deleting unreadable domain (%w): %v\n", err, domain)
			err = tx.Delete(iter.Key(), nil)
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
