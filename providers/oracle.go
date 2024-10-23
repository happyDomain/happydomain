// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
// Authors: David Dernoncourt, et al.
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
	"github.com/StackExchange/dnscontrol/v4/providers"
	_ "github.com/StackExchange/dnscontrol/v4/providers/oracle"
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
	RegisterProvider(func() Provider {
		return &OracleAPI{}
	}, ProviderInfos{
		Name:        "Oracle Cloud",
		Description: "American multinational computer technology corporation headquartered in Austin, Texas",
	})
}
