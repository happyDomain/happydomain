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

package secret

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/model"
)

var testSafeId = happydns.Identifier{0x01, 0x02, 0x03, 0x04}

func TestFormatParseSealedRoundTrip(t *testing.T) {
	payload := []byte{0x00, 0xff, 0x10, 'a', 'b'}

	token := FormatSealed(testSafeId, payload)
	if token != "hds:1:AQIDBA:AP8QYWI" {
		t.Errorf("FormatSealed = %q", token)
	}

	sv, err := ParseSealed(token)
	if err != nil {
		t.Fatalf("ParseSealed(%q) error = %v", token, err)
	}
	if !sv.SafeId.Equals(testSafeId) {
		t.Errorf("SafeId = %v, want %v", sv.SafeId, testSafeId)
	}
	if !bytes.Equal(sv.Payload, payload) {
		t.Errorf("Payload = %x, want %x", sv.Payload, payload)
	}
}

func TestParseSealedRejects(t *testing.T) {
	for name, token := range map[string]string{
		"empty":              "",
		"plaintext":          "my-api-key",
		"wrong version":      "hds:2:AQIDBA:AP8QYWI",
		"no version":         "hds::AQIDBA:AP8QYWI",
		"prefix only":        "hds:1:",
		"missing payload":    "hds:1:AQIDBA",
		"empty payload":      "hds:1:AQIDBA:",
		"empty safe id":      "hds:1::AP8QYWI",
		"extra colon in id":  "hds:1:AQ:IDBA:AP8QYWI",
		"invalid id base64":  "hds:1:A!IDBA:AP8QYWI",
		"invalid b64":        "hds:1:AQIDBA:AP8Q*WI",
		"padded payload":     "hds:1:AQIDBA:AP8QYWI=",
		"std base64 payload": "hds:1:AQIDBA:+/8=",
		"non canonical":      "hds:1:AQIDBA:AP8QYWJ",
		"upper prefix":       "HDS:1:AQIDBA:AP8QYWI",
		"leading space":      " hds:1:AQIDBA:AP8QYWI",
	} {
		if _, err := ParseSealed(token); err == nil {
			t.Errorf("%s: ParseSealed(%q) succeeded, want an error", name, token)
		}
	}
}

func TestIsSealed(t *testing.T) {
	for token, want := range map[string]bool{
		"hds:1:AQIDBA:AP8QYWI": true,
		"hds:1:":               true,
		"hds:1":                false,
		"hds:2:AQIDBA:AP8QYWI": false,
		"HDS:1:AQIDBA:AP8QYWI": false,
		"":                     false,
		"x hds:1:":             false,
	} {
		if got := IsSealed(token); got != want {
			t.Errorf("IsSealed(%q) = %v, want %v", token, got, want)
		}
	}
}

func FuzzParseSealed(f *testing.F) {
	for _, seed := range []string{
		"hds:1:AQIDBA:AP8QYWI",
		"hds:1:AQ:IDBA:AP8QYWI",
		"hds:1::",
		"hds:1:AQIDBA:AP8QYWJ",
		"hds:1:" + strings.Repeat("A", 22) + ":" + strings.Repeat("_-", 40),
		"plaintext",
		"",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, token string) {
		sv, err := ParseSealed(token)
		if err != nil {
			return
		}
		if again := FormatSealed(sv.SafeId, sv.Payload); again != token {
			t.Errorf("ParseSealed accepted %q, which formats back as %q", token, again)
		}
	})
}

func testContext() SecretContext {
	return SecretContext{
		Owner:      happydns.Identifier{0xaa, 0xbb},
		ObjectType: "provider",
		ObjectId:   "cHJvdmlkZXI",
		Field:      "ApiToken",
	}
}

func TestAssociatedDataGolden(t *testing.T) {
	// A change to this value breaks every sealed secret already stored.
	const golden = "000000056864733a31" + // "hds:1"
		"0000000401020304" + // safe id
		"00000002aabb" + // owner
		"0000000870726f7669646572" + // "provider"
		"0000000b63484a76646d6c6b5a5849" + // "cHJvdmlkZXI"
		"00000008417069546f6b656e" // "ApiToken"

	got := hex.EncodeToString(AssociatedData(testSafeId, testContext()))
	if got != golden {
		t.Errorf("AssociatedData = %s, want %s", got, golden)
	}
}

func TestAssociatedDataBindsEveryComponent(t *testing.T) {
	base := AssociatedData(testSafeId, testContext())

	variants := map[string]func(*happydns.Identifier, *SecretContext){
		"safe id":     func(id *happydns.Identifier, _ *SecretContext) { *id = happydns.Identifier{0x01, 0x02, 0x03, 0x05} },
		"owner":       func(_ *happydns.Identifier, sc *SecretContext) { sc.Owner = happydns.Identifier{0xaa, 0xbc} },
		"object type": func(_ *happydns.Identifier, sc *SecretContext) { sc.ObjectType = "notification" },
		"object id":   func(_ *happydns.Identifier, sc *SecretContext) { sc.ObjectId = "other" },
		"field":       func(_ *happydns.Identifier, sc *SecretContext) { sc.Field = "ApiSecret" },
	}

	for name, change := range variants {
		id := append(happydns.Identifier(nil), testSafeId...)
		sc := testContext()
		change(&id, &sc)

		if bytes.Equal(AssociatedData(id, sc), base) {
			t.Errorf("changing the %s does not change the associated data", name)
		}
	}
}

func TestAssociatedDataIsUnambiguous(t *testing.T) {
	a := testContext()
	a.ObjectId, a.Field = "ab", "c"
	b := testContext()
	b.ObjectId, b.Field = "a", "bc"

	if bytes.Equal(AssociatedData(testSafeId, a), AssociatedData(testSafeId, b)) {
		t.Error(`("ab","c") and ("a","bc") give the same associated data`)
	}
}

func TestSecretContextValidate(t *testing.T) {
	if err := testContext().Validate(); err != nil {
		t.Errorf("Validate() on a full context = %v", err)
	}

	for name, change := range map[string]func(*SecretContext){
		"owner":       func(sc *SecretContext) { sc.Owner = nil },
		"object type": func(sc *SecretContext) { sc.ObjectType = "" },
		"object id":   func(sc *SecretContext) { sc.ObjectId = "" },
		"field":       func(sc *SecretContext) { sc.Field = "" },
	} {
		sc := testContext()
		change(&sc)
		if err := sc.Validate(); err == nil {
			t.Errorf("Validate() without %s succeeded, want an error", name)
		}
	}
}

func TestSecretContextBinding(t *testing.T) {
	sc := SecretContext{Owner: happydns.Identifier{0x01}, ObjectType: "provider", ObjectId: "p", Field: "apikey"}

	if sc.binding() == "" || sc.binding() != sc.binding() {
		t.Fatal("binding must be non-empty and stable")
	}

	for name, other := range map[string]SecretContext{
		"owner":  {Owner: happydns.Identifier{0x02}, ObjectType: "provider", ObjectId: "p", Field: "apikey"},
		"type":   {Owner: happydns.Identifier{0x01}, ObjectType: "channel", ObjectId: "p", Field: "apikey"},
		"object": {Owner: happydns.Identifier{0x01}, ObjectType: "provider", ObjectId: "q", Field: "apikey"},
		"field":  {Owner: happydns.Identifier{0x01}, ObjectType: "provider", ObjectId: "p", Field: "token"},
	} {
		if other.binding() == sc.binding() {
			t.Errorf("changing the %s does not change the binding", name)
		}
	}
}
