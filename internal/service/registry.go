// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package service

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"

	"git.happydns.org/happyDomain/model"
)

type Svc struct {
	Creator  happydns.ServiceCreator
	Analyzer ServiceAnalyzer
	Infos    happydns.ServiceInfos
	Weight   uint32
}

type ByWeight []*Svc

func (a ByWeight) Len() int           { return len(a) }
func (a ByWeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByWeight) Less(i, j int) bool { return a[i].Weight < a[j].Weight }

// The service and sub-service registries below are intentionally unguarded.
// All writes (RegisterService, RegisterPluginSubServices, RegisterSubServices)
// happen from App.initPlugins() at startup, *before* App.initUsecases() and
// before any goroutine that could read them (see internal/app/app.go). From
// that point on the maps are read-only for the rest of the process lifetime,
// so concurrent reads are safe without locking. Any future code path that
// needs to mutate these maps after startup must introduce its own
// synchronisation (sync.RWMutex around services, subServices and
// ordered_services together).
var (
	services         map[string]*Svc                       = map[string]*Svc{}
	subServices      map[string]happydns.SubServiceCreator = map[string]happydns.SubServiceCreator{}
	pathToSvcsModule string                                = "git.happydns.org/happyDomain/services"
	ordered_services []*Svc
)

func RegisterService(creator happydns.ServiceCreator, analyzer ServiceAnalyzer, infos happydns.ServiceInfos, weight uint32, aliases ...string) {
	baseType := reflect.Indirect(reflect.ValueOf(creator())).Type()
	name := baseType.String()

	// A second registration of the same name almost always means a plugin is
	// shadowing a built-in (or another plugin) by accident. Log loudly and
	// keep the existing entry rather than silently overwriting it.
	if _, exists := services[name]; exists {
		log.Printf("Warning: service %q is already registered; ignoring duplicate registration", name)
		return
	}

	// Invalidate ordered_services, which serve as cache
	ordered_services = nil

	log.Println("Registering new service:", name)

	// Override given parameters by true one
	infos.Type = name

	svc := &Svc{
		creator,
		analyzer,
		infos,
		weight,
	}
	services[name] = svc

	// Register aliases
	for _, alias := range aliases {
		if _, exists := services[alias]; exists {
			log.Printf("Warning: service alias %q is already registered; ignoring", alias)
			continue
		}
		services[alias] = svc
	}

	// Register sub types
	RegisterSubServices(baseType)
}

// RegisterPluginSubServices walks the type tree rooted at t and registers
// every nested struct type as a sub-service, regardless of its package path.
//
// The built-in RegisterSubServices intentionally restricts itself to types
// declared under git.happydns.org/happyDomain/services to avoid registering
// random struct types pulled in from third-party libraries by built-in
// services. Plugin services live in a completely different module path, so
// that filter would skip every nested type they declare and break
// (de)serialisation of any non-flat plugin payload. The plugin loader calls
// this function explicitly to opt the plugin's own types into the registry.
func RegisterPluginSubServices(t reflect.Type) {
	switch t.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Map:
		RegisterPluginSubServices(t.Elem())
		return
	case reflect.Struct:
		// Anonymous structs have no name and cannot be looked up later.
		if t.Name() == "" {
			return
		}
		key := t.String()
		if _, ok := subServices[key]; ok {
			return
		}
		log.Println("Registering new plugin subservice:", key)
		subServices[key] = func() any {
			return reflect.New(t).Interface()
		}
		for i := 0; i < t.NumField(); i++ {
			RegisterPluginSubServices(t.Field(i).Type)
		}
	}
}

func RegisterSubServices(t reflect.Type) {
	if t.Kind() == reflect.Struct && strings.HasPrefix(t.PkgPath(), pathToSvcsModule) {
		if _, ok := subServices[t.String()]; !ok {
			log.Println("Registering new subservice:", t.String())

			subServices[t.String()] = func() any {
				return reflect.New(t).Interface()
			}
		}

		for i := 0; i < t.NumField(); i += 1 {
			RegisterSubServices(t.Field(i).Type)
		}
	} else if t.Kind() == reflect.Array || t.Kind() == reflect.Map || t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		RegisterSubServices(t.Elem())
	} else if t.PkgPath() == pathToSvcsModule {
		if _, ok := subServices[t.String()]; ok {
			return
		}

		log.Println("Registering new subservice:", t.String())

		subServices[t.String()] = func() any {
			return reflect.New(t).Interface()
		}
	}
}

func OrderedServices() []*Svc {
	if ordered_services == nil {
		// Create the list
		for _, svc := range services {
			ordered_services = append(ordered_services, svc)
		}

		// Sort the list
		sort.Sort(ByWeight(ordered_services))
	}

	return ordered_services
}

func ListServices() *map[string]*Svc {
	return &services
}

func FindService(name string) (happydns.ServiceBody, error) {
	svc, ok := services[name]
	if !ok {
		return nil, happydns.NewServiceNotFoundError(name)
	}

	return svc.Creator(), nil
}

func FindSubService(name string) (any, error) {
	if svc, ok := services[name]; ok {
		return svc.Creator(), nil
	} else if ssvc, ok := subServices[name]; ok {
		return ssvc(), nil
	} else {
		return nil, fmt.Errorf("Unable to find corresponding service `%s`.", name)
	}
}
