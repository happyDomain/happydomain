// Copyright or © or Copr. happyDNS (2023)
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

package providers // import "git.happydns.org/happyDomain/providers"

import (
	"github.com/StackExchange/dnscontrol/v4/providers"
	_ "github.com/StackExchange/dnscontrol/v4/providers/azure_private_dns"

	"git.happydns.org/happyDomain/model"
)

type AzurePrivateDnsAPI struct {
	SubscriptionID string `json:"SubscriptionID,omitempty" happydomain:"label=Subscription ID,placeholder=xxxxxxxx,required,description=Your Azure Client Subscription ID."`
	ResourceGroup  string `json:"ResourceGroup,omitempty" happydomain:"label=Resource Group,placeholder=xxxxxxxx,required,description=Your Azure Resource Group."`
	TenantID       string `json:"TenantID,omitempty" happydomain:"label=Tenant ID,placeholder=xxxxxxxx,description=Your Azure Tenant ID."`
	ClientID       string `json:"ClientID,omitempty" happydomain:"label=Client ID,placeholder=xxxxxxxx,description=Your Azure Client ID."`
	ClientSecret   string `json:"ClientSecret,omitempty" happydomain:"label=Client Secret,placeholder=xxxxxxxx,description=Your Azure Client Secret."`
}

func (s *AzurePrivateDnsAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"SubscriptionID": s.SubscriptionID,
		"ResourceGroup":  s.ResourceGroup,
		"TenantID":       s.TenantID,
		"ClientID":       s.ClientID,
		"ClientSecret":   s.ClientSecret,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *AzurePrivateDnsAPI) DNSControlName() string {
	return "AZURE_PRIVATE_DNS"
}

func init() {
	RegisterProvider(func() happydns.Provider {
		return &AzurePrivateDnsAPI{}
	}, ProviderInfos{
		Name:        "Azure Private DNS",
		Description: "Exclusively to manage Private DNS zones. Use Azure DNS for public zones.",
	})
}
