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
	_ "github.com/StackExchange/dnscontrol/v4/providers/gcloud"
)

type GCloudAPI struct {
	ProjectId     string `json:"project_id,omitempty" happydomain:"label=Project ID,placeholder=xxxxxxxx,required,description=Project ID."`
	PrivateKey    string `json:"private_key,omitempty" happydomain:"label=Private key,placeholder=xxxxxxxx,description=Private key."`
	ClientEmail   string `json:"client_email,omitempty" happydomain:"label=Client Email,placeholder=xxxxxxxx,description=Client Email."`
	NameServerSet string `json:"name_server_set,omitempty" happydomain:"label=Name server sets,placeholder=xxxxxxxx,description=Name server sets special permission from your TAM at Google)."`
}

func (s *GCloudAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"project_id":      s.ProjectId,
		"private_key":     s.PrivateKey,
		"client_email":    s.ClientEmail,
		"name_server_set": s.NameServerSet,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *GCloudAPI) DNSControlName() string {
	return "GCLOUD"
}

func init() {
	RegisterProvider(func() Provider {
		return &GCloudAPI{}
	}, ProviderInfos{
		Name:        "Google Cloud Platform (GCP)",
		Description: "A suite of cloud computing services that runs on the same infrastructure that Google uses internally for its end-user products",
	})
}
