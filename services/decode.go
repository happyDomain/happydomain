// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package svcs

import (
	"fmt"
	"log"
	"reflect"
	"sort"

	"git.happydns.org/happydns/model"
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
	pathToSvcsModule string                       = "git.happydns.org/happydns/services"
	ordered_services []*Svc
)

func RegisterService(creator ServiceCreator, analyzer ServiceAnalyzer, infos ServiceInfos, weight uint32, aliases ...string) {
	// Invalidate ordered_services, which serve as cache
	ordered_services = nil

	baseType := reflect.Indirect(reflect.ValueOf(creator())).Type()
	name := baseType.String()
	log.Println("Registering new service:", name)

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
	if t.Kind() == reflect.Struct && t.PkgPath() == pathToSvcsModule {
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
		return nil, fmt.Errorf("Unable to find corresponding service for `%s`.", name)
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
