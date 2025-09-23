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

package plugin

import (
	"fmt"
	"sort"

	"git.happydns.org/happyDomain/model"
)

type testPluginUsecase struct {
	config  *happydns.Options
	manager happydns.PluginManager
	store   PluginStorage
}

func NewTestPluginUsecase(cfg *happydns.Options, manager happydns.PluginManager, store PluginStorage) happydns.TestPluginUsecase {
	return &testPluginUsecase{
		config:  cfg,
		manager: manager,
		store:   store,
	}
}

func (tu *testPluginUsecase) GetTestPlugin(pname string) (happydns.TestPlugin, error) {
	plugin, ok := tu.manager.GetTestPlugin(pname)
	if !ok {
		return nil, fmt.Errorf("unable to find plugin named %q", pname)
	} else {
		return plugin, nil
	}
}

type ByOptionPosition []*happydns.PluginOptionsPositional

func (a ByOptionPosition) Len() int      { return len(a) }
func (a ByOptionPosition) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByOptionPosition) Less(i, j int) bool {
	if a[i].PluginName != a[j].PluginName {
		return a[i].PluginName < a[j].PluginName
	}

	if res := compareIdentifiers(a[i].UserId, a[j].UserId); res != 0 {
		return res < 0
	}

	if res := compareIdentifiers(a[i].DomainId, a[j].DomainId); res != 0 {
		return res < 0
	}

	if res := compareIdentifiers(a[i].ServiceId, a[j].ServiceId); res != 0 {
		return res < 0
	}

	return false
}

func compareIdentifiers(a, b *happydns.Identifier) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	if a.Equals(*b) {
		return 0
	}

	return a.Compare(*b)
}

func (tu *testPluginUsecase) GetTestPluginOptions(pname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier) (*happydns.PluginOptions, error) {
	configs, err := tu.store.GetPluginConfiguration(pname, userid, domainid, serviceid)
	if err != nil {
		return nil, err
	}

	sort.Sort(ByOptionPosition(configs))

	opts := make(happydns.PluginOptions)

	for _, c := range configs {
		for k, v := range c.Options {
			opts[k] = v
		}
	}

	return &opts, nil
}

func (tu *testPluginUsecase) ListTestPlugins() ([]happydns.TestPlugin, error) {
	return tu.manager.GetTestPlugins(), nil
}

func (tu *testPluginUsecase) SetTestPluginOptions(pname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier, opts happydns.PluginOptions) error {
	return tu.store.UpdatePluginConfiguration(pname, userid, domainid, serviceid, opts)
}

func (tu *testPluginUsecase) OverwriteSomeTestPluginOptions(pname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier, opts happydns.PluginOptions) error {
	current, err := tu.GetTestPluginOptions(pname, userid, domainid, serviceid)
	if err != nil {
		return err
	}

	for k, v := range opts {
		(*current)[k] = v
	}

	return tu.store.UpdatePluginConfiguration(pname, userid, domainid, serviceid, *current)
}
