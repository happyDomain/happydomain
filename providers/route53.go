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
	"github.com/StackExchange/dnscontrol/v4/providers"
	_ "github.com/StackExchange/dnscontrol/v4/providers/route53"

	"git.happydns.org/happydomain/model"
)

type Route53API struct {
	DelegationSet string `json:"delegation_set,omitempty" happydomain:"label=Delegation Set ID,placeholder=xxxxxxxx,description=Optional delegation set ID."`
	KeyId string `json:"key_id,omitempty" happydomain:"label=AWS key,placeholder=xxxxxxxx,required,description=Your AWS key."`
	SecretKey string `json:"secret_key,omitempty" happydomain:"label=AWS secret key,placeholder=xxxxxxxx,required,description=Your AWS secret key."`
	Token string `json:"token,omitempty" happydomain:"label=Token,placeholder=xxxxxxxx,description=Optional STS token."`
}

func (s *Route53API) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"DelegationSet": s.DelegationSet,
		"KeyId": s.KeyId,
		"SecretKey": s.SecretKey,
		"Token": s.Token,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *Route53API) DNSControlName() string {
	return "ROUTE53"
}

func init() {
	RegisterProvider(func() happydns.Provider {
		return &Route53API{}
	}, ProviderInfos{
		Name:        "AWS Route 53",
		Description: "Amazon Web Services (AWS) is a subsidiary of Amazon that provides on-demand cloud computing platforms and APIs. Route 53 is their DNS service.",
	})
}
