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

//go:build !nooidc

package config

import (
	"flag"
	"net/url"

	"git.happydns.org/happyDomain/model"
)

var (
	oidcClientID     string
	oidcClientSecret string
	oidcProviderURL  url.URL
)

func init() {
	flag.StringVar(&oidcClientID, "oidc-client-id", oidcClientID, "ClientID for OIDC")
	flag.StringVar(&oidcClientSecret, "oidc-client-secret", oidcClientSecret, "Secret for OIDC")
	flag.Var(&URL{&oidcProviderURL}, "oidc-provider-url", "Base URL of the OpenId Connect service")
}

func ExtendsConfigWithOIDC(o *happydns.Options) error {
	if oidcProviderURL.String() != "" {
		o.OIDCClients = append(o.OIDCClients, happydns.OIDCSettings{
			ClientID:     oidcClientID,
			ClientSecret: oidcClientSecret,
			ProviderURL:  oidcProviderURL,
		})
	}

	return nil
}
