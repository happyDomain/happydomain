// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

//go:build linux || darwin || freebsd

package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"reflect"

	sdk "git.happydns.org/checker-sdk-go/checker"
	"git.happydns.org/happyDomain/internal/checker"
	providerReg "git.happydns.org/happyDomain/internal/provider"
	svcs "git.happydns.org/happyDomain/internal/service"
	"git.happydns.org/happyDomain/model"
)

// pluginSymbols is the minimal subset of *plugin.Plugin used by the loaders.
// It exists so that loaders can be unit-tested with a fake instead of
// requiring a real .so file built via `go build -buildmode=plugin`.
type pluginSymbols interface {
	Lookup(symName string) (plugin.Symbol, error)
}

// pluginLoader attempts to find and register one specific kind of plugin
// symbol from an already-opened .so file.
//
// It returns (true, nil) when the symbol was found and registration
// succeeded, (true, err) when the symbol was found but something went wrong,
// and (false, nil) when the symbol simply isn't present in that file (which
// is not considered an error: a single .so may implement only a subset of
// the known plugin types).
type pluginLoader func(p pluginSymbols, fname string) (found bool, err error)

// safeCall invokes fn while recovering from any panic raised by plugin code.
// A panicking factory must not take the whole server down at startup; the
// recovered value is converted to an error so the caller can log/skip the
// offending plugin like any other failure.
func safeCall(symbol string, fname string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("plugin %q panicked in %s: %v", fname, symbol, r)
		}
	}()
	return fn()
}

// pluginLoaders is the authoritative list of plugin types that happyDomain
// knows about. To support a new plugin type, add a single entry here.
var pluginLoaders = []pluginLoader{
	loadCheckerPlugin,
	loadProviderPlugin,
	loadServicePlugin,
}

// loadCheckerPlugin handles the NewCheckerPlugin symbol exported by checkers
// built against checker-sdk-go (see ../../checker-dummy/README.md).
func loadCheckerPlugin(p pluginSymbols, fname string) (bool, error) {
	sym, err := p.Lookup("NewCheckerPlugin")
	if err != nil {
		// Symbol not present in this .so, not an error.
		return false, nil
	}

	factory, ok := sym.(func() (*sdk.CheckerDefinition, sdk.ObservationProvider, error))
	if !ok {
		return true, fmt.Errorf("symbol NewCheckerPlugin has unexpected type %T", sym)
	}

	var (
		def      *sdk.CheckerDefinition
		provider sdk.ObservationProvider
	)
	if err := safeCall("NewCheckerPlugin", fname, func() error {
		var ferr error
		def, provider, ferr = factory()
		return ferr
	}); err != nil {
		return true, err
	}
	if def == nil {
		return true, fmt.Errorf("NewCheckerPlugin returned a nil CheckerDefinition")
	}
	if provider == nil {
		return true, fmt.Errorf("NewCheckerPlugin returned a nil ObservationProvider")
	}

	checker.RegisterObservationProvider(provider)
	checker.RegisterExternalizableChecker(def)
	log.Printf("Plugin %s (%s) loaded", def.ID, fname)
	return true, nil
}

// loadProviderPlugin handles the NewProviderPlugin symbol exported by DNS
// provider plugins. The factory returns the creator/infos pair that the
// provider registry expects.
func loadProviderPlugin(p pluginSymbols, fname string) (bool, error) {
	sym, err := p.Lookup("NewProviderPlugin")
	if err != nil {
		// Symbol not present in this .so, not an error.
		return false, nil
	}

	factory, ok := sym.(func() (happydns.ProviderCreatorFunc, happydns.ProviderInfos, error))
	if !ok {
		return true, fmt.Errorf("symbol NewProviderPlugin has unexpected type %T", sym)
	}

	var (
		creator happydns.ProviderCreatorFunc
		infos   happydns.ProviderInfos
	)
	if err := safeCall("NewProviderPlugin", fname, func() error {
		var ferr error
		creator, infos, ferr = factory()
		return ferr
	}); err != nil {
		return true, err
	}
	if creator == nil {
		return true, fmt.Errorf("NewProviderPlugin returned a nil ProviderCreatorFunc")
	}

	// Plugin-registered providers go through the qualified-name API so that
	// two plugins exporting providers with the same struct name (in different
	// packages) cannot silently overwrite each other in the global registry.
	sample := creator()
	baseType := reflect.Indirect(reflect.ValueOf(sample)).Type()
	qualified := baseType.String()

	providerReg.RegisterProviderAs(qualified, creator, infos)
	log.Printf("Plugin provider %q registered as %q (%s)", infos.Name, qualified, fname)
	return true, nil
}

