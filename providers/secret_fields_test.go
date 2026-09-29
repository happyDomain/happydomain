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
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	adapter "git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
	_ "git.happydns.org/happyDomain/providers"
)

var secretType = reflect.TypeFor[happydns.Secret]()

// secretLookingName matches field names that suggest a credential.
var secretLookingName = regexp.MustCompile(`(?i)key|token|secret|password|passwd|credential|totp`)

// notSecret lists fields whose name looks like a credential but that hold
// none: identifiers and names of keys, not the keys themselves. Each entry
// says why.
var notSecret = map[string]string{
	"AliDNSAPI.AccessKeyID":      "identifies the key, the secret is AccessKeySecret",
	"DDNSServer.KeyAlgo":         "the algorithm of the TSIG key",
	"DDNSServer.KeyName":         "the name of the TSIG key, sent in clear with every signed message",
	"GoDaddyAPI.APIKey":          "identifies the key, the secret is APISecret",
	"HuaweiCloudAPI.KeyID":       "identifies the key, the secret is SecretKey",
	"MythicBeastsAPI.KeyID":      "identifies the key, the secret is Secret",
	"PorkbunAPI.APIKey":          "the public part of the key, the secret is SecretKey",
	"Route53API.KeyId":           "identifies the key, the secret is SecretKey",
	"SakuraCloudAPI.AccessToken": "identifies the token, the secret is AccessTokenSecret",
	"ScalewayAPI.AccessKey":      "identifies the key, the secret is SecretKey",
}

type secretField struct {
	name string // Go path, Type.Field
	json string // JSON path
	typ  reflect.Type
	tag  string
}

func collectFields(t reflect.Type, goPrefix, jsonPrefix string, out *[]secretField) {
	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() && !sf.Anonymous {
			continue
		}

		jsonName, _, _ := strings.Cut(sf.Tag.Get("json"), ",")
		if jsonName == "-" {
			continue
		}
		if jsonName == "" {
			jsonName = sf.Name
		}

		ft := sf.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}

		if ft.Kind() == reflect.Struct && ft != secretType {
			jp := jsonPrefix
			if !sf.Anonymous {
				jp = joinPath(jsonPrefix, jsonName)
			}
			collectFields(ft, goPrefix, jp, out)
			continue
		}

		*out = append(*out, secretField{
			name: goPrefix + "." + sf.Name,
			json: joinPath(jsonPrefix, jsonName),
			typ:  sf.Type,
			tag:  sf.Tag.Get("happydomain"),
		})
	}
}

func joinPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func sortedProviders() []string {
	var names []string
	for name := range providerReg.GetProviders() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func providerFields(t *testing.T, name string) []secretField {
	t.Helper()
	body, err := providerReg.FindProvider(name)
	if err != nil {
		t.Fatal(err)
	}
	var fields []secretField
	collectFields(reflect.Indirect(reflect.ValueOf(body)).Type(), name, "", &fields)
	return fields
}

func TestProviderSecretFieldsAreSecrets(t *testing.T) {
	for _, name := range sortedProviders() {
		for _, f := range providerFields(t, name) {
			tagged := false
			for opt := range strings.SplitSeq(f.tag, ",") {
				if opt == "secret" {
					tagged = true
				}
			}

			if tagged && f.typ != secretType {
				t.Errorf("%s is tagged secret but is a %s, want happydns.Secret", f.name, f.typ)
			}

			if _, allowed := notSecret[f.name]; !allowed && secretLookingName.MatchString(f.name[strings.LastIndex(f.name, ".")+1:]) && f.typ != secretType {
				t.Errorf("%s looks like a credential but is a %s: make it a happydns.Secret, or list it in notSecret with the reason", f.name, f.typ)
			}
		}
	}
}

func TestNotSecretAllowlistIsCurrent(t *testing.T) {
	known := map[string]bool{}
	for _, name := range sortedProviders() {
		for _, f := range providerFields(t, name) {
			if f.typ != secretType {
				known[f.name] = true
			}
		}
	}

	for name := range notSecret {
		if !known[name] {
			t.Errorf("notSecret lists %s, which is not a non-secret field of a provider", name)
		}
	}
}

// legacyValue is a unique value per field, valid base64 so that fields holding
// encoded key material decode it.
func legacyValue(f secretField) string {
	return base64.StdEncoding.EncodeToString([]byte("legacy:" + f.name))
}

// configOf returns everything the backend library is given for body.
func configOf(t *testing.T, body happydns.ProviderBody) (string, bool) {
	t.Helper()

	switch b := body.(type) {
	case adapter.DNSControlConfigAdapter:
		cfg, err := b.ToDNSControlConfig()
		if err != nil {
			return fmt.Sprintf("error: %s", err), true
		}
		var parts []string
		for k, v := range cfg {
			parts = append(parts, k+"="+v)
		}
		return strings.Join(parts, "\n"), true
	case adapter.LibdnsConfigAdapter:
		return fmt.Sprintf("%#v", reflect.Indirect(reflect.ValueOf(b.LibdnsProvider())).Interface()), true
	}
	return "", false
}

// Records stored before Secret existed hold plain strings. They must still
// decode, and reach the backend library in clear once opened.
func TestProviderLegacyRecordsReachBackend(t *testing.T) {
	secrets, err := secret.NewManager(secret.PolicyPlaintext)
	if err != nil {
		t.Fatal(err)
	}

	sc := secret.SecretContext{Owner: happydns.Identifier{1}, ObjectType: "provider", ObjectId: "legacy"}

	for _, name := range sortedProviders() {
		t.Run(name, func(t *testing.T) {
			fields := providerFields(t, name)

			legacy := map[string]any{}
			var secretsOf []secretField
			for _, f := range fields {
				switch {
				case f.typ == secretType:
					secretsOf = append(secretsOf, f)
					setPath(legacy, f.json, legacyValue(f))
				case f.typ.Kind() == reflect.String:
					// Some backends only use a secret alongside another
					// setting, a key name for example.
					setPath(legacy, f.json, "setting")
				}
			}
			if len(secretsOf) == 0 {
				return
			}

			raw, err := json.Marshal(legacy)
			if err != nil {
				t.Fatal(err)
			}

			body, err := providerReg.FindProvider(name)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(raw, body); err != nil {
				t.Fatalf("legacy record %s does not decode: %v", raw, err)
			}

			if err := secrets.OpenObject(context.Background(), sc, body); err != nil {
				t.Fatalf("OpenObject: %v", err)
			}

			cfg, ok := configOf(t, body)
			if !ok {
				t.Skipf("%T has no known backend configuration", body)
			}

			for _, f := range secretsOf {
				if !strings.Contains(cfg, legacyValue(f)) {
					t.Errorf("%s does not reach the backend in clear:\n%s", f.name, cfg)
				}
			}

			// And the stored form does not change once sealed in plaintext.
			if err := secrets.SealObject(context.Background(), sc, body); err != nil {
				t.Fatalf("SealObject: %v", err)
			}
			back, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("json.Marshal after sealing: %v", err)
			}
			for _, f := range secretsOf {
				if !strings.Contains(string(back), `"`+legacyValue(f)+`"`) {
					t.Errorf("%s is not stored as before: %s", f.name, back)
				}
			}
		})
	}
}

func setPath(m map[string]any, path string, v any) {
	head, rest, nested := strings.Cut(path, ".")
	if !nested {
		m[head] = v
		return
	}
	sub, ok := m[head].(map[string]any)
	if !ok {
		sub = map[string]any{}
		m[head] = sub
	}
	setPath(sub, rest, v)
}

// KeyBlob used to be a []byte, stored as base64: the same text is now the
// value of the Secret, and must give the backend the same key.
func TestDDNSLegacyKeyBlob(t *testing.T) {
	key := []byte{0x00, 0x01, 0xfe, 0xff, 'k', 'e', 'y'}

	legacy, err := json.Marshal(struct {
		Server  string `json:"server"`
		KeyName string `json:"keyname"`
		KeyAlgo string `json:"algorithm"`
		KeyBlob []byte `json:"keyblob"`
	}{"192.0.2.1", "ddns", "hmac-sha256", key})
	if err != nil {
		t.Fatal(err)
	}

	body, err := providerReg.FindProvider("DDNSServer")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(legacy, body); err != nil {
		t.Fatalf("legacy record %s does not decode: %v", legacy, err)
	}

	cfg, err := body.(adapter.DNSControlConfigAdapter).ToDNSControlConfig()
	if err != nil {
		t.Fatal(err)
	}

	want := "hmac-sha256:ddns:" + base64.StdEncoding.EncodeToString(key)
	if cfg["transfer-key"] != want || cfg["update-key"] != want {
		t.Errorf("keys = %q, %q; want %q", cfg["transfer-key"], cfg["update-key"], want)
	}
}

func TestDDNSInvalidKeyBlob(t *testing.T) {
	body, err := providerReg.FindProvider("DDNSServer")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"keyname":"ddns","algorithm":"hmac-sha256","keyblob":"not base64!"}`), body); err != nil {
		t.Fatal(err)
	}

	if _, err := body.(adapter.DNSControlConfigAdapter).ToDNSControlConfig(); err == nil {
		t.Error("an invalid base64 key must be refused")
	}
}
