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

package usecase

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type serviceSpecsUsecase struct {
}

func NewServiceSpecsUsecase() happydns.ServiceSpecsUsecase {
	return &serviceSpecsUsecase{}
}

func (ssu *serviceSpecsUsecase) ListServices() map[string]happydns.ServiceInfos {
	services := svcs.ListServices()

	ret := map[string]happydns.ServiceInfos{}
	for k, service := range *services {
		ret[k] = service.Infos
	}

	return ret
}

func (ssu *serviceSpecsUsecase) GetServiceIcon(ssid string) ([]byte, error) {
	cnt, ok := svcs.Icons[strings.TrimSuffix(ssid, ".png")]
	if !ok {
		return nil, happydns.NotFoundError{Msg: "service icon not found"}
	}

	return cnt, nil
}

func (ssu *serviceSpecsUsecase) GetServiceSpecs(svctype reflect.Type) (*happydns.ServiceSpecs, error) {
	return ssu.getSpecs(svctype)
}

func (ssu *serviceSpecsUsecase) InitializeService(svctype reflect.Type) (any, error) {
	// Create a new instance of the service
	svcPtr := reflect.New(svctype)
	svc := svcPtr.Interface()

	// Check if the service implements ServiceInitializer interface
	if initializer, ok := svc.(happydns.ServiceInitializer); ok {
		return initializer.Initialize()
	}

	// Otherwise, initialize with default empty values
	svcValue := svcPtr.Elem()

	// Special case: if there's only one field and it's a slice of non-complex types, initialize with one empty element
	settableFields := ssu.countSettableFields(svcValue)
	if settableFields == 1 {
		for i := 0; i < svcValue.NumField(); i++ {
			field := svcValue.Field(i)
			fieldType := svcValue.Type().Field(i)

			if !field.CanSet() || fieldType.Anonymous {
				continue
			}

			// If it's a slice, initialize with one empty element only if it's not a pointer or struct
			if field.Kind() == reflect.Slice {
				elemType := field.Type().Elem()

				// Only initialize with one element if it's not a pointer or struct
				if elemType.Kind() != reflect.Ptr && elemType.Kind() != reflect.Struct {
					slice := reflect.MakeSlice(field.Type(), 1, 1)
					// Set the first element to zero value (e.g., "" for string)
					slice.Index(0).Set(reflect.Zero(elemType))
					field.Set(slice)
					return svc, nil
				}
			}
			break
		}
	}

	ssu.initializeStructFields(svcValue)

	return svc, nil
}

func (ssu *serviceSpecsUsecase) countSettableFields(v reflect.Value) int {
	count := 0
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)

		if field.CanSet() && !fieldType.Anonymous {
			count++
		}
	}
	return count
}

func (ssu *serviceSpecsUsecase) initializeStructFields(v reflect.Value) {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Handle anonymous embedded structs
		if fieldType.Anonymous {
			if field.Kind() == reflect.Struct {
				ssu.initializeStructFields(field)
			}
			continue
		}

		// Initialize based on field type
		switch field.Kind() {
		case reflect.Slice:
			// Initialize slices as empty (non-nil)
			field.Set(reflect.MakeSlice(field.Type(), 0, 0))
		case reflect.Map:
			// Initialize maps as empty (non-nil)
			field.Set(reflect.MakeMap(field.Type()))
		case reflect.Ptr:
			// For pointer types, check if it's a DNS type first
			elemType := field.Type().Elem()
			if ssu.isDNSType(elemType) {
				newVal := reflect.New(elemType)
				ssu.initializeDNSRecord(newVal.Elem())
				field.Set(newVal)
			} else if elemType.Kind() == reflect.Struct {
				newVal := reflect.New(elemType)
				ssu.initializeStructFields(newVal.Elem())
				field.Set(newVal)
			}
		case reflect.Struct:
			// Check if it's a DNS type
			if ssu.isDNSType(field.Type()) {
				ssu.initializeDNSRecord(field)
			} else {
				// Recursively initialize nested structs
				ssu.initializeStructFields(field)
			}
			// Numeric types, strings, bools, etc. already have their zero values
		}
	}
}

