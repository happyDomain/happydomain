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
