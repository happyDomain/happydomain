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

package provider

import (
	"fmt"
	"log"
	"reflect"
	"slices"

	"git.happydns.org/happyDomain/model"
)

// providerRegistry stores all existing Provider in happyDNS.
//
// The map is intentionally unguarded: all writes (RegisterProvider /
// RegisterProviderAs) happen from App.initPlugins() at startup, before any
// usecase or HTTP handler can read it (see internal/app/app.go). From that
// point on the registry is read-only for the rest of the process lifetime,
// so concurrent reads are safe without locking. Any future code path that
// needs to mutate it after startup must introduce its own synchronisation.
var providerRegistry = map[string]happydns.ProviderCreator{}

// RegisterProvider registers a provider definition globally under the
// unqualified Go type name of the value returned by creator(). This is the
// historical entry point used by built-in providers; the persisted
// happydns.Provider.Type field stores this same unqualified name, so
// changing the keying scheme here would break existing data.
func RegisterProvider(creator happydns.ProviderCreatorFunc, infos happydns.ProviderInfos) {
	provider := creator()
	baseType := reflect.Indirect(reflect.ValueOf(provider)).Type()
	RegisterProviderAs(baseType.Name(), creator, infos)
}

// RegisterProviderAs registers a provider definition globally under the
// caller-supplied name. It exists so that plugin loaders can pick a
// fully-qualified name (typically "package.Type") and avoid silently
// overwriting a built-in or another plugin that happens to expose a
// provider struct with the same short name.
//
// A second registration under an existing name is refused with a loud
// warning rather than overwriting the previous entry: in production this
// almost always indicates a deployment mistake (two plugins shipping the
// same provider, or a plugin shadowing a built-in).
func RegisterProviderAs(name string, creator happydns.ProviderCreatorFunc, infos happydns.ProviderInfos) {
	if _, exists := providerRegistry[name]; exists {
		log.Printf("Warning: provider %q is already registered; ignoring duplicate registration", name)
		return
	}
	log.Println("Registering new provider:", name)

	providerRegistry[name] = happydns.ProviderCreator{
		Creator: creator,
		Infos:   infos,
	}
}

// GetProviders returns all registered provider definitions.
func GetProviders() map[string]happydns.ProviderCreator {
	return providerRegistry
}

// ProviderHasCapability checks if the registered provider type has the given capability.
func ProviderHasCapability(provider *happydns.Provider, capability string) bool {
	creator, ok := providerRegistry[provider.Type]
	if !ok {
		return false
	}
	return slices.Contains(creator.Infos.Capabilities, capability)
}

// FindProvider returns the Provider corresponding to the given name, or an error if it doesn't exist.
func FindProvider(name string) (happydns.ProviderBody, error) {
	src, ok := providerRegistry[name]
	if !ok {
		return nil, fmt.Errorf("Unable to find corresponding provider for `%s`.", name)
	}

	return src.Creator(), nil
}
