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
	"strings"

	"github.com/StackExchange/dnscontrol/v4/providers"
	"github.com/miekg/dns"
)

// Provider is where Domains and Zones can be managed.
type Provider interface {
	NewDNSServiceProvider() (providers.DNSServiceProvider, error)
	DNSControlName() string
}

// ProviderInfos describes the purpose of a user usable provider.
type ProviderInfos struct {
	// Name is the name displayed.
	Name string `json:"name"`

	// Description is a brief description of what the provider is.
	Description string `json:"description"`

	// Capabilites is a list of special ability of the provider (automatically filled).
	Capabilities []string `json:"capabilities,omitempty"`

	// HelpLink is the link to the documentation of the provider configuration.
	HelpLink string `json:"helplink,omitempty"`
}

// ProviderCreator abstract the instanciation of a Provider
type ProviderCreatorFunc func() Provider

// Provider aggregates way of create a Provider and information about it.
type ProviderCreator struct {
	Creator ProviderCreatorFunc
	Infos   ProviderInfos
}

// providers stores all existing Provider in happyDNS.
var providersList map[string]ProviderCreator = map[string]ProviderCreator{}

// RegisterProvider declares the existence of the given Provider.
func RegisterProvider(creator ProviderCreatorFunc, infos ProviderInfos) {
	prvInstance := creator()
	baseType := reflect.Indirect(reflect.ValueOf(prvInstance)).Type()
	name := baseType.Name()
	log.Println("Registering new provider:", name)

	infos.Capabilities = GetProviderCapabilities(prvInstance)
	infos.HelpLink = "https://docs.dnscontrol.org/service-providers/providers/" + strings.ToLower(prvInstance.DNSControlName())

	providersList[name] = ProviderCreator{
		creator,
		infos,
	}
}

// GetProviders retrieves the list of all existing Providers.
func GetProviders() *map[string]ProviderCreator {
	return &providersList
}

// FindProvider returns the Provider corresponding to the given name, or an error if it doesn't exist.
func FindProvider(name string) (Provider, error) {
	src, ok := providersList[name]
	if !ok {
		return nil, fmt.Errorf("Unable to find corresponding provider for `%s`.", name)
	}

	return src.Creator(), nil
}

// GetProviderCapabilities lists available capabilities for the given Provider.
func GetProviderCapabilities(prvd Provider) (caps []string) {
	// Features
	if providers.ProviderHasCapability(prvd.DNSControlName(), providers.CanGetZones) {
		caps = append(caps, "ListDomains")
	}

	// Compatible RR
	for _, v := range []uint16{dns.TypeA, dns.TypeAAAA, dns.TypeCNAME, dns.TypeMX, dns.TypeNS, dns.TypeTXT} {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", v, dns.TypeToString[v]))
	}
	if providers.ProviderHasCapability(prvd.DNSControlName(), providers.CanUseSOA) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeSOA, dns.TypeToString[dns.TypeSOA]))
	}
	if providers.ProviderHasCapability(prvd.DNSControlName(), providers.CanUseCAA) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeCAA, dns.TypeToString[dns.TypeCAA]))
	}
	if providers.ProviderHasCapability(prvd.DNSControlName(), providers.CanUseDS) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeDS, dns.TypeToString[dns.TypeDS]))
	}
	if providers.ProviderHasCapability(prvd.DNSControlName(), providers.CanUseNAPTR) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeNAPTR, dns.TypeToString[dns.TypeNAPTR]))
	}
	if providers.ProviderHasCapability(prvd.DNSControlName(), providers.CanUseOPENPGPKEY) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeOPENPGPKEY, dns.TypeToString[dns.TypeOPENPGPKEY]))
	}
	if providers.ProviderHasCapability(prvd.DNSControlName(), providers.CanUsePTR) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypePTR, dns.TypeToString[dns.TypePTR]))
	}
	if providers.ProviderHasCapability(prvd.DNSControlName(), providers.CanUseSRV) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeSRV, dns.TypeToString[dns.TypeSRV]))
	}
	if providers.ProviderHasCapability(prvd.DNSControlName(), providers.CanUseSSHFP) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeSSHFP, dns.TypeToString[dns.TypeSSHFP]))
	}
	if providers.ProviderHasCapability(prvd.DNSControlName(), providers.CanUseTLSA) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeTLSA, dns.TypeToString[dns.TypeTLSA]))
	}

	return
}
