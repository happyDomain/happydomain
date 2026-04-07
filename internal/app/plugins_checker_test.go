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

//go:build linux || darwin || freebsd

package app

import (
	"context"
	"errors"
	"plugin"
	"strings"
	"testing"

	sdk "git.happydns.org/checker-sdk-go/checker"
	"git.happydns.org/happyDomain/internal/checker"
	"git.happydns.org/happyDomain/model"
)

// dummyCheckerProvider is a minimal ObservationProvider used by the tests
// below. It is intentionally trivial: the loader tests only care that
// registration succeeds, not what the provider actually collects.
type dummyCheckerProvider struct {
	key happydns.ObservationKey
}

func (d *dummyCheckerProvider) Key() happydns.ObservationKey { return d.key }
func (d *dummyCheckerProvider) Collect(ctx context.Context, _ happydns.CheckerOptions) (any, error) {
	return nil, nil
}

func newDummyCheckerFactory(id string) func() (*sdk.CheckerDefinition, sdk.ObservationProvider, error) {
	return func() (*sdk.CheckerDefinition, sdk.ObservationProvider, error) {
		def := &sdk.CheckerDefinition{
			ID:   id,
			Name: "Dummy checker",
		}
		return def, &dummyCheckerProvider{key: happydns.ObservationKey("dummy-" + id)}, nil
	}
}

func TestLoadCheckerPlugin_SymbolMissing(t *testing.T) {
	found, err := loadCheckerPlugin(&fakeSymbols{}, "missing.so")
	if found || err != nil {
		t.Fatalf("expected (false, nil) when symbol is absent, got (%v, %v)", found, err)
	}
}

func TestLoadCheckerPlugin_WrongSymbolType(t *testing.T) {
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{
		"NewCheckerPlugin": 42, // not a function
	}}
	found, err := loadCheckerPlugin(fs, "wrongtype.so")
	if !found || err == nil || !strings.Contains(err.Error(), "unexpected type") {
		t.Fatalf("expected wrong-type error, got (%v, %v)", found, err)
	}
}

func TestLoadCheckerPlugin_FactoryError(t *testing.T) {
	factory := func() (*sdk.CheckerDefinition, sdk.ObservationProvider, error) {
		return nil, nil, errors.New("boom")
	}
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{"NewCheckerPlugin": factory}}

	found, err := loadCheckerPlugin(fs, "factoryerr.so")
	if !found || err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected factory error to propagate, got (%v, %v)", found, err)
	}
}

func TestLoadCheckerPlugin_NilDefinition(t *testing.T) {
	factory := func() (*sdk.CheckerDefinition, sdk.ObservationProvider, error) {
		return nil, &dummyCheckerProvider{key: "k"}, nil
	}
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{"NewCheckerPlugin": factory}}

	found, err := loadCheckerPlugin(fs, "nildef.so")
	if !found || err == nil || !strings.Contains(err.Error(), "nil CheckerDefinition") {
		t.Fatalf("expected nil-definition error, got (%v, %v)", found, err)
	}
}

func TestLoadCheckerPlugin_NilProvider(t *testing.T) {
	factory := func() (*sdk.CheckerDefinition, sdk.ObservationProvider, error) {
		return &sdk.CheckerDefinition{ID: "x"}, nil, nil
	}
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{"NewCheckerPlugin": factory}}

	found, err := loadCheckerPlugin(fs, "nilprov.so")
	if !found || err == nil || !strings.Contains(err.Error(), "nil ObservationProvider") {
		t.Fatalf("expected nil-provider error, got (%v, %v)", found, err)
	}
}

func TestLoadCheckerPlugin_FactoryPanics(t *testing.T) {
	factory := func() (*sdk.CheckerDefinition, sdk.ObservationProvider, error) {
		panic("kaboom")
	}
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{"NewCheckerPlugin": factory}}

	found, err := loadCheckerPlugin(fs, "panic.so")
	if !found || err == nil {
		t.Fatalf("expected panic to be converted to error, got (%v, %v)", found, err)
	}
	if !strings.Contains(err.Error(), "panicked") || !strings.Contains(err.Error(), "kaboom") {
		t.Errorf("expected wrapped panic error, got %v", err)
	}
}

func TestLoadCheckerPlugin_Success(t *testing.T) {
	factory := newDummyCheckerFactory("dummy-success")
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{"NewCheckerPlugin": factory}}

	found, err := loadCheckerPlugin(fs, "first.so")
	if !found || err != nil {
		t.Fatalf("expected success, got (%v, %v)", found, err)
	}

	if got := checker.FindChecker("dummy-success"); got == nil {
		t.Errorf("expected checker %q to be registered", "dummy-success")
	}
	if got := sdk.FindObservationProvider(happydns.ObservationKey("dummy-dummy-success")); got == nil {
		t.Errorf("expected observation provider %q to be registered", "dummy-dummy-success")
	}
}
