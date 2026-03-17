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
	t := v.Type()

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
