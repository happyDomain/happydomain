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

package happydns_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/model"
)

const (
	clearValue  = "s3cr3t-api-key"
	sealedToken = "hds:1:AQIDBA:c2VhbGVk"
)

func unmarshalSecret(t *testing.T, raw string) happydns.Secret {
	t.Helper()

	var s happydns.Secret
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", raw, err)
	}
	return s
}

func TestSecretZeroValue(t *testing.T) {
	var s happydns.Secret

	if !s.IsEmpty() {
		t.Error("zero value IsEmpty() = false, want true")
	}
	if !s.IsZero() {
		t.Error("zero value IsZero() = false, want true")
	}
	if s.Reveal() != "" {
		t.Errorf("zero value Reveal() = %q, want empty", s.Reveal())
	}

	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal(zero) error = %v", err)
	}
	if string(b) != `""` {
		t.Errorf("json.Marshal(zero) = %s, want \"\"", b)
	}
}

func TestSecretUnmarshalEmpty(t *testing.T) {
	for _, raw := range []string{`""`, `null`} {
		s := unmarshalSecret(t, raw)
		if !s.IsEmpty() {
			t.Errorf("unmarshal %s: IsEmpty() = false, want true", raw)
		}
	}
}

func TestSecretUnmarshalNullKeepsNothing(t *testing.T) {
	s := happydns.NewSecret(clearValue)
	if err := json.Unmarshal([]byte(`null`), &s); err != nil {
		t.Fatalf("json.Unmarshal(null) error = %v", err)
	}
	if !s.IsEmpty() {
		t.Error("null over a clear value: IsEmpty() = false, want true")
	}
}

func TestSecretUnmarshalInvalidTypes(t *testing.T) {
	for _, raw := range []string{`42`, `true`, `{"a":"b"}`, `["a"]`} {
		var s happydns.Secret
		if err := json.Unmarshal([]byte(raw), &s); err == nil {
			t.Errorf("json.Unmarshal(%s) succeeded, want an error", raw)
		}
	}
}

func TestSecretClear(t *testing.T) {
	s := unmarshalSecret(t, `"`+clearValue+`"`)

	if !s.IsClear() {
		t.Fatal("unmarshal of an unprefixed value: IsClear() = false, want true")
	}
	if s.IsEmpty() || s.IsSealed() || s.IsOpened() || s.IsRedacted() {
		t.Error("clear secret reports another state too")
	}
	if s.Reveal() != clearValue {
		t.Errorf("Reveal() = %q, want %q", s.Reveal(), clearValue)
	}

	_, err := json.Marshal(s)
	if !errors.Is(err, happydns.ErrUnsealedSecret) {
		t.Errorf("json.Marshal(clear) error = %v, want ErrUnsealedSecret", err)
	}
}

func TestNewSecret(t *testing.T) {
	s := happydns.NewSecret(clearValue)
	if !s.IsClear() || s.Reveal() != clearValue {
		t.Errorf("NewSecret: IsClear() = %v, Reveal() = %q", s.IsClear(), s.Reveal())
	}

	if e := happydns.NewSecret(""); !e.IsEmpty() {
		t.Error(`NewSecret(""): IsEmpty() = false, want true`)
	}
}

func TestSecretSealed(t *testing.T) {
	s := unmarshalSecret(t, `"`+sealedToken+`"`)

	if !s.IsSealed() {
		t.Fatal("unmarshal of a hds:1: value: IsSealed() = false, want true")
	}
	if s.IsEmpty() || s.IsClear() || s.IsOpened() || s.IsRedacted() {
		t.Error("sealed secret reports another state too")
	}
	if s.Reveal() != "" {
		t.Errorf("sealed Reveal() = %q, want empty", s.Reveal())
	}
	if s.Token() != sealedToken {
		t.Errorf("Token() = %q, want %q", s.Token(), sealedToken)
	}

	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal(sealed) error = %v", err)
	}
	if string(b) != `"`+sealedToken+`"` {
		t.Errorf("json.Marshal(sealed) = %s, want the token byte for byte", b)
	}
}

func TestSecretSealedPrefixOnly(t *testing.T) {
	// Only the exact prefix marks a sealed value.
	for _, v := range []string{"hds:1", "hds:2:abc:def", "HDS:1:abc:def", " hds:1:abc:def"} {
		s := unmarshalSecret(t, `"`+v+`"`)
		if !s.IsClear() {
			t.Errorf("unmarshal %q: IsClear() = false, want true", v)
		}
	}
}

