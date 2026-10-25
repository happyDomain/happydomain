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

package zonetemplate

import (
	"git.happydns.org/happyDomain/model"
)

// ZoneTemplateStorage stores the zone templates of the users, and the global
// ones.
type ZoneTemplateStorage interface {
	// ListAllZoneTemplates lists every template, of every user and global,
	// for the administrative passes.
	ListAllZoneTemplates() (happydns.Iterator[happydns.ZoneTemplateMessage], error)
	ListZoneTemplatesByUser(userId happydns.Identifier) ([]*happydns.ZoneTemplateMessage, error)
	// ListGlobalZoneTemplates lists the templates without owner.
	ListGlobalZoneTemplates() ([]*happydns.ZoneTemplateMessage, error)
	GetZoneTemplate(id happydns.Identifier) (*happydns.ZoneTemplateMessage, error)
	// CreateZoneTemplate stores a new template, giving it its identifier.
	CreateZoneTemplate(t *happydns.ZoneTemplate) error
	// UpdateZoneTemplate stores the changes made to t, whose owner must not
	// change: the index of the templates of a user is left as it is.
	UpdateZoneTemplate(t *happydns.ZoneTemplate) error
	DeleteZoneTemplate(id happydns.Identifier) error
	// RestoreZoneTemplate stores t under its identifier, replacing any
	// template stored under it, as a backup restore does.
	RestoreZoneTemplate(t *happydns.ZoneTemplateMessage) error
	// TidyZoneTemplateIndexes removes stale index entries.
	TidyZoneTemplateIndexes() error
}
