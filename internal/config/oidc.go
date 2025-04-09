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
	"context"
	"flag"
	"net/url"
	"path"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

var (
	OIDCClientID     string
	oidcClientSecret string
	OIDCProviderURL  string
)

func init() {
	flag.StringVar(&OIDCClientID, "oidc-client-id", OIDCClientID, "ClientID for OIDC")
	flag.StringVar(&oidcClientSecret, "oidc-client-secret", oidcClientSecret, "Secret for OIDC")
	flag.StringVar(&OIDCProviderURL, "oidc-provider-url", OIDCProviderURL, "Base URL of the OpenId Connect service")
}

func (o *Options) GetAuthURL() *url.URL {
	redirecturl := *o.ExternalURL.URL
	redirecturl.Path = path.Join(redirecturl.Path, o.baseURL, "auth", "callback")
	return &redirecturl
}

func (o *Options) GetOIDCProvider(ctx context.Context) (*oidc.Provider, error) {
	return oidc.NewProvider(ctx, strings.TrimSuffix(OIDCProviderURL, "/.well-known/openid-configuration"))
}

func (o *Options) GetOIDCProviderURL() string {
	return OIDCProviderURL
}

func (o *Options) GetOAuth2Config(provider *oidc.Provider) *oauth2.Config {
	oauth2Config := oauth2.Config{
		ClientID:     OIDCClientID,
		ClientSecret: oidcClientSecret,
		RedirectURL:  o.GetAuthURL().String(),
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return &oauth2Config
}
