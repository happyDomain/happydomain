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

	"git.happydns.org/happyDomain/model"
)

func (s *KVStorage) ListAllCheckerConfigurations() (happydns.Iterator[happydns.CheckerOptions], error) {
	iter := s.db.Search("chckrcfg-")
	return NewKVIterator[happydns.CheckerOptions](s.db, iter), nil
}

func buildCheckerKey(cname string, user *happydns.Identifier, domain *happydns.Identifier, service *happydns.Identifier) string {
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

	return strings.Join([]string{cname, u, d, s}, "/")
}

func keyToPositional(key string, opts *happydns.CheckerOptions) (*happydns.CheckerOptionsPositional, error) {
	tmp := strings.Split(key, "/")

	if len(tmp) < 4 {
		return nil, fmt.Errorf("malformed plugin configuration key, got %q", key)
	}

	cname := tmp[0]

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

	return &happydns.CheckerOptionsPositional{
		CheckName: cname,
		UserId:    userid,
		DomainId:  domainid,
		ServiceId: serviceid,
		Options:   *opts,
	}, nil
}

func (s *KVStorage) ListCheckerConfiguration(cname string) (configs []*happydns.CheckerOptionsPositional, err error) {
	iter := s.db.Search("chckrcfg-" + cname + "/")
	defer iter.Release()

	for iter.Next() {
		var p happydns.CheckerOptions

		e := s.db.DecodeData(iter.Value(), &p)
		if e != nil {
			err = errors.Join(err, e)
			continue
		}

		opts, e := keyToPositional(strings.TrimPrefix(iter.Key(), "chckrcfg-"), &p)
		if e != nil {
			err = errors.Join(err, e)
			continue
		}

		configs = append(configs, opts)
	}

	return
}

func (s *KVStorage) GetCheckerConfiguration(cname string, user *happydns.Identifier, domain *happydns.Identifier, service *happydns.Identifier) (configs []*happydns.CheckerOptionsPositional, err error) {
	iter := s.db.Search("chckrcfg-" + cname + "/")
	defer iter.Release()

	for iter.Next() {
		var p happydns.CheckerOptions

		e := s.db.DecodeData(iter.Value(), &p)
		if e != nil {
			err = errors.Join(err, e)
			continue
		}

		opts, e := keyToPositional(strings.TrimPrefix(iter.Key(), "chckrcfg-"), &p)
		if e != nil {
			err = errors.Join(err, e)
			continue
		}

		// Match logic:
		// - When parameter is nil: match ONLY configs with nil ID (requesting specific scope)
		// - When parameter is not nil: match configs with nil ID (admin-level) OR matching ID
		matchUser := (user == nil && opts.UserId == nil) ||
			(user != nil && (opts.UserId == nil || opts.UserId.Equals(*user)))

		matchDomain := (domain == nil && opts.DomainId == nil) ||
			(domain != nil && (opts.DomainId == nil || opts.DomainId.Equals(*domain)))

		matchService := (service == nil && opts.ServiceId == nil) ||
			(service != nil && (opts.ServiceId == nil || opts.ServiceId.Equals(*service)))

		if matchUser && matchDomain && matchService {
			configs = append(configs, opts)
		}
	}

	return
}

func (s *KVStorage) UpdateCheckerConfiguration(cname string, user *happydns.Identifier, domain *happydns.Identifier, service *happydns.Identifier, opts happydns.CheckerOptions) error {
	return s.db.Put(fmt.Sprintf("chckrcfg-%s", buildCheckerKey(cname, user, domain, service)), opts)
}

func (s *KVStorage) DeleteCheckerConfiguration(cname string, user *happydns.Identifier, domain *happydns.Identifier, service *happydns.Identifier) error {
	return s.db.Delete(fmt.Sprintf("chckrcfg-%s", buildCheckerKey(cname, user, domain, service)))
}

func (s *KVStorage) ClearCheckerConfigurations() error {
	iter := s.db.Search("chckrcfg-")
	defer iter.Release()

	for iter.Next() {
		err := s.db.Delete(iter.Key())
		if err != nil {
			return err
		}
	}

	return nil
}
