// Copyright or © or Copr. happyDNS (2023)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

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

func TestBuildURL(t *testing.T) {
	u, _ := url.Parse("http://localhost:8081")

	cfg := Options{
		ExternalURL: URL{URL: u},
	}

	builded_url := cfg.BuildURL("/test")
	if builded_url != "http://localhost:8081/test" {
		t.Fatalf(`BuildURL("/test") = %q, want "http://localhost:8081/test"`, builded_url)
	}

	builded_url = cfg.BuildURL("/test%s")
	if builded_url != "http://localhost:8081/test%s" {
		t.Fatalf(`BuildURL("/test") = %q, want "http://localhost:8081/test%%s"`, builded_url)
	}

	cfg.BaseURL = "/base"

	builded_url = cfg.BuildURL("/test")
	if builded_url != "http://localhost:8081/base/test" {
		t.Fatalf(`BuildURL("/test") = %q, want "http://localhost:8081/base/test"`, builded_url)
	}
}

func TestBuildURL_noescape(t *testing.T) {
	u, _ := url.Parse("http://localhost:8081")

	cfg := Options{
		ExternalURL: URL{URL: u},
	}

	builded_url := cfg.BuildURL_noescape("/test")
	if builded_url != "http://localhost:8081/test" {
		t.Fatalf(`BuildURL_noescape("/test") = %q, want "http://localhost:8081/test"`, builded_url)
	}

	builded_url = cfg.BuildURL_noescape("/test%s", "/api")
	if builded_url != "http://localhost:8081/test/api" {
		t.Fatalf(`BuildURL_noescape("/test") = %q, want "http://localhost:8081/test/api"`, builded_url)
	}

	cfg.BaseURL = "/base"

	builded_url = cfg.BuildURL_noescape("/test%s", "?test=foo")
	if builded_url != "http://localhost:8081/base/test?test=foo" {
		t.Fatalf(`BuildURL_noescape("/test") = %q, want "http://localhost:8081/base/test?test=foo"`, builded_url) //
	}
}
