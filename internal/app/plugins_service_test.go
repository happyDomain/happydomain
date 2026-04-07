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

	svcs "git.happydns.org/happyDomain/internal/service"
	"git.happydns.org/happyDomain/model"
)

// dummyNested is referenced as a struct field by dummyServiceBody to verify
// that loadServicePlugin walks the type tree and registers nested types as
// sub-services, something the built-in walker refuses to do for types that
// live outside the happydomain/services module path.
type dummyNested struct {
	Value string
}

type dummyServiceBody struct {
	Hostname string
	Detail   dummyNested
}

func (d *dummyServiceBody) GetNbResources() int { return 1 }
func (d *dummyServiceBody) GenComment() string  { return "dummy" }
func (d *dummyServiceBody) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	return nil, nil
}

func newDummyServiceFactory() func() (happydns.ServiceCreator, svcs.ServiceAnalyzer, happydns.ServiceInfos, uint32, []string, error) {
	return func() (happydns.ServiceCreator, svcs.ServiceAnalyzer, happydns.ServiceInfos, uint32, []string, error) {
		creator := func() happydns.ServiceBody { return &dummyServiceBody{} }
		return creator, nil, happydns.ServiceInfos{Name: "Dummy"}, 100, nil, nil
	}
}

func TestLoadServicePlugin_SymbolMissing(t *testing.T) {
	found, err := loadServicePlugin(&fakeSymbols{}, "missing.so")
	if found || err != nil {
		t.Fatalf("expected (false, nil), got (%v, %v)", found, err)
	}
}

func TestLoadServicePlugin_WrongSymbolType(t *testing.T) {
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{
		"NewServicePlugin": "not a function",
	}}
	found, err := loadServicePlugin(fs, "wrongtype.so")
	if !found || err == nil || !strings.Contains(err.Error(), "unexpected type") {
		t.Fatalf("expected wrong-type error, got (%v, %v)", found, err)
	}
}

func TestLoadServicePlugin_FactoryError(t *testing.T) {
	factory := func() (happydns.ServiceCreator, svcs.ServiceAnalyzer, happydns.ServiceInfos, uint32, []string, error) {
		return nil, nil, happydns.ServiceInfos{}, 0, nil, errors.New("boom")
	}
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{"NewServicePlugin": factory}}

	found, err := loadServicePlugin(fs, "factoryerr.so")
	if !found || err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected factory error to propagate, got (%v, %v)", found, err)
	}
}

func TestLoadServicePlugin_NilCreator(t *testing.T) {
	factory := func() (happydns.ServiceCreator, svcs.ServiceAnalyzer, happydns.ServiceInfos, uint32, []string, error) {
		return nil, nil, happydns.ServiceInfos{Name: "Dummy"}, 0, nil, nil
	}
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{"NewServicePlugin": factory}}

	found, err := loadServicePlugin(fs, "nilcreator.so")
	if !found || err == nil || !strings.Contains(err.Error(), "nil ServiceCreator") {
		t.Fatalf("expected nil-creator error, got (%v, %v)", found, err)
	}
}

func TestLoadServicePlugin_FactoryPanics(t *testing.T) {
	factory := func() (happydns.ServiceCreator, svcs.ServiceAnalyzer, happydns.ServiceInfos, uint32, []string, error) {
		panic("kaboom")
	}
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{"NewServicePlugin": factory}}

	found, err := loadServicePlugin(fs, "panic.so")
	if !found || err == nil || !strings.Contains(err.Error(), "panicked") {
		t.Fatalf("expected wrapped panic error, got (%v, %v)", found, err)
	}
}

func TestLoadServicePlugin_SuccessRegistersSubServices(t *testing.T) {
	factory := newDummyServiceFactory()
	fs := &fakeSymbols{syms: map[string]plugin.Symbol{"NewServicePlugin": factory}}

	found, err := loadServicePlugin(fs, "first.so")
	if !found || err != nil {
		t.Fatalf("expected success, got (%v, %v)", found, err)
	}

	// The service itself must be reachable through the registry.
	const svcKey = "app.dummyServiceBody"
	if _, err := svcs.FindService(svcKey); err != nil {
		t.Fatalf("expected service %q to be registered: %v", svcKey, err)
	}

	// And so must the nested struct: this is the regression-prevention test
	// for the built-in walker's pathToSvcsModule prefix check, which would
	// otherwise refuse to register types from outside happydomain/services.
	const nestedKey = "app.dummyNested"
	if _, err := svcs.FindSubService(nestedKey); err != nil {
		t.Errorf("expected nested type %q to be registered as a sub-service: %v", nestedKey, err)
	}

	// Loading the same plugin twice must be a no-op (collision warning).
	found, err = loadServicePlugin(fs, "second.so")
	if !found || err != nil {
		t.Fatalf("expected duplicate load to be silently ignored, got (%v, %v)", found, err)
	}
}
