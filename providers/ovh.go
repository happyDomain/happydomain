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
	"flag"

	"github.com/StackExchange/dnscontrol/v4/providers"
	_ "github.com/StackExchange/dnscontrol/v4/providers/ovh"

	"git.happydns.org/happydomain/model"
)

var (
	appKey    string
	appSecret string
)

type OVHAPI struct {
	Endpoint    string `json:"endpoint,omitempty"`
	ConsumerKey string `json:"consumerkey,omitempty" happydomain:"required"`
}

func (s *OVHAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"app-key":        appKey,
		"app-secret-key": appSecret,
		"consumer-key":   s.ConsumerKey,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *OVHAPI) DNSControlName() string {
	return "OVH"
}

func init() {
	flag.StringVar(&appKey, "ovh-application-key", "", "Application Key for using the OVH API")
	flag.StringVar(&appSecret, "ovh-application-secret", "", "Application Secret for using the OVH API")

	RegisterProvider(func() happydns.Provider {
		return &OVHAPI{}
	}, ProviderInfos{
		Name:        "OVH",
		Description: "European hosting provider.",
	})
}
