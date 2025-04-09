// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

package providers // import "git.happydns.org/happyDomain/providers"

import (
	"fmt"
	"log"
	"reflect"

	"git.happydns.org/happyDomain/model"
)

// providers stores all existing Provider in happyDNS.
var providersList map[string]happydns.ProviderCreator = map[string]happydns.ProviderCreator{}

// RegisterProvider declares the existence of the given Provider.
func RegisterProvider(creator happydns.ProviderCreatorFunc, infos happydns.ProviderInfos) {
	provider := creator()
	baseType := reflect.Indirect(reflect.ValueOf(provider)).Type()
	name := baseType.Name()
	log.Println("Registering new provider:", name)

	providersList[name] = happydns.ProviderCreator{
		Creator: creator,
		Infos:   infos,
	}
}

// GetProviders retrieves the list of all existing Providers.
func GetProviders() *map[string]happydns.ProviderCreator {
	return &providersList
}

// FindProvider returns the Provider corresponding to the given name, or an error if it doesn't exist.
func FindProvider(name string) (happydns.ProviderBody, error) {
	src, ok := providersList[name]
	if !ok {
		return nil, fmt.Errorf("Unable to find corresponding provider for `%s`.", name)
	}

	return src.Creator(), nil
}
