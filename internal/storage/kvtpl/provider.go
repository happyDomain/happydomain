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

func (s *KVStorage) ListAllProviders() (happydns.Iterator[happydns.ProviderMessage], error) {
	iter := s.db.Search("provider-")
	return NewKVIterator[happydns.ProviderMessage](s.db, iter), nil
}

func (s *KVStorage) getProviderMeta(id happydns.Identifier) (*happydns.ProviderMessage, error) {
	srcMsg := &happydns.ProviderMessage{}
	err := s.db.Get(id.String(), srcMsg)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrProviderNotFound
	}
	return srcMsg, err
}

func (s *KVStorage) ListProviders(u *happydns.User) (srcs happydns.ProviderMessages, err error) {
	iter, err := s.ListAllProviders()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.Next() {
		srcMsg := iter.Item()
		if !bytes.Equal(srcMsg.Owner, u.Id) {
			continue
		}

		srcs = append(srcs, srcMsg)
	}

	return
}

func (s *KVStorage) GetProvider(id happydns.Identifier) (*happydns.ProviderMessage, error) {
	var prvdMsg happydns.ProviderMessage
	err := s.db.Get(fmt.Sprintf("provider-%s", id.String()), &prvdMsg)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrProviderNotFound
	}
	if err != nil {
		return nil, err
	}

	return &prvdMsg, nil
}

func (s *KVStorage) CreateProvider(prvd *happydns.Provider) error {
	key, id, err := s.db.FindIdentifierKey("provider-")
	if err != nil {
		return err
	}

	prvd.Id = id

	return s.db.Put(key, prvd)
}

func (s *KVStorage) UpdateProvider(prvd *happydns.Provider) error {
	return s.db.Put(fmt.Sprintf("provider-%s", prvd.Id.String()), prvd)
}

func (s *KVStorage) DeleteProvider(prvdId happydns.Identifier) error {
	return s.db.Delete(fmt.Sprintf("provider-%s", prvdId.String()))
}

func (s *KVStorage) ClearProviders() error {
	iter, err := s.ListAllProviders()
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
