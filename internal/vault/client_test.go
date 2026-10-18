// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

package vault

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeVault answers as Vault does, for the token "good" only.
func fakeVault(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/secret/data/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vault-Token") != "good" {
			http.Error(w, `{"errors":["permission denied"]}`, http.StatusForbidden)
			return
		}
		path := strings.TrimPrefix(r.URL.EscapedPath(), "/v1/secret/data/")
		switch {
		case path == "dns/ovh" && r.URL.Query().Get("version") == "":
			w.Write([]byte(`{"data":{"data":{"app_secret":"s3cr3t","port":53},"metadata":{"version":2}}}`))
		case path == "dns/ovh" && r.URL.Query().Get("version") == "1":
			w.Write([]byte(`{"data":{"data":{"app_secret":"old"},"metadata":{"version":1}}}`))
		case path == "a%20b/c":
			w.Write([]byte(`{"data":{"data":{"k":"spaced"}}}`))
		case path == "deleted":
			// A deleted version answers 404 with its metadata.
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"data":{"data":null,"metadata":{"deletion_time":"2026-01-01T00:00:00Z"}}}`))
		case path == "huge":
			w.Write([]byte(`{"data":{"data":{"k":"` + strings.Repeat("x", maxResponse) + `"}}}`))
		default:
			http.Error(w, `{"errors":[]}`, http.StatusNotFound)
		}
	})
	mux.HandleFunc("POST /v1/auth/approle/login", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["role_id"] != "role" || body["secret_id"] != "sid" {
			http.Error(w, `{"errors":["invalid role or secret ID"]}`, http.StatusBadRequest)
			return
		}
		w.Write([]byte(`{"auth":{"client_token":"good","lease_duration":3600,"renewable":true}}`))
	})
	mux.HandleFunc("POST /v1/auth/token/renew-self", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vault-Token") == "root" {
			http.Error(w, `{"errors":["lease is not renewable"]}`, http.StatusBadRequest)
			return
		}
		if r.Header.Get("X-Vault-Token") != "good" {
			http.Error(w, `{"errors":["permission denied"]}`, http.StatusForbidden)
			return
		}
		w.Write([]byte(`{"auth":{"client_token":"good","lease_duration":7200,"renewable":true}}`))
	})
	mux.HandleFunc("GET /v1/auth/token/lookup-self", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vault-Token") != "good" {
			http.Error(w, `{"errors":["permission denied"]}`, http.StatusForbidden)
			return
		}
		w.Write([]byte(`{"data":{"ttl":3500,"renewable":true,"period":3600,"num_uses":0,"type":"service"}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := New(srv.URL, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestReadKV2(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t, fakeVault(t))

	data, err := c.ReadKV2(ctx, "good", "secret", "dns/ovh", 0)
	if err != nil || data["app_secret"] != "s3cr3t" {
		t.Errorf("ReadKV2 = %v, %v", data, err)
	}
	if data, err := c.ReadKV2(ctx, "good", "secret", "dns/ovh", 1); err != nil || data["app_secret"] != "old" {
		t.Errorf("ReadKV2(version 1) = %v, %v", data, err)
	}
	if data, err := c.ReadKV2(ctx, "good", "secret", "a b/c", 0); err != nil || data["k"] != "spaced" {
		t.Errorf("ReadKV2(escaped path) = %v, %v", data, err)
	}

	for name, path := range map[string]string{"missing": "nope", "deleted": "deleted"} {
		if _, err := c.ReadKV2(ctx, "good", "secret", path, 0); !errors.Is(err, ErrNotFound) {
			t.Errorf("ReadKV2(%s) = %v, want ErrNotFound", name, err)
		}
	}
	if _, err := c.ReadKV2(ctx, "bad", "secret", "dns/ovh", 0); !errors.Is(err, ErrForbidden) {
		t.Errorf("ReadKV2(bad token) = %v, want ErrForbidden", err)
	}
	if _, err := c.ReadKV2(ctx, "good", "secret", "huge", 0); err == nil {
		t.Error("an unbounded response was read")
	}
}

// What Vault answers is never repeated in an error: it may name the path,
// which may be a secret entered by mistake.
func TestErrorsDoNotRepeatTheAnswer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":["no handler for route secret/data/pa55w0rd"]}`, http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	_, err := c.ReadKV2(context.Background(), "good", "secret", "pa55w0rd", 0)
	if err == nil || strings.Contains(err.Error(), "pa55w0rd") {
		t.Errorf("ReadKV2 = %v, want an error not naming the path", err)
	}
}

