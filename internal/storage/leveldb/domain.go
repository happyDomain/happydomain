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

func (s *LevelDBStorage) ListAllDomains() (domains happydns.Domains, err error) {
	iter := s.search("domain-")
	defer iter.Release()

	for iter.Next() {
		var domain happydns.Domain

		err = decodeData(iter.Value(), &domain)
		if err != nil {
			return
		}

		domains = append(domains, &domain)
	}

	return
}

func (s *LevelDBStorage) ListDomains(u *happydns.User) (domains happydns.Domains, err error) {
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

func (s *LevelDBStorage) GetDomain(id happydns.Identifier) (*happydns.Domain, error) {
	return s.getDomain(fmt.Sprintf("domain-%s", id.String()))
}

func (s *LevelDBStorage) GetDomainByDN(u *happydns.User, dn string) ([]*happydns.Domain, error) {
	domains, err := s.ListDomains(u)
	if err != nil {
		return nil, err
	}

	var ret []*happydns.Domain
	for _, domain := range domains {
		if domain.DomainName == dn {
			ret = append(ret, domain)
		}
	}

	if len(ret) == 0 {
		return nil, leveldb.ErrNotFound
	}

	return ret, nil
}

func (s *LevelDBStorage) CreateDomain(z *happydns.Domain) error {
	key, id, err := s.findIdentifierKey("domain-")
	if err != nil {
		return err
	}

	z.Id = id
	return s.put(key, z)
}

func (s *LevelDBStorage) UpdateDomain(z *happydns.Domain) error {
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
			_, err = s.GetUser(domain.Owner)
			if err == leveldb.ErrNotFound {
				// Drop domain of unexistant users
				err = tx.Delete(iter.Key(), nil)
				log.Printf("Deleting orphan domain (user %s not found): %v\n", domain.Owner.String(), domain)
			}

			_, err = s.GetProvider(domain.IdProvider)
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
