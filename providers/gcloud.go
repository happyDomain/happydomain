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
	_ "github.com/StackExchange/dnscontrol/v4/providers/gcloud"

	"git.happydns.org/happydomain/model"
)

type GCloudAPI struct {
	ProjectId string `json:"project_id,omitempty" happydomain:"label=Project ID,placeholder=xxxxxxxx,required,description=Project ID."`
	PrivateKey string `json:"private_key,omitempty" happydomain:"label=Private key,placeholder=xxxxxxxx,description=Private key."`
	ClientEmail string `json:"client_email,omitempty" happydomain:"label=Client Email,placeholder=xxxxxxxx,description=Client Email."`
	NameServerSet string `json:"name_server_set,omitempty" happydomain:"label=Name server sets,placeholder=xxxxxxxx,description=Name server sets special permission from your TAM at Google)."`
}

func (s *GCloudAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"project_id": s.ProjectId,
		"private_key": s.PrivateKey,
		"client_email": s.ClientEmail,
		"name_server_set": s.NameServerSet,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *GCloudAPI) DNSControlName() string {
	return "GCLOUD"
}

func init() {
	RegisterProvider(func() happydns.Provider {
		return &GCloudAPI{}
	}, ProviderInfos{
		Name:        "Google Cloud Platform (GCP)",
		Description: "A suite of cloud computing services that runs on the same infrastructure that Google uses internally for its end-user products",
	})
}
