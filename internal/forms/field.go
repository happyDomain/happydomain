// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

package forms // import "git.happydns.org/happyDomain/forms"

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"git.happydns.org/happyDomain/model"
)

// GenField generates a generic Field based on the happydomain tag.
func GenField(field reflect.StructField) (f *happydns.Field) {
	jsonTag := field.Tag.Get("json")
	jsonTuples := strings.Split(jsonTag, ",")

	f = &happydns.Field{
		Type: field.Type.String(),
	}

	if len(jsonTuples) > 0 && len(jsonTuples[0]) > 0 {
		f.Id = jsonTuples[0]
	} else {
		f.Id = field.Name
	}

	tag := field.Tag.Get("happydomain")

	for t := range strings.SplitSeq(tag, ",") {
		kv := strings.SplitN(t, "=", 2)
		if len(kv) > 1 {
			switch strings.ToLower(kv[0]) {
			case "label":
				f.Label = kv[1]
			case "placeholder":
				f.Placeholder = kv[1]
			case "default":
				f.Default = kv[1]
			case "description":
				f.Description = kv[1]
			case "choices":
				f.Choices = strings.Split(kv[1], ";")
			}
		} else {
			switch strings.ToLower(kv[0]) {
			case "hidden":
				f.Hide = true
			case "required":
				f.Required = true
			case "secret":
				f.Secret = true
			case "textarea":
				f.Textarea = true
			default:
				f.Label = kv[0]
			}
		}
	}
	return
}

// ValidateStructValues validates the field values of a struct against the
// constraints declared in its happydomain struct tags (choices, required).
// Since the struct is already typed, basic type checking is handled by the
// JSON decoder; this function validates higher-level constraints.
func ValidateStructValues(data any) error {
	if data == nil {
		return nil
	}

	v := reflect.Indirect(reflect.ValueOf(data))
	if !v.IsValid() {
		return nil
	}
	t := v.Type()
	if t.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.Anonymous {
			if err := ValidateStructValues(v.Field(i).Interface()); err != nil {
				return err
			}
			continue
		}

		field := GenField(sf)
		fv := v.Field(i)

		if field.Required && fv.IsZero() {
			label := field.Label
			if label == "" {
				label = field.Id
			}
			return fmt.Errorf("field %q is required", label)
		}

		if len(field.Choices) > 0 && fv.Kind() == reflect.String {
			s := fv.String()
			if s != "" && !slices.Contains(field.Choices, s) {
				label := field.Label
				if label == "" {
					label = field.Id
				}
				return fmt.Errorf("field %q: value %q is not a valid choice (valid: %v)", label, s, field.Choices)
			}
		}
	}

	return nil
}

// ValidateMapValues validates a map[string]any against a slice of Field definitions.
// It checks required fields, choices constraints, basic type compatibility,
// and rejects unknown keys not declared in any field definition.
func ValidateMapValues(opts map[string]any, fields []happydns.Field) error {
	known := make(map[string]*happydns.Field, len(fields))
	for i := range fields {
		known[fields[i].Id] = &fields[i]
	}

	// Reject unknown keys.
	for k := range opts {
		if _, ok := known[k]; !ok {
			return fmt.Errorf("unknown option %q", k)
		}
	}

	for _, f := range fields {
		v, exists := opts[f.Id]

		label := f.Label
		if label == "" {
			label = f.Id
		}

		// Required check.
		if f.Required {
			if !exists || v == nil {
				return fmt.Errorf("field %q is required", label)
			}
			if s, ok := v.(string); ok && s == "" {
				return fmt.Errorf("field %q is required", label)
			}
		}

		if !exists || v == nil {
			continue
		}

		// Choices check.
		if len(f.Choices) > 0 {
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("field %q: expected a string value for choices field", label)
			}
			if s != "" && !slices.Contains(f.Choices, s) {
				return fmt.Errorf("field %q: value %q is not a valid choice (valid: %v)", label, s, f.Choices)
			}
		}

		// Basic type check.
		if f.Type != "" {
			if err := checkMapValueType(f.Type, v, label); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkMapValueType performs a basic type compatibility check between a Field.Type
// string and the actual value from a map[string]any (JSON-decoded).
func checkMapValueType(fieldType string, value any, label string) error {
	switch {
	case strings.HasPrefix(fieldType, "string"):
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field %q: expected string, got %T", label, value)
		}
	case strings.HasPrefix(fieldType, "int") || strings.HasPrefix(fieldType, "uint") || strings.HasPrefix(fieldType, "float"):
		// JSON numbers decode as float64.
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("field %q: expected number, got %T", label, value)
		}
	case fieldType == "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field %q: expected bool, got %T", label, value)
		}
	}
	return nil
}

// GenStructFields generates corresponding SourceFields of the given Source.
func GenStructFields(data any) (fields []*happydns.Field) {
	if data != nil {
		dataMeta := reflect.Indirect(reflect.ValueOf(data)).Type()

		for i := 0; i < dataMeta.NumField(); i += 1 {
			if dataMeta.Field(i).Anonymous {
				fields = append(fields, GenStructFields(reflect.Indirect(reflect.ValueOf(data)).Field(i))...)
			} else {
				fields = append(fields, GenField(dataMeta.Field(i)))
			}
		}
	}
	return
}
