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
	"reflect"
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

func TestValidateMapValues_Required(t *testing.T) {
	fields := []happydns.Field{
		{Id: "name", Type: "string", Required: true, Label: "Name"},
	}

	// Missing required field.
	if err := ValidateMapValues(map[string]any{}, fields); err == nil {
		t.Fatal("expected error for missing required field")
	}

	// Nil value.
	if err := ValidateMapValues(map[string]any{"name": nil}, fields); err == nil {
		t.Fatal("expected error for nil required field")
	}

	// Empty string value.
	if err := ValidateMapValues(map[string]any{"name": ""}, fields); err == nil {
		t.Fatal("expected error for empty string required field")
	}

	// Valid value.
	if err := ValidateMapValues(map[string]any{"name": "hello"}, fields); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMapValues_Choices(t *testing.T) {
	fields := []happydns.Field{
		{Id: "color", Type: "string", Choices: []string{"red", "green", "blue"}},
	}

	if err := ValidateMapValues(map[string]any{"color": "red"}, fields); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := ValidateMapValues(map[string]any{"color": "yellow"}, fields); err == nil {
		t.Fatal("expected error for invalid choice")
	}

	// Empty string is allowed (field not required).
	if err := ValidateMapValues(map[string]any{"color": ""}, fields); err != nil {
		t.Fatalf("unexpected error for empty choice: %v", err)
	}
}

func TestValidateMapValues_TypeCheck(t *testing.T) {
	fields := []happydns.Field{
		{Id: "count", Type: "int"},
		{Id: "label", Type: "string"},
		{Id: "enabled", Type: "bool"},
	}

	// Valid types.
	if err := ValidateMapValues(map[string]any{"count": float64(5), "label": "test", "enabled": true}, fields); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wrong type for int field.
	if err := ValidateMapValues(map[string]any{"count": "notanumber"}, fields); err == nil {
		t.Fatal("expected error for wrong type on int field")
	}

	// Wrong type for string field.
	if err := ValidateMapValues(map[string]any{"label": float64(42)}, fields); err == nil {
		t.Fatal("expected error for wrong type on string field")
	}

	// Wrong type for bool field.
	if err := ValidateMapValues(map[string]any{"enabled": "yes"}, fields); err == nil {
		t.Fatal("expected error for wrong type on bool field")
	}
}

func TestValidateMapValues_UnknownKeys(t *testing.T) {
	fields := []happydns.Field{
		{Id: "name", Type: "string"},
	}

	if err := ValidateMapValues(map[string]any{"name": "ok", "unknown": "bad"}, fields); err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestValidateMapValues_EmptyFieldsAndOpts(t *testing.T) {
	// No fields defined, empty options: valid.
	if err := ValidateMapValues(map[string]any{}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No fields defined, but has options: rejected as unknown.
	if err := ValidateMapValues(map[string]any{"x": 1}, nil); err == nil {
		t.Fatal("expected error for unknown key with no fields")
	}
}

func TestValidateMapValues_ChoicesNonString(t *testing.T) {
	fields := []happydns.Field{
		{Id: "mode", Type: "string", Choices: []string{"a", "b"}},
	}

	// Non-string value on a choices field.
	if err := ValidateMapValues(map[string]any{"mode": float64(1)}, fields); err == nil {
		t.Fatal("expected error for non-string choices value")
	}
}

func TestValidateMapValues_RequiredNonString(t *testing.T) {
	fields := []happydns.Field{
		{Id: "count", Type: "int", Required: true, Label: "Count"},
	}

	// Missing required int field.
	if err := ValidateMapValues(map[string]any{}, fields); err == nil {
		t.Fatal("expected error for missing required int field")
	}

	// Nil value for required int field.
	if err := ValidateMapValues(map[string]any{"count": nil}, fields); err == nil {
		t.Fatal("expected error for nil required int field")
	}

	// Zero value passes (not treated as empty for non-string types).
	if err := ValidateMapValues(map[string]any{"count": float64(0)}, fields); err != nil {
		t.Fatalf("unexpected error for zero-value required int: %v", err)
	}

	// Valid non-zero value.
	if err := ValidateMapValues(map[string]any{"count": float64(5)}, fields); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMapValues_ChoicesWithTypeCheck(t *testing.T) {
	fields := []happydns.Field{
		{Id: "color", Type: "string", Choices: []string{"red", "green", "blue"}},
	}

	// Valid choice passes both choices and type check.
	if err := ValidateMapValues(map[string]any{"color": "red"}, fields); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invalid choice fails at choices check (type is correct).
	if err := ValidateMapValues(map[string]any{"color": "yellow"}, fields); err == nil {
		t.Fatal("expected error for invalid choice with type+choices field")
	}

	// Wrong type fails at choices check before reaching type check.
	if err := ValidateMapValues(map[string]any{"color": float64(42)}, fields); err == nil {
		t.Fatal("expected error for non-string value on choices+type field")
	}
}

func TestValidateStructValues_NilPointer(t *testing.T) {
	type S struct {
		Name string `happydomain:"required"`
	}
	// Typed nil pointer must not panic.
	if err := ValidateStructValues((*S)(nil)); err != nil {
		t.Fatalf("expected nil error for typed nil pointer, got %v", err)
	}
}

func TestValidateStructValues_NonStruct(t *testing.T) {
	// Non-struct values must not panic.
	if err := ValidateStructValues("hello"); err != nil {
		t.Fatalf("expected nil error for non-struct, got %v", err)
	}
	if err := ValidateStructValues(42); err != nil {
		t.Fatalf("expected nil error for non-struct, got %v", err)
	}
}

func TestGenFieldEndpoint(t *testing.T) {
	type S struct {
		Bare      string `json:"bare" happydomain:"label=Bare,endpoint"`
		Defaulted string `json:"defaulted" happydomain:"label=Defaulted,endpoint=127.0.0.1"`
		Plain     string `json:"plain" happydomain:"label=Plain"`
	}

	t.Run("bare marker", func(t *testing.T) {
		f := GenField(reflect.TypeFor[S]().Field(0))
		if !f.Endpoint {
			t.Error("Endpoint = false, want true")
		}
		// The bare-keyword switch falls through to setting the label, so a
		// missing case would silently rename the field to "endpoint".
		if f.Label != "Bare" {
			t.Errorf("Label = %q, want %q: the marker leaked into the label", f.Label, "Bare")
		}
	})

	t.Run("marker with a default", func(t *testing.T) {
		f := GenField(reflect.TypeFor[S]().Field(1))
		if !f.Endpoint {
			t.Error("Endpoint = false, want true")
		}
		if f.EndpointDefault != "127.0.0.1" {
			t.Errorf("EndpointDefault = %q, want %q", f.EndpointDefault, "127.0.0.1")
		}
	})

	t.Run("untagged field", func(t *testing.T) {
		if f := GenField(reflect.TypeFor[S]().Field(2)); f.Endpoint {
			t.Error("Endpoint = true on an untagged field")
		}
	})
}

func TestEndpoints(t *testing.T) {
	type Nested struct {
		Inner string `json:"inner" happydomain:"label=Inner,endpoint"`
	}
	type Embedded struct {
		Parent string `json:"parent" happydomain:"label=Parent,endpoint"`
	}
	type Body struct {
		Embedded
		Host      string `json:"host" happydomain:"label=Host,endpoint"`
		Fallback  string `json:"fallback" happydomain:"label=Fallback,endpoint=127.0.0.1"`
		Blank     string `json:"blank" happydomain:"label=Blank,endpoint"`
		NotTagged string `json:"not_tagged" happydomain:"label=Not tagged"`
		Port      int    `json:"port" happydomain:"label=Port,endpoint"`
		Nested    Nested `json:"nested"`
		unexposed string `happydomain:"endpoint"`
	}

	body := &Body{
		Embedded:  Embedded{Parent: "parent.example.com"},
		Host:      "pdns.example.com",
		NotTagged: "ignored.example.com",
		Port:      8081,
		Nested:    Nested{Inner: "inner.example.com"},
		unexposed: "unexposed.example.com",
	}

	got := Endpoints(body)

	want := map[string]string{
		"Parent":   "parent.example.com",
		"Host":     "pdns.example.com",
		"Fallback": "127.0.0.1",
		"Inner":    "inner.example.com",
	}

	if len(got) != len(want) {
		t.Fatalf("Endpoints() = %v, want %d entries", got, len(want))
	}
	for _, e := range got {
		if want[e.Label] != e.Value {
			t.Errorf("Endpoints() returned %q = %q, want %q", e.Label, e.Value, want[e.Label])
		}
	}
}

func TestEndpointsNoPanic(t *testing.T) {
	type S struct {
		Host string `happydomain:"endpoint"`
	}

	if got := Endpoints((*S)(nil)); got != nil {
		t.Errorf("Endpoints(typed nil) = %v, want nil", got)
	}
	if got := Endpoints(nil); got != nil {
		t.Errorf("Endpoints(nil) = %v, want nil", got)
	}
	if got := Endpoints("hello"); got != nil {
		t.Errorf("Endpoints(non struct) = %v, want nil", got)
	}
}
