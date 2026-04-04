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
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

func TestCheckPlan_IsFullyDisabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled map[string]bool
		want    bool
	}{
		{"nil map", nil, false},
		{"empty map", map[string]bool{}, false},
		{"all false", map[string]bool{"a": false, "b": false}, true},
		{"one true", map[string]bool{"a": false, "b": true}, false},
		{"all true", map[string]bool{"a": true, "b": true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &happydns.CheckPlan{Enabled: tt.enabled}
			if got := p.IsFullyDisabled(); got != tt.want {
				t.Errorf("IsFullyDisabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckPlan_IsRuleEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  map[string]bool
		rule     string
		want     bool
	}{
		{"nil map", nil, "any", true},
		{"empty map", map[string]bool{}, "any", true},
		{"rule explicitly enabled", map[string]bool{"r1": true}, "r1", true},
		{"rule explicitly disabled", map[string]bool{"r1": false}, "r1", false},
		{"rule missing from map", map[string]bool{"r1": false}, "r2", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &happydns.CheckPlan{Enabled: tt.enabled}
			if got := p.IsRuleEnabled(tt.rule); got != tt.want {
				t.Errorf("IsRuleEnabled(%q) = %v, want %v", tt.rule, got, tt.want)
			}
		})
	}
}

func TestTargetIdentifier(t *testing.T) {
	if got := happydns.TargetIdentifier(""); got != nil {
		t.Errorf("TargetIdentifier(\"\") = %v, want nil", got)
	}

	if got := happydns.TargetIdentifier("not-valid-hex"); got != nil {
		t.Errorf("TargetIdentifier(\"not-valid-hex\") = %v, want nil", got)
	}

	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatalf("NewRandomIdentifier: %v", err)
	}
	s := id.String()
	got := happydns.TargetIdentifier(s)
	if got == nil {
		t.Fatalf("TargetIdentifier(%q) = nil, want non-nil", s)
	}
	if !got.Equals(id) {
		t.Errorf("TargetIdentifier(%q) = %v, want %v", s, got, id)
	}
}

func TestFormatIdentifier(t *testing.T) {
	if got := happydns.FormatIdentifier(nil); got != "" {
		t.Errorf("FormatIdentifier(nil) = %q, want empty", got)
	}

	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatalf("NewRandomIdentifier: %v", err)
	}
	got := happydns.FormatIdentifier(&id)
	if got != id.String() {
		t.Errorf("FormatIdentifier(&id) = %q, want %q", got, id.String())
	}
}

func TestFieldFromCheckerOption(t *testing.T) {
	opt := happydns.CheckerOptionDocumentation{
		Id:          "myopt",
		Type:        "string",
		Label:       "My Option",
		Placeholder: "enter value",
		Default:     "default-val",
		Choices:     []string{"a", "b"},
		Required:    true,
		Secret:      true,
		Hide:        true,
		Textarea:    true,
		Description: "help text",
	}

	f := happydns.FieldFromCheckerOption(opt)

	if f.Id != opt.Id {
		t.Errorf("Id = %q, want %q", f.Id, opt.Id)
	}
	if f.Type != opt.Type {
		t.Errorf("Type = %q, want %q", f.Type, opt.Type)
	}
	if f.Label != opt.Label {
		t.Errorf("Label = %q, want %q", f.Label, opt.Label)
	}
	if f.Placeholder != opt.Placeholder {
		t.Errorf("Placeholder = %q, want %q", f.Placeholder, opt.Placeholder)
	}
	if f.Default != opt.Default {
		t.Errorf("Default = %v, want %v", f.Default, opt.Default)
	}
	if len(f.Choices) != len(opt.Choices) {
		t.Errorf("Choices len = %d, want %d", len(f.Choices), len(opt.Choices))
	}
	if f.Required != opt.Required {
		t.Errorf("Required = %v, want %v", f.Required, opt.Required)
	}
	if f.Secret != opt.Secret {
		t.Errorf("Secret = %v, want %v", f.Secret, opt.Secret)
	}
	if f.Hide != opt.Hide {
		t.Errorf("Hide = %v, want %v", f.Hide, opt.Hide)
	}
	if f.Textarea != opt.Textarea {
		t.Errorf("Textarea = %v, want %v", f.Textarea, opt.Textarea)
	}
	if f.Description != opt.Description {
		t.Errorf("Description = %q, want %q", f.Description, opt.Description)
	}
}
