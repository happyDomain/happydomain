// Copyright or © or Copr. happyDNS (2021)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the provider code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package providers // import "happydns.org/providers"

import (
	"fmt"
	"log"
	"reflect"

	"github.com/StackExchange/dnscontrol/v3/providers"
	"github.com/miekg/dns"

	"git.happydns.org/happydomain/model"
)

// ProviderInfos describes the purpose of a user usable provider.
type ProviderInfos struct {
	// Name is the name displayed.
	Name string `json:"name"`

	// Description is a brief description of what the provider is.
	Description string `json:"description"`

	// Capabilites is a list of special ability of the provider (automatically filled).
	Capabilities []string `json:"capabilities,omitempty"`
}

// ProviderCreator abstract the instanciation of a Provider
type ProviderCreator func() happydns.Provider

// Provider aggregates way of create a Provider and information about it.
type Provider struct {
	Creator ProviderCreator
	Infos   ProviderInfos
}

// providers stores all existing Provider in happyDNS.
var providersList map[string]Provider = map[string]Provider{}

// RegisterProvider declares the existence of the given Provider.
func RegisterProvider(creator ProviderCreator, infos ProviderInfos) {
	baseType := reflect.Indirect(reflect.ValueOf(creator())).Type()
	name := baseType.Name()
	log.Println("Registering new provider:", name)

	infos.Capabilities = GetProviderCapabilities(creator())

	providersList[name] = Provider{
		creator,
		infos,
	}
}

// GetProviders retrieves the list of all existing Providers.
func GetProviders() *map[string]Provider {
	return &providersList
}

// FindProvider returns the Provider corresponding to the given name, or an error if it doesn't exist.
func FindProvider(name string) (happydns.Provider, error) {
	src, ok := providersList[name]
	if !ok {
		return nil, fmt.Errorf("Unable to find corresponding provider for `%s`.", name)
	}

	return src.Creator(), nil
}

// GetProviderCapabilities lists available capabilities for the given Provider.
func GetProviderCapabilities(prvd happydns.Provider) (caps []string) {
	// Features
	if providers.ProviderHasCapability(prvd.DNSControlName(), providers.CanGetZones) {
		caps = append(caps, "ListDomains")
	}

	// Compatible RR
	for _, v := range []uint16{dns.TypeA, dns.TypeAAAA, dns.TypeCNAME, dns.TypeNS, dns.TypeTXT} {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", v, dns.TypeToString[v]))
	}
	if providers.ProviderHasCapability(prvd.DNSControlName(), providers.CanUseOPENPGPKEY) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeOPENPGPKEY, dns.TypeToString[dns.TypeOPENPGPKEY]))
	}
	if providers.ProviderHasCapability(prvd.DNSControlName(), providers.CanUseSOA) {
		caps = append(caps, fmt.Sprintf("rr-%d-%s", dns.TypeSOA, dns.TypeToString[dns.TypeSOA]))
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
