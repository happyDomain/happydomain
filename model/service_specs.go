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

package happydns

import (
	"fmt"
	"reflect"
)

type ServiceNotFoundError struct {
	name string
}

func NewServiceNotFoundError(name string) *ServiceNotFoundError {
	return &ServiceNotFoundError{
		name,
	}
}

func (err ServiceNotFoundError) Error() string {
	return fmt.Sprintf("Unable to find corresponding service for `%s`.", err.name)
}

const (
	SERVICE_FAMILY_ABSTRACT = "abstract"
	SERVICE_FAMILY_HIDDEN   = "hidden"
	SERVICE_FAMILY_PROVIDER = "provider"
)

type ServiceCreator func() ServiceBody
type SubServiceCreator func() interface{}

type ServiceSpecsUsecase interface {
	ListServices() map[string]ServiceInfos
	GetServiceIcon(string) ([]byte, error)
	GetServiceSpecs(reflect.Type) (*ServiceSpecs, error)
}
