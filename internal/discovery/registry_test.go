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

package discovery

import (
	"encoding/json"
	"slices"
	"testing"
)

// factory returns the factory registered under id, failing the test when
// there is none.
func factory(t *testing.T, id string) SourceFactory {
	t.Helper()

	for _, f := range SourceFactories() {
		if f.ID == id {
			return f
		}
	}

	t.Fatalf("no technique %q is registered", id)
	return SourceFactory{}
}

// isolateSources runs the test against an empty registry of techniques,
// restoring the real one when it ends.
func isolateSources(t *testing.T) {
	t.Helper()

	sourceFactoriesMu.Lock()
	saved := sourceFactories
	sourceFactories = map[string]SourceFactory{}
	sourceFactoriesMu.Unlock()

	t.Cleanup(func() {
		sourceFactoriesMu.Lock()
		sourceFactories = saved
		sourceFactoriesMu.Unlock()
	})
}

// settingsFrom returns the settings of the discovery, as decoded from js.
func settingsFrom(t *testing.T, js string) *Settings {
	t.Helper()

	settings := new(Settings)
	if err := json.Unmarshal([]byte(js), settings); err != nil {
		t.Fatalf("settings: %v", err)
	}
	return settings
}

func TestSourceFactories(t *testing.T) {
	factories := SourceFactories()
	if !slices.IsSortedFunc(factories, func(a, b SourceFactory) int { return a.Priority - b.Priority }) {
		t.Error("techniques are not in priority order")
	}

	for _, f := range factories {
		// Built out of its default settings, a technique enabled by
		// default must carry the ID it is registered under.
		if source := f.New(new(Settings)); source != nil && source.ID() != f.ID {
			t.Errorf("technique %q builds sources identified %q", f.ID, source.ID())
		}
	}
}

func TestRegisterSourceTwice(t *testing.T) {
	isolateSources(t)

	build := func(*Settings) NameSource { return nil }
	RegisterSource("test-registered-twice", 0, build)

	defer func() {
		if recover() == nil {
			t.Error("registering a technique twice did not panic")
		}
	}()
	RegisterSource("test-registered-twice", 0, build)
}

func TestSourceFactoriesEqualPriority(t *testing.T) {
	isolateSources(t)

	build := func(*Settings) NameSource { return nil }
	RegisterSource("test-order-b", 12345, build)
	RegisterSource("test-order-a", 12345, build)

	var ids []string
	for _, f := range SourceFactories() {
		ids = append(ids, f.ID)
	}
	if !slices.Equal(ids, []string{"test-order-a", "test-order-b"}) {
		t.Errorf("techniques of equal priority = %v; want them by ID", ids)
	}
}
