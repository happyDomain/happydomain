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

package storage

import (
	"fmt"
)

// StorageEngine defines an interface that handle configuration throught custom
// flag.
type StorageEngine string

func (i *StorageEngine) String() string {
	return string(*i)
}

func (i *StorageEngine) Set(value string) (err error) {
	if _, ok := StorageEngines[value]; !ok {
		return fmt.Errorf("unexistant storage engine: please select one between: %v", GetStorageEngines())
	}
	*i = StorageEngine(value)
	return nil
}

// StorageInstanciation is a function that a Storage implementation
// has to expose in order to be usable in configuration.
type StorageInstanciation func() (Storage, error)

// StorageEngines lists all Storage implementations declared, with a
// way to instanciate automatically each.
var StorageEngines = map[string]StorageInstanciation{}

// GetStorageEngines returns all declared Storage implementation.
func GetStorageEngines() (se []string) {
	for k := range StorageEngines {
		se = append(se, string(k))
	}

	return
}
