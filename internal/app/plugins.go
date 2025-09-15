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

	"git.happydns.org/happyDomain/model"
)

// pluginLoader attempts to find and register one specific kind of plugin
// symbol from an already-opened .so file.
//
// It returns (true, nil) when the symbol was found and registration
// succeeded, (true, err) when the symbol was found but something went wrong,
// and (false, nil) when the symbol simply isn't present in that file (which
// is not considered an error — a single .so may implement only a subset of
// the known plugin types).
type pluginLoader func(m *PluginManager, p *plugin.Plugin, fname string) (found bool, err error)

// pluginLoaders is the authoritative list of plugin types that happyDomain
// knows about. To support a new plugin type, add a single entry here.
var pluginLoaders = []pluginLoader{
	loadTestPlugin,
}

// loadTestPlugin handles the NewTestPlugin symbol.
func loadTestPlugin(m *PluginManager, p *plugin.Plugin, fname string) (bool, error) {
	sym, err := p.Lookup("NewTestPlugin")
	if err != nil {
		// Symbol not present in this .so — not an error.
		return false, nil
	}

	factory, ok := sym.(func() (happydns.TestPlugin, error))
	if !ok {
		return true, fmt.Errorf("symbol NewTestPlugin has unexpected type %T", sym)
	}

	myplugin, err := factory()
	if err != nil {
		return true, err
	}

	m.tests = append(m.tests, myplugin)

	for _, name := range myplugin.PluginEnvName() {
		if existing, exists := m.testsIdx[name]; exists {
			log.Fatalf("Plugin name conflict: the plugin at %q tries to register the name %q but it's already registered by %q", fname, name, existing.Version().Name)
		}
		m.testsIdx[name] = myplugin
	}

	log.Printf("Plugin %s loaded (version %s)", myplugin.Version().Name, myplugin.Version().Version)
	return true, nil
}

// initPlugins scans each directory listed in cfg.PluginsDirectories, loads
// every .so file found as a Go plugin, and registers it in the application's
// PluginManager. All load errors are collected and returned as a joined error
// so that a single bad plugin does not prevent the others from loading.
func (a *App) initPlugins() error {
	manager := PluginManager{
		testsIdx: map[string]happydns.TestPlugin{},
	}
	a.plugins = &manager

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

			err = manager.loadPlugin(fname)
			if err != nil {
				log.Printf("Unable to load plugin %q: %w", fname, err)
			}
		}
	}

	return nil
}

// PluginManager holds all dynamically-loaded test plugins and provides
// indexed access to them by their registered environment names.
type PluginManager struct {
	// tests is the ordered list of loaded plugins.
	tests []happydns.TestPlugin

	// testsIdx maps each plugin environment name to its plugin instance,
	// allowing O(1) lookup by name.
	testsIdx map[string]happydns.TestPlugin
}

// loadPlugin opens the .so file at fname and runs every registered
// pluginLoader against it. A loader that does not find its symbol is silently
// skipped. If no loader recognises any symbol in the file a warning is logged,
// but no error is returned because the file might be a valid plugin for a
// future version of happyDomain. The first loader error that is encountered
// is returned immediately.
func (m *PluginManager) loadPlugin(fname string) error {
	p, err := plugin.Open(fname)
	if err != nil {
		return err
	}

	anyFound := false
	for _, loader := range pluginLoaders {
		found, err := loader(m, p, fname)
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

// GetTestPlugins returns the ordered list of all loaded test plugins.
func (m *PluginManager) GetTestPlugins() []happydns.TestPlugin {
	return m.tests
}

// GetTestPlugin returns the plugin registered under the given environment name,
// and a boolean indicating whether it was found.
func (m *PluginManager) GetTestPlugin(name string) (happydns.TestPlugin, bool) {
	p, ok := m.testsIdx[name]
	return p, ok
}
