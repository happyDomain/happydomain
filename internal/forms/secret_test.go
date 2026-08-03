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

package forms

import (
	"bytes"
	"encoding/json"
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

// Mirrors the shapes found in providers/: a plain credential, a blob one
// (providers/axfrddns.go KeyBlob), an embedded common part and a nested struct.
type commonAuth struct {
	Account string `json:"account"`
	AppKey  string `json:"appkey" happydomain:"label=Application Key,required,secret"`
}

type nestedAuth struct {
	Endpoint string `json:"endpoint"`
	Token    string `json:"token" happydomain:"label=Token,secret"`
}

type testProvider struct {
	commonAuth
	Host    string      `json:"host" happydomain:"label=Host"`
	Secret  string      `json:"secret" happydomain:"label=Secret,required,secret"`
	KeyBlob []byte      `json:"keyblob,omitempty" happydomain:"label=Secret Key,secret"`
	Unset   string      `json:"unset,omitempty" happydomain:"label=Unset,secret"`
	Nested  *nestedAuth `json:"nested,omitempty"`
}

func filled() *testProvider {
	return &testProvider{
		commonAuth: commonAuth{Account: "acct", AppKey: "app-key"},
		Host:       "dns.example.com",
		Secret:     "s3cr3t",
		KeyBlob:    []byte("raw-key-material"),
		Nested:     &nestedAuth{Endpoint: "https://example.com", Token: "nested-token"},
	}
}

func TestRedactSecrets(t *testing.T) {
	p := filled()
	RedactSecrets(p)

	if p.Secret != happydns.RedactedSecret {
		t.Errorf("Secret = %q, want the sentinel", p.Secret)
	}
	if p.AppKey != happydns.RedactedSecret {
		t.Errorf("embedded AppKey = %q, want the sentinel", p.AppKey)
	}
	if p.Nested.Token != happydns.RedactedSecret {
		t.Errorf("nested Token = %q, want the sentinel", p.Nested.Token)
	}
	if !bytes.Equal(p.KeyBlob, []byte(happydns.RedactedSecret)) {
		t.Errorf("KeyBlob = %q, want the sentinel", p.KeyBlob)
	}

	// Untagged neighbours must survive untouched, at every depth.
	if p.Host != "dns.example.com" {
		t.Errorf("Host = %q, want it left alone", p.Host)
	}
	if p.Account != "acct" {
		t.Errorf("embedded Account = %q, want it left alone", p.Account)
	}
	if p.Nested.Endpoint != "https://example.com" {
		t.Errorf("nested Endpoint = %q, want it left alone", p.Nested.Endpoint)
	}

	// An empty secret stays empty: the sentinel would make it look set.
	if p.Unset != "" {
		t.Errorf("Unset = %q, want it to stay empty", p.Unset)
	}
}

func TestRedactSecretsSurvivesJSONRoundTrip(t *testing.T) {
	p := filled()
	RedactSecrets(p)

	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var back testProvider
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if back.Secret != happydns.RedactedSecret {
		t.Errorf("Secret = %q after round-trip, want the sentinel", back.Secret)
	}
	// []byte travels as base64, which is why the sentinel has to survive it.
	if !bytes.Equal(back.KeyBlob, []byte(happydns.RedactedSecret)) {
		t.Errorf("KeyBlob = %q after round-trip, want the sentinel", back.KeyBlob)
	}

	if bytes.Contains(raw, []byte("s3cr3t")) || bytes.Contains(raw, []byte("app-key")) ||
		bytes.Contains(raw, []byte("nested-token")) {
		t.Errorf("redacted payload still carries a credential: %s", raw)
	}
}

func TestMergeSecretsRestoresUnchanged(t *testing.T) {
	existing := filled()

	incoming := filled()
	RedactSecrets(incoming)
	// What a client that only renamed the host sends back.
	incoming.Host = "other.example.com"

	MergeSecrets(existing, incoming)

	if incoming.Secret != "s3cr3t" {
		t.Errorf("Secret = %q, want the stored value restored", incoming.Secret)
	}
	if incoming.AppKey != "app-key" {
		t.Errorf("embedded AppKey = %q, want the stored value restored", incoming.AppKey)
	}
	if incoming.Nested.Token != "nested-token" {
		t.Errorf("nested Token = %q, want the stored value restored", incoming.Nested.Token)
	}
	if !bytes.Equal(incoming.KeyBlob, []byte("raw-key-material")) {
		t.Errorf("KeyBlob = %q, want the stored value restored", incoming.KeyBlob)
	}
	if incoming.Host != "other.example.com" {
		t.Errorf("Host = %q, want the submitted value kept", incoming.Host)
	}

	// The restored blob must not alias the stored one: the caller keeps one
	// body and drops the other.
	incoming.KeyBlob[0] = 'X'
	if existing.KeyBlob[0] == 'X' {
		t.Error("KeyBlob aliases the stored slice")
	}
}

func TestMergeSecretsKeepsNewValues(t *testing.T) {
	existing := filled()

	incoming := filled()
	RedactSecrets(incoming)
	incoming.Secret = "brand-new"
	incoming.KeyBlob = []byte("new-key-material")

	MergeSecrets(existing, incoming)

	if incoming.Secret != "brand-new" {
		t.Errorf("Secret = %q, want the newly submitted value", incoming.Secret)
	}
	if !bytes.Equal(incoming.KeyBlob, []byte("new-key-material")) {
		t.Errorf("KeyBlob = %q, want the newly submitted value", incoming.KeyBlob)
	}
	// Untouched ones still come back from storage.
	if incoming.AppKey != "app-key" {
		t.Errorf("embedded AppKey = %q, want the stored value restored", incoming.AppKey)
	}
}

func TestMergeSecretsWithoutExistingClearsSentinel(t *testing.T) {
	incoming := filled()
	RedactSecrets(incoming)

	// Creation: nothing is stored, so the placeholder must not survive into a
	// provider API call. `required` then reports the field as empty.
	MergeSecrets(nil, incoming)

	if incoming.Secret != "" {
		t.Errorf("Secret = %q, want it cleared", incoming.Secret)
	}
	if incoming.AppKey != "" {
		t.Errorf("embedded AppKey = %q, want it cleared", incoming.AppKey)
	}
	if incoming.Nested.Token != "" {
		t.Errorf("nested Token = %q, want it cleared", incoming.Nested.Token)
	}
	if len(incoming.KeyBlob) != 0 {
		t.Errorf("KeyBlob = %q, want it cleared", incoming.KeyBlob)
	}
	if incoming.Host != "dns.example.com" {
		t.Errorf("Host = %q, want it left alone", incoming.Host)
	}
}

func TestMergeSecretsIgnoresMismatchedType(t *testing.T) {
	// The user changed the provider type: the two bodies are unrelated, so
	// there is nothing to carry forward and the sentinel is cleared.
	existing := &nestedAuth{Endpoint: "https://example.com", Token: "nested-token"}

	incoming := filled()
	RedactSecrets(incoming)

	MergeSecrets(existing, incoming)

	if incoming.Secret != "" {
		t.Errorf("Secret = %q, want it cleared", incoming.Secret)
	}
	if incoming.Nested.Token != "" {
		t.Errorf("nested Token = %q, want it cleared", incoming.Nested.Token)
	}
}

func TestMergeSecretsNilNestedExisting(t *testing.T) {
	// Stored body has no nested part; the client submits one holding a
	// sentinel. There is nothing to restore, so it must be cleared, never kept.
	existing := filled()
	existing.Nested = nil

	incoming := filled()
	RedactSecrets(incoming)

	MergeSecrets(existing, incoming)

	if incoming.Nested.Token != "" {
		t.Errorf("nested Token = %q, want it cleared", incoming.Nested.Token)
	}
	if incoming.Secret != "s3cr3t" {
		t.Errorf("Secret = %q, want the stored value restored", incoming.Secret)
	}
}

func TestRedactAndMergeToleratePartialInput(t *testing.T) {
	// Neither helper may panic on the shapes a handler can realistically hand
	// them before anything has been decoded.
	RedactSecrets(nil)
	RedactSecrets((*testProvider)(nil))
	RedactSecrets("not a struct")

	MergeSecrets(nil, nil)
	MergeSecrets(filled(), nil)
	MergeSecrets((*testProvider)(nil), filled())
}
