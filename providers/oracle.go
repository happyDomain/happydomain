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
	_ "github.com/StackExchange/dnscontrol/v4/providers/oracle"

	"git.happydns.org/happyDomain/model"
)

type OracleAPI struct {
	Compartment string `json:"compartment,omitempty" happydomain:"label=Compartment,placeholder=ORACLE_COMPARTMENT,description=Compartment."`
	Fingerprint string `json:"fingerprint,omitempty" happydomain:"label=Fingerprint,placeholder=ORACLE_FINGERPRINT,required,description=Fingerprint."`
	PrivateKey  string `json:"private_key,omitempty" happydomain:"label=Private hey,placeholder=ORACLE_PRIVATE_KEY,required,description=Private key."`
	Region      string `json:"region,omitempty" happydomain:"label=Region,placeholder=ORACLE_REGION,required,description=Region."`
	TenancyOcid string `json:"tenancy_ocid,omitempty" happydomain:"label=Tenancy OCID,placeholder=ORACLE_TENANCY_OCID,required,description=Tenancy OCID."`
	UserOcid    string `json:"user_ocid,omitempty" happydomain:"label=User OCID,placeholder=ORACLE_USER_OCID,required,description=User OCID."`
}

func (s *OracleAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"compartment":  s.Compartment,
		"fingerprint":  s.Fingerprint,
		"private_key":  s.PrivateKey,
		"region":       s.Region,
		"tenancy_ocid": s.TenancyOcid,
		"user_ocid":    s.UserOcid,
	}

	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *OracleAPI) DNSControlName() string {
	return "ORACLE"
}

func init() {
	RegisterProvider(func() happydns.Provider {
		return &OracleAPI{}
	}, ProviderInfos{
		Name:        "Oracle Cloud",
		Description: "American multinational computer technology corporation headquartered in Austin, Texas",
	})
}
