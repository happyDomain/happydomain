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

package ct

import "testing"

func TestBackends(t *testing.T) {
	seen := map[string]bool{}
	for _, b := range Backends {
		if b.ID == "" || b.Name == "" || b.New == nil {
			t.Errorf("backend %+v lacks an ID, a name or a constructor", b)
			continue
		}
		if seen[b.ID] {
			t.Errorf("backend %q listed twice", b.ID)
		}
		seen[b.ID] = true

		if b.New(Config{}) == nil {
			t.Errorf("backend %q builds no searcher", b.ID)
		}
	}
}

func TestTakesAPIKey(t *testing.T) {
	if !certSpotter.TakesAPIKey {
		t.Error("Cert Spotter does not tell it takes an API key")
	}
}
