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

package providerregistry

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"git.happydns.org/happyDomain/model"
)

// providerRegistry stores all existing Provider in happyDNS.
var providerRegistry = map[string]happydns.ProviderCreator{}

// nameserverMatcher recognizes the name servers of a provider.
type nameserverMatcher []*regexp.Regexp

// nameserverMatchers stores the compiled ProviderInfos.Nameservers of the
// registered providers having some.
var nameserverMatchers = map[string]nameserverMatcher{}

// RegisterProvider registers a provider definition globally.
func RegisterProvider(creator happydns.ProviderCreatorFunc, infos happydns.ProviderInfos) {
	provider := creator()
	baseType := reflect.Indirect(reflect.ValueOf(provider)).Type()
	name := baseType.Name()

	// Always expose a list, even empty, rather than null.
	if infos.Capabilities == nil {
		infos.Capabilities = []string{}
	}

	providerRegistry[name] = happydns.ProviderCreator{
		Creator: creator,
		Infos:   infos,
	}

	if len(infos.Nameservers) > 0 {
		var matcher nameserverMatcher
		for _, pattern := range infos.Nameservers {
			matcher = append(matcher, regexp.MustCompile(`^(?:`+pattern+`)$`))
		}
		nameserverMatchers[name] = matcher
	}
}

// match tells whether ns, in lower case and without its final dot, is a name
// server of the provider.
func (m nameserverMatcher) match(ns string) bool {
	return slices.ContainsFunc(m, func(re *regexp.Regexp) bool { return re.MatchString(ns) })
}

// MatchNameservers returns, sorted, the types of the providers at least one of
// the given name servers belongs to.
func MatchNameservers(nameservers []string) []string {
	types := []string{}

	for name, matcher := range nameserverMatchers {
		if slices.ContainsFunc(nameservers, func(ns string) bool {
			return matcher.match(strings.TrimSuffix(strings.ToLower(ns), "."))
		}) {
			types = append(types, name)
		}
	}

	slices.Sort(types)
	return types
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
