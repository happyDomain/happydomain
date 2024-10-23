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

package config // import "git.happydns.org/happyDomain/config"

import (
	"net/url"
	"testing"
)

func TestParseLine(t *testing.T) {
	cfg := Options{}
	cfg.declareFlags()

	err := cfg.parseLine("HAPPYDOMAIN_BIND=:8080")
	if err != nil {
		t.Fatalf(`parseLine("BIND=:8080") => %v`, err.Error())
	}
	if cfg.Bind != ":8080" {
		t.Fatalf(`parseLine("BIND=:8080") = %q, want ":8080"`, cfg.Bind)
	}

	err = cfg.parseLine("BASEURL=/base")
	if err != nil {
		t.Fatalf(`parseLine("BASEURL=/base") => %v`, err.Error())
	}
	if cfg.BaseURL != "/base" {
		t.Fatalf(`parseLine("BASEURL=/base") = %q, want "/base"`, cfg.BaseURL)
	}

	cfg.parseLine("EXTERNALURL=https://happydomain.org")
	if cfg.ExternalURL.String() != "https://happydomain.org" {
		t.Fatalf(`parseLine("EXTERNAL_URL=https://happydomain.org") = %q, want "https://happydomain.org"`, cfg.ExternalURL)
	}

	cfg.parseLine("DEFAULT-NS=42.42.42.42:5353")
	if cfg.DefaultNameServer != "42.42.42.42:5353" {
		t.Fatalf(`parseLine("DEFAULT-NS=42.42.42.42:5353") = %q, want "42.42.42.42:5353"`, cfg.DefaultNameServer)
	}

	cfg.parseLine("DEFAULT_NS=42.42.42.42:3535")
	if cfg.DefaultNameServer != "42.42.42.42:3535" {
		t.Fatalf(`parseLine("DEFAULT_NS=42.42.42.42:3535") = %q, want "42.42.42.42:3535"`, cfg.DefaultNameServer)
	}

	err = cfg.parseLine("NO_AUTH=true")
	if err != nil {
		t.Fatalf(`parseLine("NO_AUTH=true") => %v`, err.Error())
	}
	if !cfg.NoAuth {
		t.Fatalf(`parseLine("NO_AUTH=true") = %v, want true`, cfg.NoAuth)
	}
}

func TestGetBaseURL(t *testing.T) {
	u, _ := url.Parse("http://localhost:8081")

	cfg := Options{
		ExternalURL: URL{URL: u},
	}

	builded_url := cfg.GetBaseURL()
	if builded_url != "http://localhost:8081" {
		t.Fatalf(`GetBaseURL() = %q, want "http://localhost:8081"`, builded_url)
	}

	cfg.BaseURL = "/base"

	builded_url = cfg.GetBaseURL()
	if builded_url != "http://localhost:8081/base" {
		t.Fatalf(`GetBaseURL() = %q, want "http://localhost:8081/base"`, builded_url)
	}
}
