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

	"git.happydns.org/happyDomain/model"
)

type testPluginUsecase struct {
	config  *happydns.Options
	manager happydns.PluginManager
}

func NewTestPluginUsecase(cfg *happydns.Options, manager happydns.PluginManager) happydns.TestPluginUsecase {
	return &testPluginUsecase{
		config:  cfg,
		manager: manager,
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

func (tu *testPluginUsecase) ListTestPlugins() ([]happydns.TestPlugin, error) {
	return tu.manager.GetTestPlugins(), nil
}
