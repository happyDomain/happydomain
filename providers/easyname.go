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
	"github.com/StackExchange/dnscontrol/v3/providers"
	_ "github.com/StackExchange/dnscontrol/v3/providers/easyname"

	"git.happydns.org/happydomain/model"
)

type EasynameAPI struct {
	ApiKey string `json:"username,omitempty" happydomain:"label=API Key,placeholder=xxxxxxxx,required,description=Your Easyname API key (You must enable API-Access for your account)."`
	AuthSalt string `json:"password,omitempty" happydomain:"label=API Authentication Salt,placeholder=xxxxxxxx,required,description=Your Easyname API Authentication Salt."`
	Signsalt string `json:"context,omitempty" happydomain:"label=API Signing Salt,placeholder=xxxxxxxx,description=Your Easyname API Signing Salt."`
	Email string `json:"context,omitempty" happydomain:"label=Email,placeholder=xxxxxxxx,description=Your Easyname e-mail."`
	UserId string `json:"context,omitempty" happydomain:"label=User ID,placeholder=xxxxxxxx,description=Your Easyname User ID."`
}

func (s *EasynameAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"apikey": s.ApiKey,
		"authsalt": s.AuthSalt,
		"signsalt": s.Signsalt,
		"email": s.Email,
		"userid": s.UserId,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *EasynameAPI) DNSControlName() string {
	return "EASYNAME"
}

func init() {
	RegisterProvider(func() happydns.Provider {
		return &EasynameAPI{}
	}, ProviderInfos{
		Name:        "Easyname GmbH",
		Description: "Austrian hosting provider based in Vienna.",
	})
}
