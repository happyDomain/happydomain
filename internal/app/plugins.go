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

package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"plugin"

	"git.happydns.org/happyDomain/model"
)

func (a *App) LoadPlugins() error {
	a.pluginsIdx = map[string]happydns.TestPlugin{}

	var ret error

	for _, directory := range a.cfg.PluginsDirectories {
		files, err := os.ReadDir(directory)
		if err != nil {
			ret = errors.Join(ret, err)
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			fname := path.Join(directory, file.Name())

			err = a.loadPlugin(fname)
			if err != nil {
				ret = errors.Join(ret, fmt.Errorf("unable to load plugin %q: %w", fname, err))
			}
		}
	}

	return ret
}

func (a *App) loadPlugin(fname string) error {
	p, err := plugin.Open(fname)
	if err != nil {
		return err
	}

	newplugin, err := p.Lookup("NewTestPlugin")
	if err != nil {
		return err
	}

	myplugin, err := newplugin.(func() (happydns.TestPlugin, error))()
	if err != nil {
		return err
	}

	a.plugins = append(a.plugins, myplugin)

	// Index the plugin by its names
	pluginNames := myplugin.PluginEnvName()
	for _, name := range pluginNames {
		if p, exists := a.pluginsIdx[name]; exists {
			log.Printf("Plugin name conflict: the plugin at %q tries to register the name %q but it's already registered by %q", fname, name, p.Version().Name)
			continue
		}

		a.pluginsIdx[name] = myplugin
	}

	log.Printf("Plugin %s loaded (version %s)", myplugin.Version().Name, myplugin.Version().Version)
	return nil
}
