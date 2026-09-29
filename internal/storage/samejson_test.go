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

package storage

import "testing"

func TestSameJSON(t *testing.T) {
	for _, tc := range []struct {
		name string
		a, b string
		want bool
	}{
		{"identical", `{"a":1,"b":"x"}`, `{"a":1,"b":"x"}`, true},
		{"key order and spacing", `{"a":1,"b":"x"}`, "{ \"b\": \"x\",\n \"a\": 1 }", true},
		{"different value", `{"a":1}`, `{"a":2}`, false},
		{"extra key", `{"a":1}`, `{"a":1,"b":null}`, false},
		{"array order matters", `[1,2]`, `[2,1]`, false},
		{"large integers kept exact", `9007199254740993`, `9007199254740992`, false},
		{"string vs number", `"1"`, `1`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SameJSON([]byte(tc.a), []byte(tc.b))
			if err != nil || got != tc.want {
				t.Errorf("SameJSON(%s, %s) = %v, %v; want %v", tc.a, tc.b, got, err, tc.want)
			}
		})
	}
}

func TestSameJSONRefusesInvalid(t *testing.T) {
	for _, pair := range [][2]string{{`{`, `{}`}, {`{}`, `{"a":}`}, {`{} {}`, `{}`}} {
		if _, err := SameJSON([]byte(pair[0]), []byte(pair[1])); err == nil {
			t.Errorf("SameJSON(%s, %s) accepted invalid JSON", pair[0], pair[1])
		}
	}
}
