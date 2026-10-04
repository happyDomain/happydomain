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
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	_ "git.happydns.org/happyDomain/providers"
)

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

type providerField struct {
	name string // Go path, Type.Field
	tag  string
}

func collectFields(t reflect.Type, prefix string, out *[]providerField) {
	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() && !sf.Anonymous {
			continue
		}
		if name, _, _ := strings.Cut(sf.Tag.Get("json"), ","); name == "-" {
			continue
		}

		ft := sf.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			collectFields(ft, prefix, out)
			continue
		}

		*out = append(*out, providerField{
			name: prefix + "." + sf.Name,
			tag:  sf.Tag.Get("happydomain"),
		})
	}
}

func sortedProviders() []string {
	var names []string
	for name := range providerReg.GetProviders() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func providerFields(t *testing.T, name string) []providerField {
	t.Helper()
	body, err := providerReg.FindProvider(name)
	if err != nil {
		t.Fatal(err)
	}
	var fields []providerField
	collectFields(reflect.Indirect(reflect.ValueOf(body)).Type(), name, &fields)
	return fields
}

func isTaggedSecret(f providerField) bool {
	for opt := range strings.SplitSeq(f.tag, ",") {
		if opt == "secret" {
			return true
		}
	}
	return false
}

// A credential not tagged `secret` is sent back to the user in clear, and
// written in clear into the account export.
func TestProviderCredentialsAreTaggedSecret(t *testing.T) {
	for _, name := range sortedProviders() {
		for _, f := range providerFields(t, name) {
			if _, allowed := notSecret[f.name]; allowed || isTaggedSecret(f) {
				continue
			}
			if secretLookingName.MatchString(f.name[strings.LastIndex(f.name, ".")+1:]) {
				t.Errorf("%s looks like a credential but is not tagged secret: tag it, or list it in notSecret with the reason", f.name)
			}
		}
	}
}

func TestNotSecretAllowlistIsCurrent(t *testing.T) {
	known := map[string]bool{}
	for _, name := range sortedProviders() {
		for _, f := range providerFields(t, name) {
			if !isTaggedSecret(f) {
				known[f.name] = true
			}
		}
	}

	for name := range notSecret {
		if !known[name] {
			t.Errorf("notSecret lists %s, which is not an untagged field of a provider", name)
		}
	}
}
