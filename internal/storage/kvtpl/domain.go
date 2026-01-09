// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

func (s *KVStorage) ListAllDomains() (happydns.Iterator[happydns.Domain], error) {
	iter := s.db.Search("domain-")
	return NewKVIterator[happydns.Domain](s.db, iter), nil
}

func (s *KVStorage) ListDomains(u *happydns.User) (domains []*happydns.Domain, err error) {
	iter := s.db.Search("domain-")
	defer iter.Release()

	for iter.Next() {
		var z happydns.Domain

		err = s.db.DecodeData(iter.Value(), &z)
		if err != nil {
			return
		}

		if bytes.Equal(z.Owner, u.Id) {
			domains = append(domains, &z)
		}
	}

	return
}

func (s *KVStorage) getDomain(id string) (*happydns.Domain, error) {
	domain := &happydns.Domain{}
	err := s.db.Get(id, domain)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrDomainNotFound
	}
	return domain, err
}

func (s *KVStorage) GetDomain(id happydns.Identifier) (*happydns.Domain, error) {
	return s.getDomain(fmt.Sprintf("domain-%s", id.String()))
}

func (s *KVStorage) GetDomainByDN(u *happydns.User, dn string) ([]*happydns.Domain, error) {
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
		return nil, happydns.ErrNotFound
	}

	return ret, nil
}

func (s *KVStorage) CreateDomain(z *happydns.Domain) error {
	key, id, err := s.db.FindIdentifierKey("domain-")
	if err != nil {
		return err
	}

	z.Id = id
	return s.db.Put(key, z)
}

func (s *KVStorage) UpdateDomain(z *happydns.Domain) error {
	return s.db.Put(fmt.Sprintf("domain-%s", z.Id.String()), z)
}

func (s *KVStorage) DeleteDomain(zId happydns.Identifier) error {
	return s.db.Delete(fmt.Sprintf("domain-%s", zId.String()))
}

func (s *KVStorage) ClearDomains() error {
	err := s.ClearZones()
	if err != nil {
		return err
	}

	iter, err := s.ListAllDomains()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		err = s.db.Delete(iter.Key())
		if err != nil {
			return err
		}
	}

	return nil
}