func TestSecretRedacted(t *testing.T) {
	raw, _ := json.Marshal(happydns.RedactedSecret)
	s := unmarshalSecret(t, string(raw))

	if !s.IsRedacted() {
		t.Fatal("unmarshal of the sentinel: IsRedacted() = false, want true")
	}
	if s.Reveal() != "" {
		t.Errorf("redacted Reveal() = %q, want empty", s.Reveal())
	}

	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal(redacted) error = %v", err)
	}
	if string(b) != string(raw) {
		t.Errorf("json.Marshal(redacted) = %s, want %s", b, raw)
	}
}

func TestSecretRedact(t *testing.T) {
	sealed := unmarshalSecret(t, `"`+sealedToken+`"`)
	opened := sealed
	opened.SetOpened(sealedToken, []byte(clearValue), "")

	for name, s := range map[string]happydns.Secret{
		"clear":  happydns.NewSecret(clearValue),
		"sealed": sealed,
		"opened": opened,
	} {
		s.Redact()
		if !s.IsRedacted() {
			t.Errorf("%s.Redact(): IsRedacted() = false, want true", name)
		}
		if s.Reveal() != "" {
			t.Errorf("%s.Redact(): Reveal() = %q, want empty", name, s.Reveal())
		}
	}

	var empty happydns.Secret
	empty.Redact()
	if !empty.IsEmpty() {
		t.Error("empty.Redact(): IsEmpty() = false, want true")
	}
}

func TestSecretSetSealed(t *testing.T) {
	s := happydns.NewSecret(clearValue)
	s.SetSealed(sealedToken)

	if !s.IsSealed() {
		t.Fatal("SetSealed: IsSealed() = false, want true")
	}
	if s.Reveal() != "" {
		t.Errorf("SetSealed: Reveal() = %q, want empty, the clear value must be dropped", s.Reveal())
	}

	b, err := json.Marshal(s)
	if err != nil || string(b) != `"`+sealedToken+`"` {
		t.Errorf("json.Marshal = %s, %v; want the token", b, err)
	}
}

func TestSecretSetOpened(t *testing.T) {
	s := unmarshalSecret(t, `"`+sealedToken+`"`)
	s.SetOpened(sealedToken, []byte(clearValue), "")

	if !s.IsOpened() {
		t.Fatal("SetOpened: IsOpened() = false, want true")
	}
	if s.Reveal() != clearValue {
		t.Errorf("opened Reveal() = %q, want %q", s.Reveal(), clearValue)
	}
	if s.Token() != sealedToken {
		t.Errorf("opened Token() = %q, want %q", s.Token(), sealedToken)
	}

	b, err := json.Marshal(s)
	if err != nil || string(b) != `"`+sealedToken+`"` {
		t.Errorf("json.Marshal(opened) = %s, %v; want the token", b, err)
	}
}

func TestSecretBinding(t *testing.T) {
	opened := unmarshalSecret(t, `"`+sealedToken+`"`)
	opened.SetOpened(sealedToken, []byte(clearValue), "ctx-a")

	if opened.Binding() != "ctx-a" {
		t.Errorf("opened Binding() = %q, want %q", opened.Binding(), "ctx-a")
	}

	// Only an opened secret is known to belong somewhere: the others say
	// nothing, rather than a binding they cannot vouch for.
	for name, s := range map[string]happydns.Secret{
		"empty":    {},
		"clear":    happydns.NewSecret(clearValue),
		"sealed":   unmarshalSecret(t, `"`+sealedToken+`"`),
		"redacted": unmarshalSecret(t, `"`+happydns.RedactedSecret+`"`),
	} {
		if s.Binding() != "" {
			t.Errorf("%s Binding() = %q, want empty", name, s.Binding())
		}
	}

	resealed := opened
	resealed.SetSealed(sealedToken)
	if resealed.Binding() != "" {
		t.Error("SetSealed must drop the binding along with the clear value")
	}
	if opened.Binding() != "ctx-a" {
		t.Error("changing a copy changed the binding of the original")
	}
}

func TestSecretSetOpenedPlaintextPolicy(t *testing.T) {
	// The plaintext policy "seals" a clear value by opening it under its own
	// value, so the record keeps the exact format it has today.
	s := happydns.NewSecret(clearValue)
	s.SetOpened(clearValue, []byte(clearValue), "")

	b, err := json.Marshal(s)
	if err != nil || string(b) != `"`+clearValue+`"` {
		t.Errorf("json.Marshal = %s, %v; want the raw value", b, err)
	}
	if s.Reveal() != clearValue {
		t.Errorf("Reveal() = %q, want %q", s.Reveal(), clearValue)
	}
}

