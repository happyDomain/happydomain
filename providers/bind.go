// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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
	"flag"

	_ "github.com/StackExchange/dnscontrol/v4/providers/bind"

	"git.happydns.org/happyDomain/adapters"
	"git.happydns.org/happyDomain/model"
)

type BindServer struct {
	Directory  string `json:"directory,omitempty" happydomain:"label=Directory,placeholder=/etc/named/zones/,required,description=Local directory on the same host running happyDomain, containing your zones"`
	Fileformat string `json:"fileformat,omitempty" happydomain:"label=File format,placeholder=%U.zone,description=See format at https://docs.dnscontrol.org/service-providers/providers/bind#filenameformat"`
}

func (s *BindServer) DNSControlName() string {
	return "BIND"
}

func (s *BindServer) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *BindServer) ToDNSControlConfig() (map[string]string, error) {
	config := map[string]string{
		"directory": s.Directory,
	}

	if s.Fileformat != "" {
		config["filenameformat"] = s.Fileformat
	}

	return config, nil
}

func init() {
	flag.BoolFunc("with-bind-provider", "Enable the BIND provider (not suitable for cloud/shared instance as it'll access the local file system)", func(s string) error {
		adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
			return &BindServer{}
		}, happydns.ProviderInfos{
			Name:        "Bind files/RFC 1035",
			Description: "Use zone files saved in the RFC 1035 format.",
		}, RegisterProvider)
		return nil
	})
}
