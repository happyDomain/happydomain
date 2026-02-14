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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"

	"git.happydns.org/happyDomain/checks"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/providers"
	"git.happydns.org/happyDomain/services"
)

// pluginLoader attempts to find and register one specific kind of plugin
// symbol from an already-opened .so file.
//
// It returns (true, nil) when the symbol was found and registration
// succeeded, (true, err) when the symbol was found but something went wrong,
// and (false, nil) when the symbol simply isn't present in that file (which
// is not considered an error — a single .so may implement only a subset of
// the known plugin types).
type pluginLoader func(p *plugin.Plugin, fname string) (found bool, err error)

// pluginLoaders is the authoritative list of plugin types that happyDomain
// knows about. To support a new plugin type, add a single entry here.
var pluginLoaders = []pluginLoader{
	loadCheckPlugin,
	loadProviderPlugin,
	loadServicePlugin,
}

// loadCheckPlugin handles the NewTestPlugin symbol.
func loadCheckPlugin(p *plugin.Plugin, fname string) (bool, error) {
	sym, err := p.Lookup("NewCheckPlugin")
	if err != nil {
		// Symbol not present in this .so — not an error.
		return false, nil
	}

	factory, ok := sym.(func() (string, happydns.Checker, error))
	if !ok {
		return true, fmt.Errorf("symbol NewCheckPlugin has unexpected type %T", sym)
	}

	pluginname, myplugin, err := factory()
	if err != nil {
		return true, err
	}

	checks.RegisterCheck(pluginname, myplugin)
	log.Printf("Plugin %s loaded", pluginname)
	return true, nil
}

// loadProviderPlugin handles the NewProviderPlugin symbol.
func loadProviderPlugin(_ *PluginManager, p *plugin.Plugin, fname string) (bool, error) {
	sym, err := p.Lookup("NewProviderPlugin")
	if err != nil {
		// Symbol not present in this .so — not an error.
		return false, nil
	}

	factory, ok := sym.(func() (happydns.ProviderCreatorFunc, happydns.ProviderInfos, error))
	if !ok {
		return true, fmt.Errorf("symbol NewProviderPlugin has unexpected type %T", sym)
	}

	creator, infos, err := factory()
	if err != nil {
		return true, err
	}

	providers.RegisterProvider(creator, infos)
	log.Printf("Plugin provider %q registered from %s", infos.Name, fname)
	return true, nil
}

// loadServicePlugin handles the NewServicePlugin symbol.
func loadServicePlugin(_ *PluginManager, p *plugin.Plugin, fname string) (bool, error) {
	sym, err := p.Lookup("NewServicePlugin")
	if err != nil {
		// Symbol not present in this .so — not an error.
		return false, nil
	}

	factory, ok := sym.(func() (happydns.ServiceCreator, svcs.ServiceAnalyzer, happydns.ServiceInfos, uint32, []string, error))
	if !ok {
		return true, fmt.Errorf("symbol NewServicePlugin has unexpected type %T", sym)
	}

	creator, analyzer, infos, weight, aliases, err := factory()
	if err != nil {
		return true, err
	}

	svcs.RegisterService(creator, analyzer, infos, weight, aliases...)
	log.Printf("Plugin service %q registered from %s", infos.Name, fname)
	return true, nil
}

// initPlugins scans each directory listed in cfg.PluginsDirectories, loads
// every .so file found as a Go plugin, and registers it in the application's
// PluginManager. All load errors are collected and returned as a joined error
// so that a single bad plugin does not prevent the others from loading.
func (a *App) initPlugins() error {
	for _, directory := range a.cfg.PluginsDirectories {
		files, err := os.ReadDir(directory)
		if err != nil {
			return fmt.Errorf("unable to read plugins directory %q: %s", directory, err)
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			// Only attempt to load shared-object files.
			if filepath.Ext(file.Name()) != ".so" {
				continue
			}

			fname := filepath.Join(directory, file.Name())

			err = loadPlugin(fname)
			if err != nil {
				log.Printf("Unable to load plugin %q: %s", fname, err)
			}
		}
	}

	return nil
}

// loadPlugin opens the .so file at fname and runs every registered
// pluginLoader against it. A loader that does not find its symbol is silently
// skipped. If no loader recognises any symbol in the file a warning is logged,
// but no error is returned because the file might be a valid plugin for a
// future version of happyDomain. The first loader error that is encountered
// is returned immediately.
func loadPlugin(fname string) error {
	p, err := plugin.Open(fname)
	if err != nil {
		return err
	}

	anyFound := false
	for _, loader := range pluginLoaders {
		found, err := loader(p, fname)
		if err != nil {
			return err
		}
		if found {
			anyFound = true
		}
	}

	if !anyFound {
		log.Printf("Warning: plugin %q exports no recognised symbols", fname)
	}
	return nil
}