func TestSecretClearForSealing(t *testing.T) {
	b, ok := happydns.NewSecret(clearValue).ClearForSealing()
	if !ok || string(b) != clearValue {
		t.Errorf("clear: ClearForSealing() = %q, %v; want the value, true", b, ok)
	}

	opened := unmarshalSecret(t, `"`+sealedToken+`"`)
	opened.SetOpened(sealedToken, []byte(clearValue), "")

	for name, s := range map[string]happydns.Secret{
		"empty":    {},
		"sealed":   unmarshalSecret(t, `"`+sealedToken+`"`),
		"opened":   opened,
		"redacted": unmarshalSecret(t, `"`+happydns.RedactedSecret+`"`),
	} {
		if b, ok := s.ClearForSealing(); ok || b != nil {
			t.Errorf("%s: ClearForSealing() = %q, %v; want nil, false", name, b, ok)
		}
	}
}

func TestSecretClearForSealingIsACopy(t *testing.T) {
	s := happydns.NewSecret(clearValue)
	b, _ := s.ClearForSealing()
	b[0] = 'X'

	if s.Reveal() != clearValue {
		t.Error("mutating the slice returned by ClearForSealing changed the secret")
	}
}

func TestSecretCopiesAreIndependent(t *testing.T) {
	a := happydns.NewSecret(clearValue)
	b := a
	b.SetSealed(sealedToken)

	if !a.IsClear() || a.Reveal() != clearValue {
		t.Error("sealing a copy changed the original")
	}
}

type nestedCreds struct {
	Token happydns.Secret `json:"token"`
}

type providerLike struct {
	Name   string          `json:"name"`
	ApiKey happydns.Secret `json:"apikey"`
	Nested *nestedCreds    `json:"nested"`
	Inline nestedCreds     `json:"inline"`
}

func TestSecretNeverFormatted(t *testing.T) {
	s := happydns.NewSecret(clearValue)
	opened := unmarshalSecret(t, `"`+sealedToken+`"`)
	opened.SetOpened(sealedToken, []byte(clearValue), "")

	p := providerLike{
		Name:   "visible",
		ApiKey: s,
		Nested: &nestedCreds{Token: opened},
		Inline: nestedCreds{Token: s},
	}

	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%X", "%d"} {
		for name, v := range map[string]any{
			"value":          s,
			"pointer":        &s,
			"opened":         opened,
			"struct":         p,
			"struct pointer": &p,
			"nested pointer": p.Nested,
		} {
			out := fmt.Sprintf(verb, v)
			if strings.Contains(out, clearValue) {
				t.Errorf("Sprintf(%s, %s) = %q leaks the clear value", verb, name, out)
			}
			// The bytes must not show up as numbers either.
			if strings.Contains(out, fmt.Sprint([]byte(clearValue))) {
				t.Errorf("Sprintf(%s, %s) = %q leaks the clear bytes", verb, name, out)
			}
			if strings.Contains(out, fmt.Sprintf("%x", clearValue)) {
				t.Errorf("Sprintf(%s, %s) = %q leaks the clear value in hex", verb, name, out)
			}
		}
	}

	if !strings.Contains(fmt.Sprintf("%+v", p), "visible") {
		t.Error("non-secret fields should still be printed")
	}
}

func TestSecretStringers(t *testing.T) {
	s := happydns.NewSecret(clearValue)

	if s.String() != happydns.RedactedSecret {
		t.Errorf("String() = %q, want the mask", s.String())
	}
	if strings.Contains(s.GoString(), clearValue) {
		t.Errorf("GoString() = %q leaks the clear value", s.GoString())
	}
}

func TestSecretStructWithClearFailsMarshal(t *testing.T) {
	p := providerLike{
		Name:   "p",
		ApiKey: happydns.NewSecret(clearValue),
	}

	b, err := json.Marshal(p)
	if !errors.Is(err, happydns.ErrUnsealedSecret) {
		t.Errorf("json.Marshal(struct with clear) = %s, %v; want ErrUnsealedSecret", b, err)
	}

	// Same through a pointer, nested deeper.
	q := providerLike{Nested: &nestedCreds{Token: happydns.NewSecret(clearValue)}}
	if _, err := json.Marshal(&q); !errors.Is(err, happydns.ErrUnsealedSecret) {
		t.Errorf("json.Marshal(nested clear) error = %v, want ErrUnsealedSecret", err)
	}
}

func TestSecretStructRoundTrip(t *testing.T) {
	raw := `{"name":"p","apikey":"` + sealedToken + `","nested":{"token":""},"inline":{"token":""}}`

	var p providerLike
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal error = %v", err)
	}
	if string(b) != raw {
		t.Errorf("round trip = %s, want %s", b, raw)
	}
}

func TestSecretOmitZero(t *testing.T) {
	type withOmit struct {
		Key happydns.Secret `json:"key,omitzero"`
	}

	b, err := json.Marshal(withOmit{})
	if err != nil || string(b) != `{}` {
		t.Errorf("json.Marshal(empty, omitzero) = %s, %v; want {}", b, err)
	}
}
