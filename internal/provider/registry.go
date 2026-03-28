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
var providerRegistry = map[string]happydns.ProviderCreator{}

// RegisterProvider registers a provider definition globally.
func RegisterProvider(creator happydns.ProviderCreatorFunc, infos happydns.ProviderInfos) {
	provider := creator()
	baseType := reflect.Indirect(reflect.ValueOf(provider)).Type()
	name := baseType.Name()
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
