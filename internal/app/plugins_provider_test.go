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
	"errors"
	"plugin"
	"strings"
	"testing"

	providerReg "git.happydns.org/happyDomain/internal/provider"
	"git.happydns.org/happyDomain/model"
)

// dummyProviderBody is a minimal happydns.ProviderBody used by the tests
// below; we only care that loadProviderPlugin can register it without
// touching real DNS code.
type dummyProviderBody struct {
	Endpoint string
}

func (d *dummyProviderBody) InstantiateProvider() (happydns.ProviderActuator, error) {
	return nil, errors.New("not implemented in tests")
}

func newDummyProviderFactory() func() (happydns.ProviderCreatorFunc, happydns.ProviderInfos, error) {
	return func() (happydns.ProviderCreatorFunc, happydns.ProviderInfos, error) {
		creator := func() happydns.ProviderBody { return &dummyProviderBody{} }
		return creator, happydns.ProviderInfos{Name: "Dummy"}, nil
	}
}

func TestLoadProviderPlugin_SymbolMissing(t *testing.T) {
	found, err := loadProviderPlugin(&fakeSymbols{}, "missing.so")
	if found || err != nil {
		t.Fatalf("expected (false, nil) when symbol is absent, got (%v, %v)", found, err)
	}
}

func TestLoadProviderPlugin_WrongSymbolType(t *testing.T) {
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{
		"NewProviderPlugin": 42, // not a function
	}}
	found, err := loadProviderPlugin(fs, "wrongtype.so")
	if !found || err == nil {
		t.Fatalf("expected (true, err) for wrong symbol type, got (%v, %v)", found, err)
	}
	if !strings.Contains(err.Error(), "unexpected type") {
		t.Errorf("expected error to mention unexpected type, got %v", err)
	}
}

func TestLoadProviderPlugin_FactoryError(t *testing.T) {
	factory := func() (happydns.ProviderCreatorFunc, happydns.ProviderInfos, error) {
		return nil, happydns.ProviderInfos{}, errors.New("boom")
	}
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{"NewProviderPlugin": factory}}

	found, err := loadProviderPlugin(fs, "factoryerr.so")
	if !found || err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected factory error to propagate, got (%v, %v)", found, err)
	}
}

func TestLoadProviderPlugin_NilCreator(t *testing.T) {
	factory := func() (happydns.ProviderCreatorFunc, happydns.ProviderInfos, error) {
		return nil, happydns.ProviderInfos{Name: "Dummy"}, nil
	}
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{"NewProviderPlugin": factory}}

	found, err := loadProviderPlugin(fs, "nilcreator.so")
	if !found || err == nil || !strings.Contains(err.Error(), "nil ProviderCreatorFunc") {
		t.Fatalf("expected nil creator to be rejected, got (%v, %v)", found, err)
	}
}

func TestLoadProviderPlugin_FactoryPanics(t *testing.T) {
	factory := func() (happydns.ProviderCreatorFunc, happydns.ProviderInfos, error) {
		panic("kaboom")
	}
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{"NewProviderPlugin": factory}}

	found, err := loadProviderPlugin(fs, "panic.so")
	if !found || err == nil {
		t.Fatalf("expected panic to be converted to error, got (%v, %v)", found, err)
	}
	if !strings.Contains(err.Error(), "panicked") || !strings.Contains(err.Error(), "kaboom") {
		t.Errorf("expected wrapped panic error, got %v", err)
	}
}

func TestLoadProviderPlugin_SuccessAndDuplicate(t *testing.T) {
	factory := newDummyProviderFactory()
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{"NewProviderPlugin": factory}}

	// First registration should succeed and use a fully-qualified name
	// (package.Type) so it cannot collide with a built-in or another plugin
	// shipping a "dummyProviderBody" struct in a different package.
	found, err := loadProviderPlugin(fs, "first.so")
	if !found || err != nil {
		t.Fatalf("expected first load to succeed, got (%v, %v)", found, err)
	}

	const expectedKey = "app.dummyProviderBody"
	if _, ok := providerReg.GetProviders()[expectedKey]; !ok {
		t.Fatalf("expected provider to be registered as %q, registry has: %v",
			expectedKey, keysOf(providerReg.GetProviders()))
	}

	// Second registration of the same qualified name must be a no-op (just
	// a warning); the existing entry should still be there afterwards.
	found, err = loadProviderPlugin(fs, "second.so")
	if !found || err != nil {
		t.Fatalf("expected second load to be silently ignored, got (%v, %v)", found, err)
	}
}

func keysOf[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