// loadServicePlugin handles the NewServicePlugin symbol exported by service
// plugins. The factory returns the creator/analyzer/infos triple along with
// the analyzer weight and any aliases the service should be reachable under.
func loadServicePlugin(p pluginSymbols, fname string) (bool, error) {
	sym, err := p.Lookup("NewServicePlugin")
	if err != nil {
		// Symbol not present in this .so, not an error.
		return false, nil
	}

	factory, ok := sym.(func() (happydns.ServiceCreator, svcs.ServiceAnalyzer, happydns.ServiceInfos, uint32, []string, error))
	if !ok {
		return true, fmt.Errorf("symbol NewServicePlugin has unexpected type %T", sym)
	}

	var (
		creator  happydns.ServiceCreator
		analyzer svcs.ServiceAnalyzer
		infos    happydns.ServiceInfos
		weight   uint32
		aliases  []string
	)
	if err := safeCall("NewServicePlugin", fname, func() error {
		var ferr error
		creator, analyzer, infos, weight, aliases, ferr = factory()
		return ferr
	}); err != nil {
		return true, err
	}
	if creator == nil {
		return true, fmt.Errorf("NewServicePlugin returned a nil ServiceCreator")
	}

	svcs.RegisterService(creator, analyzer, infos, weight, aliases...)

	// The built-in sub-service walker only descends into types whose package
	// path lives under git.happydns.org/happyDomain/services. Plugin services
	// live elsewhere, so we must explicitly walk their type tree to register
	// any nested struct types as sub-services; otherwise (de)serialisation
	// of plugin payloads breaks for anything more than a flat struct.
	baseType := reflect.Indirect(reflect.ValueOf(creator())).Type()
	svcs.RegisterPluginSubServices(baseType)

	log.Printf("Plugin service %q (%s) loaded", infos.Name, fname)
	return true, nil
}

// checkPluginDirectoryPermissions refuses to load plugins from a directory
// that any non-owner can write to. Loading a .so file is arbitrary code
// execution as the happyDomain process, so a world- or group-writable
// plugin directory is treated as a fatal misconfiguration: any local user
// (or any process sharing the group) able to drop a file there could take
// over the server. Operators who genuinely need shared deployment should
// stage plugins elsewhere and rsync them into a directory owned and
// writable only by the happyDomain user.
func checkPluginDirectoryPermissions(directory string) error {
	// Use Lstat to detect symlinks: a symlink could be silently redirected
	// to an attacker-controlled directory, bypassing the permission check
	// on the original path.
	linfo, err := os.Lstat(directory)
	if err != nil {
		return fmt.Errorf("unable to stat plugins directory %q: %s", directory, err)
	}
	if linfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("plugins directory %q is a symbolic link; refusing to follow it", directory)
	}
	if !linfo.IsDir() {
		return fmt.Errorf("plugins path %q is not a directory", directory)
	}
	mode := linfo.Mode().Perm()
	if mode&0o022 != 0 {
		return fmt.Errorf("plugins directory %q is group- or world-writable (mode %#o); refusing to load plugins from it", directory, mode)
	}
	return nil
}

// checkPluginFilePermissions refuses to load a .so file that is group- or
// world-writable. Even inside a properly locked-down directory, a writable
// plugin binary could be replaced by a malicious actor sharing the group.
// Symlinks are followed: the permission check applies to the resolved target,
// which allows the common pattern of symlinking to versioned binaries
// (e.g. checker-foo.so -> checker-foo-v1.2.so) for atomic upgrades.
// The directory-level symlink ban already prevents attackers from redirecting
// the scan root itself.
func checkPluginFilePermissions(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("unable to stat plugin file %q: %s", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("plugin %q is not a regular file (or resolves to a non-regular file)", path)
	}
	mode := info.Mode().Perm()
	if mode&0o022 != 0 {
		return fmt.Errorf("plugin file %q is group- or world-writable (mode %#o)", path, mode)
	}
	return nil
}

// initPlugins scans each directory listed in cfg.PluginsDirectories and loads
// every .so file found as a Go plugin. A directory that cannot be read is a
// fatal configuration error; individual plugin failures are logged and
// skipped so that one bad .so does not prevent the others from loading.
func (a *App) initPlugins() error {
	for _, directory := range a.cfg.PluginsDirectories {
		if err := checkPluginDirectoryPermissions(directory); err != nil {
			return err
		}

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

			if err := checkPluginFilePermissions(fname); err != nil {
				log.Printf("Skipping plugin %q: %s", fname, err)
				continue
			}

			if err := loadPlugin(fname); err != nil {
				log.Printf("Unable to load plugin %q: %s", fname, err)
			}
		}
	}

	return nil
}

// loadPlugin opens the .so file at fname and runs every registered
// pluginLoader against it. A loader that does not find its symbol is silently
// skipped. If no loader recognises any symbol in the file a warning is
// logged, because the file might be a valid plugin for a future version of
// happyDomain. Loader errors for one plugin kind do not prevent the other
// kinds in the same .so from being attempted: a single .so is allowed to
// expose more than one plugin type, and a failure to register (e.g.) the
// service half should not silently drop the checker half. All loader errors
// encountered are joined and returned together.
func loadPlugin(fname string) error {
	p, err := plugin.Open(fname)
	if err != nil {
		return err
	}

	var (
		anyFound bool
		errs     []error
	)
	for _, loader := range pluginLoaders {
		found, err := loader(p, fname)
		if found {
			anyFound = true
		}
		if err != nil {
			errs = append(errs, err)
		}
	}

	if !anyFound && len(errs) == 0 {
		log.Printf("Warning: plugin %q exports no recognised symbols", fname)
	}
	return errors.Join(errs...)
}
