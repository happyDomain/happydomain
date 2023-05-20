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
	_ "github.com/StackExchange/dnscontrol/v4/providers/autodns"

	"git.happydns.org/happydomain/model"
)

type AutoDNSAPI struct {
	Username string `json:"username,omitempty" happydomain:"label=Username,placeholder=autodns.service-account@example.com,required,description=Your AutoDNS user name."`
	Password string `json:"password,omitempty" happydomain:"label=Password,placeholder=xxxxxxxx,required,description=Your AutoDNS password."`
	Context string `json:"context,omitempty" happydomain:"label=Context,placeholder=33004,description=Your AutoDNS context."`
}

func (s *AutoDNSAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"username": s.Username,
		"password": s.Password,
		"context": s.Context,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *AutoDNSAPI) DNSControlName() string {
	return "AUTODNS"
}

func init() {
	RegisterProvider(func() happydns.Provider {
		return &AutoDNSAPI{}
	}, ProviderInfos{
		Name:        "AutoDNS / InterNetX",
		Description: "German hosting provider.",
	})
}
