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
	_ "github.com/StackExchange/dnscontrol/v4/providers/gandiv5"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type GandiAPI struct {
	APIKey    string `json:"api_key,omitempty" happydomain:"label=API Key,placeholder=xxxxxxxxxx,required,secret,description=Get your API Key in the Security section at https://account.gandi.net/"`
	SharingID string `json:"sharing_id,omitempty" happydomain:"label=Sharing ID,placeholder=xxxxxxxxxx,description=Organization sharing ID (required if member of multiple organizations)"`
}

func (s *GandiAPI) DNSControlName() string {
	return "GANDI_V5"
}

func (s *GandiAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *GandiAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"apikey":     s.APIKey,
		"sharing_id": s.SharingID,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &GandiAPI{}
	}, happydns.ProviderInfos{
		Name:        "Gandi",
		Description: "Domain registrar and hosting provider with LiveDNS service.",
	}, RegisterProvider)
}
