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
)

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
