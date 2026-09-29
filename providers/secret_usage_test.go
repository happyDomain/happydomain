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

package providers_test

import (
	"encoding/json"
	"reflect"
	"testing"

	adapter "git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/internal/forms"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/providers"
)

// unopenedSecrets are the states in which a Secret is set, so not empty, but
// has no clear value to give: Reveal returns "" for both.
func unopenedSecrets() map[string]happydns.Secret {
	redacted := happydns.NewSecret("x")
	redacted.Redact()

	var sealed happydns.Secret
	sealed.SetSealed(happydns.SealedSecretPrefix + "c2FmZQ:cGF5bG9hZA")

	return map[string]happydns.Secret{
		"redacted": redacted,
		"sealed":   sealed,
	}
}

type optionalSecretCase struct {
	name string
	prvd adapter.DNSControlConfigAdapter
	keys []string
}

// optionalSecretCases puts s in the optional credentials of a few providers,
// listing the configuration keys that carry them.
func optionalSecretCases(s happydns.Secret) []optionalSecretCase {
	return []optionalSecretCase{
		{"Joker", &providers.JokerAPI{Username: "u", Password: s, APIKey: s}, []string{"password", "api-key"}},
		{"Route53", &providers.Route53API{KeyId: "k", SecretKey: s, Token: s}, []string{"SecretKey", "Token"}},
		{"INWX", &providers.INWXAPI{Username: "u", Password: happydns.NewSecret("p"), TOTPKey: s}, []string{"totp-key"}},
		{"HEDNS", &providers.HEDNSAPI{Username: "u", Password: happydns.NewSecret("p"), TOTP: s}, []string{"totp-key"}},
	}
}

// An optional credential that is set but not opened must be left out of the
// configuration, as if it were empty, rather than sent as an empty string:
// Joker would then prefer an empty password to the API key, Route53 would
// stop falling back to the instance credentials.
func TestOptionalUnopenedSecretsAreLeftOut(t *testing.T) {
	for state, s := range unopenedSecrets() {
		for _, tc := range optionalSecretCases(s) {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				config, err := tc.prvd.ToDNSControlConfig()
				if err != nil {
					t.Fatalf("ToDNSControlConfig: %v", err)
				}
				for _, k := range tc.keys {
					if v, ok := config[k]; ok {
						t.Errorf("config[%q] = %q, want the key left out", k, v)
					}
				}
			})
		}
	}
}

// The same optional credentials, once opened, reach the configuration.
func TestOptionalOpenedSecretsAreSent(t *testing.T) {
	for _, tc := range optionalSecretCases(happydns.NewSecret("v")) {
		t.Run(tc.name, func(t *testing.T) {
			config, err := tc.prvd.ToDNSControlConfig()
			if err != nil {
				t.Fatalf("ToDNSControlConfig: %v", err)
			}
			for _, k := range tc.keys {
				if config[k] != "v" {
					t.Errorf("config[%q] = %q, want %q", k, config[k], "v")
				}
			}
		})
	}
}

// A TSIG key that is set but not opened must not turn into an empty key: the
// provider would sign with it and fail with BADSIG, far from the cause.
func TestDDNSUnopenedKeyIsAnError(t *testing.T) {
	for state, s := range unopenedSecrets() {
		t.Run(state, func(t *testing.T) {
			prvd := &providers.DDNSServer{KeyName: "ddns", KeyAlgo: "hmac-sha256", KeyBlob: s}
			if config, err := prvd.ToDNSControlConfig(); err == nil {
				t.Errorf("ToDNSControlConfig succeeded with %v, want an error", config)
			}
		})
	}
}

// The TSIG key is base64 key material: the form has to tell the client, which
// checks the encoding as the user types.
func TestDDNSKeyBlobIsBase64(t *testing.T) {
	sf, _ := reflect.TypeOf(providers.DDNSServer{}).FieldByName("KeyBlob")
	if f := forms.GenField(sf); f.Type != "base64" {
		t.Errorf("KeyBlob: Type = %q, want base64", f.Type)
	}
}

// User exports taken while KeyBlob was a []byte carry the placeholder in
// base64. Restored, it must read as a placeholder, not as a TSIG key made of
// the bytes of "••••••••".
func TestDDNSLegacyExportPlaceholderIsRedacted(t *testing.T) {
	var prvd providers.DDNSServer
	if err := json.Unmarshal([]byte(`{"keyname":"ddns","keyblob":"`+happydns.RedactedSecretBase64+`"}`), &prvd); err != nil {
		t.Fatal(err)
	}
	if !prvd.KeyBlob.IsRedacted() {
		t.Errorf("KeyBlob: IsRedacted() = false, want true")
	}
}
