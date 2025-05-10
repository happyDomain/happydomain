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
