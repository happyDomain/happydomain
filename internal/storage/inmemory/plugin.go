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

package inmemory

import (
	"errors"
	"fmt"
	"strings"

	"git.happydns.org/happyDomain/model"
)

func (s *InMemoryStorage) ListAllPluginConfigurations() (happydns.Iterator[happydns.PluginOptions], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return NewInMemoryIterator[happydns.PluginOptions](&s.pluginsCfg), nil
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

func (s *InMemoryStorage) ListPluginConfiguration(pname string) (configs []*happydns.PluginOptionsPositional, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, config := range s.pluginsCfg {
		opts, e := keyToPositional(k, config)
		if e != nil {
			err = errors.Join(err, e)
		} else if opts.PluginName == pname {
			configs = append(configs, opts)
		}
	}

	return
}

func (s *InMemoryStorage) GetPluginConfiguration(pname string, user *happydns.Identifier, domain *happydns.Identifier, service *happydns.Identifier) (configs []*happydns.PluginOptionsPositional, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, config := range s.pluginsCfg {
		opts, e := keyToPositional(k, config)
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

func (s *InMemoryStorage) UpdatePluginConfiguration(pname string, user *happydns.Identifier, domain *happydns.Identifier, service *happydns.Identifier, opts happydns.PluginOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pluginsCfg[buildPluginKey(pname, user, domain, service)] = &opts

	return nil
}

func (s *InMemoryStorage) DeletePluginConfiguration(pname string, user *happydns.Identifier, domain *happydns.Identifier, service *happydns.Identifier) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.pluginsCfg, buildPluginKey(pname, user, domain, service))

	return nil
}

func (s *InMemoryStorage) ClearPluginConfigurations() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pluginsCfg = make(map[string]*happydns.PluginOptions)
	return nil
}
