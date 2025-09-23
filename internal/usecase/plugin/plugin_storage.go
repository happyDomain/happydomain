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

package plugin

import (
	"git.happydns.org/happyDomain/model"
)

type PluginStorage interface {
	// ListAllPluginConfigurations retrieves the list of known Providers.
	ListAllPluginConfigurations() (happydns.Iterator[happydns.PluginOptions], error)

	// ListPluginConfiguration retrieves all providers own by the given User.
	ListPluginConfiguration(string) ([]*happydns.PluginOptionsPositional, error)

	// GetPluginConfiguration retrieves the full Provider with the given identifier and owner.
	GetPluginConfiguration(string, *happydns.Identifier, *happydns.Identifier, *happydns.Identifier) ([]*happydns.PluginOptionsPositional, error)

	// UpdatePluginConfiguration updates the fields of the given Provider.
	UpdatePluginConfiguration(string, *happydns.Identifier, *happydns.Identifier, *happydns.Identifier, happydns.PluginOptions) error

	// DeletePluginConfiguration removes the given Provider from the database.
	DeletePluginConfiguration(string, *happydns.Identifier, *happydns.Identifier, *happydns.Identifier) error

	// ClearPluginConfigurations deletes all Providers present in the database.
	ClearPluginConfigurations() error
}
