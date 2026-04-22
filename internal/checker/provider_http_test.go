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

package checker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sdk "git.happydns.org/checker-sdk-go/checker"
	"git.happydns.org/happyDomain/model"
)

func TestHTTPObservationProvider_Key(t *testing.T) {
	p := NewHTTPObservationProvider("my_key", "http://example.com")
	if got := p.Key(); got != "my_key" {
		t.Errorf("Key() = %q, want %q", got, "my_key")
	}
}

func TestHTTPObservationProvider_TrailingSlashTrimmed(t *testing.T) {
	p := NewHTTPObservationProvider("k", "http://example.com/")
	if p.endpoint != "http://example.com" {
		t.Errorf("endpoint = %q, want trailing slash trimmed", p.endpoint)
	}
}

func TestHTTPObservationProvider_CollectSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/collect" {
			t.Errorf("expected /collect, got %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}

		// Verify request body is well-formed.
		var req happydns.ExternalCollectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.Key != "test_obs" {
			t.Errorf("request Key = %q, want %q", req.Key, "test_obs")
		}
		if v, ok := req.Options["foo"]; !ok || v != "bar" {
			t.Errorf("request Options[foo] = %v, want %q", v, "bar")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(happydns.ExternalCollectResponse{
			Data: json.RawMessage(`{"value":42}`),
		})
	}))
	defer srv.Close()

	p := NewHTTPObservationProvider("test_obs", srv.URL)
	opts := happydns.CheckerOptions{"foo": "bar"}

	result, err := p.Collect(context.Background(), opts)
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	raw, ok := result.(json.RawMessage)
	if !ok {
		t.Fatalf("expected json.RawMessage, got %T", result)
	}

	var data map[string]int
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["value"] != 42 {
		t.Errorf("value = %d, want 42", data["value"])
	}
}

func TestHTTPObservationProvider_CollectRemoteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(happydns.ExternalCollectResponse{
			Error: "something went wrong",
		})
	}))
	defer srv.Close()

	p := NewHTTPObservationProvider("k", srv.URL)
	_, err := p.Collect(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for remote error response")
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Errorf("error = %q, want it to contain remote error message", err)
	}
}

func TestHTTPObservationProvider_CollectEmptyData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(happydns.ExternalCollectResponse{})
	}))
	defer srv.Close()

	p := NewHTTPObservationProvider("k", srv.URL)
	_, err := p.Collect(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty data response")
	}
	if !strings.Contains(err.Error(), "empty data") {
		t.Errorf("error = %q, want it to mention empty data", err)
	}
}

func TestHTTPObservationProvider_CollectNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal failure", http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewHTTPObservationProvider("k", srv.URL)
	_, err := p.Collect(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %q, want it to contain status code 500", err)
	}
	if !strings.Contains(err.Error(), "internal failure") {
		t.Errorf("error = %q, want it to contain response body excerpt", err)
	}
}

func TestHTTPObservationProvider_CollectInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, "not json")
	}))
	defer srv.Close()

	p := NewHTTPObservationProvider("k", srv.URL)
	_, err := p.Collect(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("error = %q, want it to mention decode failure", err)
	}
}

func TestHTTPObservationProvider_CollectContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the request context is cancelled.
		<-r.Context().Done()
	}))
	defer srv.Close()

	p := NewHTTPObservationProvider("k", srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := p.Collect(ctx, nil)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestHTTPObservationProvider_CollectConnectionRefused(t *testing.T) {
	// Use a server that is immediately closed to simulate connection refused.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	endpoint := srv.URL
	srv.Close()

	p := NewHTTPObservationProvider("k", endpoint)
	_, err := p.Collect(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("error = %q, want it to mention request failure", err)
	}
}

func TestHTTPObservationProvider_CollectForwardsEntries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(happydns.ExternalCollectResponse{
			Data: json.RawMessage(`{"ok":true}`),
			Entries: []happydns.DiscoveryEntry{
				{Type: "tls.endpoint.v1", Ref: "a.example.com:25"},
				{Type: "tls.endpoint.v1", Ref: "a.example.com:465"},
			},
		})
	}))
	defer srv.Close()

	p := NewHTTPObservationProvider("k", srv.URL)
	if _, err := p.Collect(context.Background(), nil); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	entries, err := p.DiscoverEntries(nil)
	if err != nil {
		t.Fatalf("DiscoverEntries: %v", err)
	}
	if len(entries) != 2 || entries[1].Ref != "a.example.com:465" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestHTTPObservationProvider_GetHTMLReportForwardsRelated(t *testing.T) {
	var gotReq happydns.ExternalReportRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/report" {
			t.Errorf("path = %q, want /report", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode: %v", err)
		}
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, "<html>ok</html>")
	}))
	defer srv.Close()

	p := NewHTTPObservationProvider("tls", srv.URL)
	related := map[happydns.ObservationKey][]happydns.RelatedObservation{
		"tls": {
			{CheckerID: "xmpp", Key: "tls", Data: json.RawMessage(`{"v":1}`), Ref: "host:443", CollectedAt: time.Unix(42, 0).UTC()},
		},
	}
	rc := sdk.NewReportContext(json.RawMessage(`{"primary":true}`), related)

	html, err := p.GetHTMLReport(rc)
	if err != nil {
		t.Fatalf("GetHTMLReport: %v", err)
	}
	if html != "<html>ok</html>" {
		t.Fatalf("html = %q", html)
	}
	if gotReq.Key != "tls" {
		t.Errorf("Key = %q, want tls", gotReq.Key)
	}
	if len(gotReq.Related["tls"]) != 1 || gotReq.Related["tls"][0].Ref != "host:443" {
		t.Errorf("Related not forwarded: %+v", gotReq.Related)
	}
}

func TestHTTPObservationProvider_GetHTMLReportSurfaces501(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not implemented", http.StatusNotImplemented)
	}))
	defer srv.Close()

	p := NewHTTPObservationProvider("k", srv.URL)
	_, err := p.GetHTMLReport(sdk.StaticReportContext(json.RawMessage(`{}`)))
	if err == nil || !strings.Contains(err.Error(), "does not support") {
		t.Fatalf("want 'does not support' error, got: %v", err)
	}
}

func TestHTTPObservationProvider_IntegrationWithObservationContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(happydns.ExternalCollectResponse{
			Data: json.RawMessage(`{"temp":23.5}`),
		})
	}))
	defer srv.Close()

	key := happydns.ObservationKey("http_test_obs")
	p := NewHTTPObservationProvider(key, srv.URL)

	oc := NewObservationContext(happydns.CheckTarget{}, happydns.CheckerOptions{}, nil, 0)
	oc.SetProviderOverride(key, p)

	var dest map[string]float64
	if err := oc.Get(context.Background(), key, &dest); err != nil {
		t.Fatalf("ObservationContext.Get() returned error: %v", err)
	}
	if dest["temp"] != 23.5 {
		t.Errorf("temp = %v, want 23.5", dest["temp"])
	}

	// Second call should use the cached value, not hit the server again.
	var dest2 map[string]float64
	if err := oc.Get(context.Background(), key, &dest2); err != nil {
		t.Fatalf("second Get() returned error: %v", err)
	}
	if dest2["temp"] != 23.5 {
		t.Errorf("cached temp = %v, want 23.5", dest2["temp"])
	}
}
