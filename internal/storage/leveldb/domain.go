// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package database

import (
	"bytes"
	"fmt"
	"log"

	"git.happydns.org/happyDomain/model"

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

		if bytes.Equal(z.Owner, u.Id) {
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

func (s *LevelDBStorage) GetDomain(u *happydns.User, id happydns.Identifier) (z *happydns.Domain, err error) {
	z, err = s.getDomain(fmt.Sprintf("domain-%s", id.String()))

	if err != nil {
		return
	}

	if !bytes.Equal(z.Owner, u.Id) {
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

func (s *LevelDBStorage) CreateDomain(u *happydns.User, z *happydns.Domain) error {
	key, id, err := s.findIdentifierKey("domain-")
	if err != nil {
		return err
	}

	z.Id = id
	z.Owner = u.Id
	return s.put(key, z)
}

func (s *LevelDBStorage) UpdateDomain(z *happydns.Domain) error {
	return s.put(fmt.Sprintf("domain-%s", z.Id.String()), z)
}

func (s *LevelDBStorage) UpdateDomainOwner(z *happydns.Domain, newOwner *happydns.User) error {
	z.Owner = newOwner.Id
	return s.put(fmt.Sprintf("domain-%s", z.Id.String()), z)
}

func (s *LevelDBStorage) DeleteDomain(zId happydns.Identifier) error {
	return s.delete(fmt.Sprintf("domain-%s", zId.String()))
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
			u, err = s.GetUser(domain.Owner)
			if err == leveldb.ErrNotFound {
				// Drop domain of unexistant users
				err = tx.Delete(iter.Key(), nil)
				log.Printf("Deleting orphan domain (user %s not found): %v\n", domain.Owner.String(), domain)
			}

			_, err = s.GetProvider(u, domain.IdProvider)
			if err == leveldb.ErrNotFound {
				// Drop domain of unexistant provider
				err = tx.Delete(iter.Key(), nil)
				log.Printf("Deleting orphan domain (provider %s not found): %v\n", domain.IdProvider.String(), domain)
			}
		} else {
			// Drop unreadable domains
			log.Printf("Deleting unreadable domain (%s): %v\n", err.Error(), domain)
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
