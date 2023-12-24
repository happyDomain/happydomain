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

package svcs

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"

	"git.happydns.org/happyDomain/model"
)

const (
	Abstract = "abstract"
	Hidden   = "hidden"
	Provider = "provider"
)

type ServiceCreator func() happydns.Service
type SubServiceCreator func() interface{}
type ServiceAnalyzer func(*Analyzer) error

type Svc struct {
	Creator  ServiceCreator
	Analyzer ServiceAnalyzer
	Infos    ServiceInfos
	Weight   uint32
}

type ByWeight []*Svc

func (a ByWeight) Len() int           { return len(a) }
func (a ByWeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByWeight) Less(i, j int) bool { return a[i].Weight < a[j].Weight }

var (
	services         map[string]*Svc              = map[string]*Svc{}
	subServices      map[string]SubServiceCreator = map[string]SubServiceCreator{}
	pathToSvcsModule string                       = "git.happydns.org/happyDomain/services"
	ordered_services []*Svc
)

func RegisterService(creator ServiceCreator, analyzer ServiceAnalyzer, infos ServiceInfos, weight uint32, aliases ...string) {
	// Invalidate ordered_services, which serve as cache
	ordered_services = nil

	baseType := reflect.Indirect(reflect.ValueOf(creator())).Type()
	name := baseType.String()
	log.Println("Registering new service:", name)

	// Override given parameters by true one
	infos.Type = name
	if _, ok := Icons[name]; ok {
		infos.Icon = "/api/service_specs/" + name + "/icon.png"
	}

	svc := &Svc{
		creator,
		analyzer,
		infos,
		weight,
	}
	services[name] = svc

	// Register aliases
	for _, alias := range aliases {
		services[alias] = svc
	}

	// Register sub types
	RegisterSubServices(baseType)
}

func RegisterSubServices(t reflect.Type) {
	if t.Kind() == reflect.Struct && strings.HasPrefix(t.PkgPath(), pathToSvcsModule) {
		if _, ok := subServices[t.String()]; !ok {
			log.Println("Registering new subservice:", t.String())

			subServices[t.String()] = func() interface{} {
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

		subServices[t.String()] = func() interface{} {
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

func GetServices() *map[string]*Svc {
	return &services
}

func FindService(name string) (happydns.Service, error) {
	svc, ok := services[name]
	if !ok {
		return nil, ServiceNotFoundError{name}
	}

	return svc.Creator(), nil
}

func FindSubService(name string) (interface{}, error) {
	if svc, ok := services[name]; ok {
		return svc.Creator(), nil
	} else if ssvc, ok := subServices[name]; ok {
		return ssvc(), nil
	} else {
		return nil, fmt.Errorf("Unable to find corresponding service `%s`.", name)
	}
}