// isDNSType checks if a type is from the miekg/dns package or a happyDomain DNS abstraction
func (ssu *serviceSpecsUsecase) isDNSType(t reflect.Type) bool {
	pkgPath := t.PkgPath()

	// Check if it's from miekg/dns package
	if pkgPath == "github.com/miekg/dns" {
		return true
	}

	// Check if it's a happyDomain DNS abstraction (e.g., happydns.TXT, happydns.SPF)
	// These have a dns.RR_Header field named "Hdr"
	if pkgPath == "git.happydns.org/happyDomain/model" && t.Kind() == reflect.Struct {
		if field, ok := t.FieldByName("Hdr"); ok {
			return field.Type == reflect.TypeOf(dns.RR_Header{})
		}
	}

	return false
}

// initializeDNSRecord initializes a DNS record with sensible defaults
func (ssu *serviceSpecsUsecase) initializeDNSRecord(v reflect.Value) {
	if v.Kind() != reflect.Struct {
		return
	}

	// Determine the Rrtype based on the DNS record type name
	rrtype := ssu.getRRType(v.Type())

	// Initialize the Hdr field if it exists
	hdrField := v.FieldByName("Hdr")
	if hdrField.IsValid() && hdrField.CanSet() {
		hdrField.Set(reflect.ValueOf(dns.RR_Header{
			Name:     "",
			Rrtype:   rrtype,
			Class:    dns.ClassINET,
			Ttl:      0,
			Rdlength: 0,
		}))
	}

	// Initialize other fields to their zero values (empty strings, 0 for numbers, etc.)
	// This is already done by Go's zero value initialization
}

func (ssu *serviceSpecsUsecase) getSpecs(svcType reflect.Type) (*happydns.ServiceSpecs, error) {
	fields := []happydns.Field{}
	for i := 0; i < svcType.NumField(); i += 1 {
		if svcType.Field(i).Anonymous {
			ssp, err := ssu.getSpecs(svcType.Field(i).Type)
			if err != nil {
				return nil, err
			}
			fields = append(fields, ssp.Fields...)
			continue
		}

		jsonTag := svcType.Field(i).Tag.Get("json")
		jsonTuples := strings.Split(jsonTag, ",")

		f := happydns.Field{
			Type: svcType.Field(i).Type.String(),
		}

		if len(jsonTuples) > 0 && len(jsonTuples[0]) > 0 {
			f.Id = jsonTuples[0]
		} else {
			f.Id = svcType.Field(i).Name
		}

		tag := svcType.Field(i).Tag.Get("happydomain")
		tuples := strings.Split(tag, ",")

		for _, t := range tuples {
			kv := strings.SplitN(t, "=", 2)
			if len(kv) > 1 {
				switch strings.ToLower(kv[0]) {
				case "label":
					f.Label = kv[1]
				case "placeholder":
					f.Placeholder = kv[1]
				case "default":
					var err error
					if strings.HasPrefix(f.Type, "uint") {
						f.Default, err = strconv.ParseUint(kv[1], 10, 64)
					} else if strings.HasPrefix(f.Type, "int") {
						f.Default, err = strconv.ParseInt(kv[1], 10, 64)
					} else if strings.HasPrefix(f.Type, "float") {
						f.Default, err = strconv.ParseFloat(kv[1], 64)
					} else if strings.HasPrefix(f.Type, "bool") {
						f.Default, err = strconv.ParseBool(kv[1])
					} else {
						f.Default = kv[1]
					}

					if err != nil {
						return nil, fmt.Errorf("format error for default field %s of type %s definition: %w", svcType.Field(i).Name, svcType.Name(), err)
					}
				case "description":
					f.Description = kv[1]
				case "choices":
					f.Choices = strings.Split(kv[1], ";")
				}
			} else {
				switch strings.ToLower(kv[0]) {
				case "required":
					f.Required = true
				case "secret":
					f.Secret = true
				case "hidden":
					f.Hide = true
				default:
					f.Label = kv[0]
				}
			}
		}
		fields = append(fields, f)
	}

	return &happydns.ServiceSpecs{Fields: fields}, nil
}
