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
	"encoding/base64"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/providers"
	_ "github.com/StackExchange/dnscontrol/v4/providers/axfrddns"
)

type DDNSServer struct {
	Server  string `json:"server,omitempty" happydomain:"label=Server,placeholder=127.0.0.1"`
	KeyName string `json:"keyname,omitempty" happydomain:"label=Key Name,placeholder=ddns,required"`
	KeyAlgo string `json:"algorithm,omitempty" happydomain:"label=Key Algorithm,default=hmac-sha256,choices=hmac-md5;hmac-sha1;hmac-sha256;hmac-sha512,required"`
	KeyBlob []byte `json:"keyblob,omitempty" happydomain:"label=Secret Key,placeholder=a0b1c2d3e4f5==,required,secret"`
}

func (s *DDNSServer) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"master": s.Server,
	}

	if s.Server == "" {
		config["master"] = "127.0.0.1"
	}

	if s.KeyName != "" {
		config["transfer-key"] = strings.Join([]string{s.KeyAlgo, s.KeyName, base64.StdEncoding.EncodeToString(s.KeyBlob)}, ":")
		config["update-key"] = config["transfer-key"]
	}

	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *DDNSServer) DNSControlName() string {
	return "AXFRDDNS"
}

func init() {
	RegisterProvider(func() Provider {
		return &DDNSServer{}
	}, ProviderInfos{
		Name:        "Dynamic DNS",
		Description: "If your zone is hosted on an authoritative name server that support Dynamic DNS (RFC 2136), such as Bind, Knot, ...",
	})
}
