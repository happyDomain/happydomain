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

//go:build !nooidc

package config // import "git.happydns.org/happyDomain/internal/config"

import (
	"testing"

	"git.happydns.org/happyDomain/model"
)

func TestOIDCConfig(t *testing.T) {
	cfg := &happydns.Options{}

	err := parseLine(cfg, "HAPPYDOMAIN_OIDC_CLIENT_ID=test-oidc-1")
	if err != nil {
		t.Fatalf(`parseLine("HAPPYDOMAIN_OIDC_CLIENT_ID=test-oidc-1") => %v`, err.Error())
	}
	if oidcClientID != "test-oidc-1" {
		t.Fatalf(`parseLine("HAPPYDOMAIN_OIDC_CLIENT_ID=test-oidc-1") = %q, want "test-oidc-1"`, oidcClientID)
	}

	err = parseLine(cfg, "HAPPYDOMAIN_OIDC_CLIENT_SECRET=s3cret$")
	if err != nil {
		t.Fatalf(`parseLine("HAPPYDOMAIN_OIDC_CLIENT_SECRET=s3cret$") => %v`, err.Error())
	}
	if oidcClientSecret != "s3cret$" {
		t.Fatalf(`parseLine("HAPPYDOMAIN_OIDC_CLIENT_SECRET=s3cret$") = %q, want "s3cret$"`, oidcClientSecret)
	}

	if oidcProviderURL.String() != "" {
		t.Fatalf(`before parseLine("HAPPYDOMAIN_OIDC_PROVIDER_URL") = %q, want ""`, oidcProviderURL.String())
	}

	err = parseLine(cfg, "HAPPYDOMAIN_OIDC_PROVIDER_URL=https://localhost:12345/secret")
	if err != nil {
		t.Fatalf(`parseLine("HAPPYDOMAIN_OIDC_PROVIDER_URL=https://localhost:12345/secret") => %v`, err.Error())
	}
	if oidcProviderURL.String() != "https://localhost:12345/secret" {
		t.Fatalf(`parseLine("HAPPYDOMAIN_OIDC_PROVIDER_URL=https://localhost:12345/secret") = %q, want "https://localhost:12345/secret"`, cfg.Bind)
	}

	// Test extended config
	err = ExtendsConfigWithOIDC(cfg)
	if err != nil {
		t.Fatalf(`ExtendsConfigWithOIDC(cfg) => %v`, err.Error())
	}

	if len(cfg.OIDCClients) != 1 {
		t.Fatalf(`len(cfg.OIDCClients) == %d, should be 1`, len(cfg.OIDCClients))
	}

	if cfg.OIDCClients[0].ClientID != "test-oidc-1" {
		t.Fatalf(`cfg.OIDCClients[0].ClientID == %q, should be test-oidc-1`, cfg.OIDCClients[0].ClientID)
	}
	if cfg.OIDCClients[0].ClientSecret != "s3cret$" {
		t.Fatalf(`cfg.OIDCClients[0].ClientSecret == %q, should be test-oidc-1`, cfg.OIDCClients[0].ClientSecret)
	}
	if cfg.OIDCClients[0].ProviderURL.String() != "https://localhost:12345/secret" {
		t.Fatalf(`cfg.OIDCClients[0].ProviderURL == %q, should be https://localhost:12345/secret`, cfg.OIDCClients[0].ProviderURL.String())
	}
}
