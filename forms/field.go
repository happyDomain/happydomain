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
	"reflect"
	"strings"
)

// Field
type Field struct {
	// Id is the field identifier.
	Id string `json:"id"`

	// Type is the string representation of the field's type.
	Type string `json:"type"`

	// Label is the title given to the field, displayed as <label> tag on the interface.
	Label string `json:"label,omitempty"`

	// Placeholder is the placeholder attribute of the corresponding <input> tag.
	Placeholder string `json:"placeholder,omitempty"`

	// Default is the preselected value or the default value in case the field is not filled by the user.
	Default string `json:"default,omitempty"`

	// Choices holds the differents choices shown to the user in <select> tag.
	Choices []string `json:"choices,omitempty"`

	// Required indicates whether the field has to be filled or not.
	Required bool `json:"required,omitempty"`

	// Secret indicates if the field contains sensitive information such as API key, in order to hide
	// the field when not needed. When typing, it doesn't hide characters like in password input.
	Secret bool `json:"secret,omitempty"`

	// Hide indicates if the field should be hidden to the user.
	Hide bool `json:"hide,omitempty"`

	// Description stores an helpfull sentence describing the field.
	Description string `json:"description,omitempty"`
}

// GenField generates a generic Field based on the happydomain tag.
func GenField(field reflect.StructField) (f *Field) {
	jsonTag := field.Tag.Get("json")
	jsonTuples := strings.Split(jsonTag, ",")

	f = &Field{
		Type: field.Type.String(),
	}

	if len(jsonTuples) > 0 && len(jsonTuples[0]) > 0 {
		f.Id = jsonTuples[0]
	} else {
		f.Id = field.Name
	}

	tag := field.Tag.Get("happydomain")
	tuples := strings.Split(tag, ",")

	for _, t := range tuples {
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
			default:
				f.Label = kv[0]
			}
		}
	}
	return
}

// GenStructFields generates corresponding SourceFields of the given Source.
func GenStructFields(data interface{}) (fields []*Field) {
	if data != nil {
		dataMeta := reflect.Indirect(reflect.ValueOf(data)).Type()

		for i := 0; i < dataMeta.NumField(); i += 1 {
			fields = append(fields, GenField(dataMeta.Field(i)))
		}
	}
	return
}

// GenDefaultSettingsForm generates a generic CustomForm presenting all the fields in one page.
func GenDefaultSettingsForm(data interface{}) *CustomForm {
	return &CustomForm{
		Fields:                 GenStructFields(data),
		NextButtonText:         "common.create-thing",
		NextEditButtonText:     "common.update",
		NextButtonState:        1,
		PreviousButtonText:     "provider.another",
		PreviousEditButtonText: "common.cancel",
		PreviousButtonState:    -1,
	}
}
