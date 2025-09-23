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
	"errors"
	"fmt"
	"strings"

	"github.com/syndtr/goleveldb/leveldb/util"

	"git.happydns.org/happyDomain/model"
)

func (s *LevelDBStorage) ListAllPluginConfigurations() (happydns.Iterator[happydns.PluginOptions], error) {
	iter := s.search("plugincfg-")
	return NewLevelDBIterator[happydns.PluginOptions](s.db, iter), nil
}

func buildPluginKey(pname string, user *happydns.Identifier, domain *happydns.Identifier, service *happydns.Identifier) string {
	u := ""
	if user != nil {
		u = user.String()
	}

	d := ""
	if domain != nil {
		d = domain.String()
	}

	s := ""
	if service != nil {
		s = service.String()
	}

	return strings.Join([]string{pname, u, d, s}, "/")
}

func keyToPositional(key string, opts *happydns.PluginOptions) (*happydns.PluginOptionsPositional, error) {
	tmp := strings.Split(key, "/")

	if len(tmp) < 4 {
		return nil, fmt.Errorf("malformed plugin configuration key, got %q", key)
	}

	pname := tmp[0]

	var userid *happydns.Identifier
	if len(tmp[1]) > 0 {
		u, err := happydns.NewIdentifierFromString(tmp[1])
		if err != nil {
			return nil, err
		}
		userid = &u
	}

	var domainid *happydns.Identifier
	if len(tmp[2]) > 0 {
		d, err := happydns.NewIdentifierFromString(tmp[2])
		if err != nil {
			return nil, err
		}
		domainid = &d
	}

	var serviceid *happydns.Identifier
	if len(tmp[3]) > 0 {
		s, err := happydns.NewIdentifierFromString(tmp[3])
		if err != nil {
			return nil, err
		}
		serviceid = &s
	}

	return &happydns.PluginOptionsPositional{
		PluginName: pname,
		UserId:     userid,
		DomainId:   domainid,
		ServiceId:  serviceid,
		Options:    *opts,
	}, nil
}

func (s *LevelDBStorage) ListPluginConfiguration(pname string) (configs []*happydns.PluginOptionsPositional, err error) {
	iter := s.search("plugincfg-" + pname + "/")

	for iter.Next() {
		var p happydns.PluginOptions

		e := decodeData(iter.Value(), &p)
		if e != nil {
			err = errors.Join(err, e)
			continue
		}

		opts, e := keyToPositional(strings.TrimPrefix(string(iter.Key()), "plugincfg-"), &p)
		if e != nil {
			err = errors.Join(err, e)
			continue
		}

		configs = append(configs, opts)
	}

	return
}

func (s *LevelDBStorage) GetPluginConfiguration(pname string, user *happydns.Identifier, domain *happydns.Identifier, service *happydns.Identifier) (configs []*happydns.PluginOptionsPositional, err error) {
	iter := s.search("plugincfg-" + pname + "/")

	for iter.Next() {
		var p happydns.PluginOptions

		e := decodeData(iter.Value(), &p)
		if e != nil {
			err = errors.Join(err, e)
			continue
		}

		opts, e := keyToPositional(strings.TrimPrefix(string(iter.Key()), "plugincfg-"), &p)
		if e != nil {
			err = errors.Join(err, e)
			continue
		}

		if (user == nil || opts.UserId == nil || opts.UserId.Equals(*user)) && (opts.DomainId == nil || (domain != nil && opts.DomainId.Equals(*domain))) && (opts.ServiceId == nil || (service != nil && opts.ServiceId.Equals(*service))) {
			configs = append(configs, opts)
		}
	}

	return
}

func (s *LevelDBStorage) UpdatePluginConfiguration(pname string, user *happydns.Identifier, domain *happydns.Identifier, service *happydns.Identifier, opts happydns.PluginOptions) error {
	return s.put(fmt.Sprintf("plugincfg-%s", buildPluginKey(pname, user, domain, service)), opts)
}

func (s *LevelDBStorage) DeletePluginConfiguration(pname string, user *happydns.Identifier, domain *happydns.Identifier, service *happydns.Identifier) error {
	return s.delete(fmt.Sprintf("plugincfg-%s", buildPluginKey(pname, user, domain, service)))
}

func (s *LevelDBStorage) ClearPluginConfigurations() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("plugincfg-")), nil)
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
