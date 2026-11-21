// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
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
	_ "github.com/DNSControl/dnscontrol/v4/providers/gandiv5"

	"git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/model"
)

// GandiAPI authenticates against the Gandi API either with a Personal Access
// Token or with a (deprecated) API key, as chosen in the first step of its
// settings form (see gandiv5_settings.go).
type GandiAPI struct {
	AuthMethod string          `json:"auth_method,omitempty" happydomain:"label=Authentication method,choices=Personal Access Token;API key (legacy)"`
	Token      happydns.Secret `json:"token,omitzero" happydomain:"label=Personal Access Token,placeholder=xxxxxxxxxx,secret,description=Create a Personal Access Token in your account settings at https://admin.gandi.net/organizations/account/pat (permission \"Manage domain name technical configurations\" is required)"`
	APIKey     happydns.Secret `json:"api_key,omitzero" happydomain:"label=API Key,placeholder=xxxxxxxxxx,secret,description=Get your API Key in the Security section at https://account.gandi.net/"`
	SharingID  string          `json:"sharing_id,omitempty" happydomain:"label=Sharing ID,placeholder=xxxxxxxxxx,description=Organization sharing ID (required if member of multiple organizations)"`
}

func (s *GandiAPI) DNSControlName() string {
	return "GANDI_V5"
}

func (s *GandiAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *GandiAPI) ToDNSControlConfig() (map[string]string, error) {
	if s.authMethod() == gandiAuthAPIKey {
		return map[string]string{
			"apikey":     s.APIKey.Reveal(),
			"sharing_id": s.SharingID,
		}, nil
	}

	return map[string]string{
		"token": s.Token.Reveal(),
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &GandiAPI{}
	}, happydns.ProviderInfos{
		Name:        "Gandi",
		Description: "Domain registrar and hosting provider with LiveDNS service.",
		Website:     "https://www.gandi.net",
	}, providerReg.RegisterProvider)
}
