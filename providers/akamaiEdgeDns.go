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
	_ "github.com/StackExchange/dnscontrol/v4/providers/akamaiedgedns"

	"git.happydns.org/happyDomain/model"
)

type AkamaiEdgeDnsAPI struct {
	ClientSecret string `json:"clientsecret,omitempty" happydomain:"label=Client Secret,placeholder=xxxxxxxx,required,description=Your Akamai Client Secret (You must enable API-Access for your account)."`
	Host         string `json:"host,omitempty" happydomain:"label=Host,placeholder=akaa-xxxxxxxxxxx.xxxx.akamaiapis.net,required,description=Your Akamai Host."`
	AccessToken  string `json:"accesstoken,omitempty" happydomain:"label=Access Token,placeholder=akaa-xxxxxxxxxxx,description=Your Akamai Access Token."`
	ClientToken  string `json:"clienttoken,omitempty" happydomain:"label=Client Token,placeholder=akaa-xxxxxxxxxxx,description=Your Akamai Client Token"`
	ContractId   string `json:"contractid,omitempty" happydomain:"label=Contract ID,placeholder=X-XXXX,description=Your Akamai Contract ID."`
	GroupId      string `json:"groupId,omitempty" happydomain:"label=Group ID,placeholder=NNNNNN,description=Your Akamai Group ID."`
}

func (s *AkamaiEdgeDnsAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"client_secret": s.ClientSecret,
		"host":          s.Host,
		"access_token":  s.AccessToken,
		"client_token":  s.ClientToken,
		"contract_id":   s.ContractId,
		"group_id":      s.GroupId,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *AkamaiEdgeDnsAPI) DNSControlName() string {
	return "AKAMAIEDGEDNS"
}

func init() {
	RegisterProvider(func() happydns.Provider {
		return &AkamaiEdgeDnsAPI{}
	}, ProviderInfos{
		Name:        "Akamai Edge DNS",
		Description: "American content delivery network and cloud service company - https://www.akamai.com",
	})
}