func TestLoginAppRoleAndRenew(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t, fakeVault(t))

	tok, err := c.LoginAppRole(ctx, "approle", "role", "sid")
	if err != nil || tok.Token != "good" || tok.LeaseSeconds != 3600 || !tok.Renewable {
		t.Fatalf("LoginAppRole = %+v, %v", tok, err)
	}
	if _, err := c.LoginAppRole(ctx, "approle", "role", "wrong"); !errors.Is(err, ErrForbidden) {
		t.Errorf("LoginAppRole(wrong) = %v, want ErrForbidden", err)
	}

	renewed, err := c.RenewSelf(ctx, tok.Token)
	if err != nil || renewed.LeaseSeconds != 7200 {
		t.Errorf("RenewSelf = %+v, %v", renewed, err)
	}
	if _, err := c.RenewSelf(ctx, "root"); !errors.Is(err, ErrNotRenewable) {
		t.Errorf("RenewSelf(not renewable) = %v, want ErrNotRenewable", err)
	}
	if _, err := c.RenewSelf(ctx, "bad"); !errors.Is(err, ErrForbidden) {
		t.Errorf("RenewSelf(bad) = %v, want ErrForbidden", err)
	}
	info, err := c.LookupSelf(ctx, "good")
	if err != nil || info != (TokenInfo{TTL: 3500, Renewable: true, Period: 3600, Type: "service"}) {
		t.Errorf("LookupSelf = %+v, %v", info, err)
	}
	if _, err := c.LookupSelf(ctx, "bad"); !errors.Is(err, ErrForbidden) {
		t.Errorf("LookupSelf(bad) = %v, want ErrForbidden", err)
	}
}

func TestNamespace(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Vault-Namespace")
		io.WriteString(w, `{"data":{"ttl":0}}`)
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	c.Namespace = "team"
	if _, err := c.LookupSelf(context.Background(), "good"); err != nil || got != "team" {
		t.Errorf("LookupSelf = %v, namespace sent %q", err, got)
	}
}

func TestNewRefusesWhatIsNotAnAddress(t *testing.T) {
	for _, addr := range []string{"", "vault:8200", "ftp://vault", "https://", "https://vault/?x=1", "https://user:pw@vault"} {
		if _, err := New(addr, http.DefaultClient); err == nil {
			t.Errorf("New(%q) succeeded", addr)
		}
	}
	if c, err := New("https://vault.example:8200/", http.DefaultClient); err != nil || c.Address() != "https://vault.example:8200" {
		t.Errorf("New = %v, %v", c, err)
	}
}

// An unreachable server is reported without the URL, which holds the path.
func TestUnreachableDoesNotRepeatThePath(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	c := newTestClient(t, srv)
	srv.Close()
	_, err := c.ReadKV2(context.Background(), "good", "secret", "pa55w0rd", 0)
	if err == nil || strings.Contains(err.Error(), "pa55w0rd") {
		t.Errorf("ReadKV2 = %v, want an error not naming the path", err)
	}
}

// Vault may be served below a path, as behind a reverse proxy.
func TestAddressWithPath(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.EscapedPath()
		io.WriteString(w, `{"data":{"data":{"k":"v"}}}`)
	}))
	defer srv.Close()
	c, err := New(srv.URL+"/vault/", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ReadKV2(context.Background(), "good", "kv", "a b/c", 0); err != nil || got != "/vault/v1/kv/data/a%20b/c" {
		t.Errorf("ReadKV2 = %v, path sent %q", err, got)
	}
}

// An answer missing what was asked, or not decoding, is no token nor secret.
func TestEmptyOrBrokenAnswers(t *testing.T) {
	ctx := context.Background()
	for name, answer := range map[string]string{"empty": `{}`, "broken": `{"data":`} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, answer)
		}))
		c := newTestClient(t, srv)
		if data, err := c.ReadKV2(ctx, "good", "secret", "x", 0); err == nil {
			t.Errorf("%s: ReadKV2 = %v, want an error", name, data)
		}
		if tok, err := c.LoginAppRole(ctx, "approle", "role", "sid"); err == nil {
			t.Errorf("%s: LoginAppRole = %+v, want an error", name, tok)
		}
		if tok, err := c.RenewSelf(ctx, "good"); err == nil {
			t.Errorf("%s: RenewSelf = %+v, want an error", name, tok)
		}
		if info, err := c.LookupSelf(ctx, "good"); err == nil {
			t.Errorf("%s: LookupSelf = %+v, want an error", name, info)
		}
		srv.Close()
	}

	// A KV v2 answer without data is a missing secret, not an empty one.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":{"data":null}}`)
	}))
	defer srv.Close()
	if _, err := newTestClient(t, srv).ReadKV2(ctx, "good", "secret", "x", 0); !errors.Is(err, ErrNotFound) {
		t.Errorf("ReadKV2(null data) = %v, want ErrNotFound", err)
	}
}
