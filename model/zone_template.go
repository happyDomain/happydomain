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

package happydns

import (
	"time"
)

// ZoneTemplateMeta holds what describes a zone template, apart from its
// services.
type ZoneTemplateMeta struct {
	Id Identifier `json:"id" swaggertype:"string" readonly:"true" binding:"required"`

	// UserId is the owner of the template; a template without owner is a
	// global one, set by an administrator and offered to every user.
	UserId *Identifier `json:"userId,omitempty" swaggertype:"string" readonly:"true"`

	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`

	// DefaultTTL is the TTL of the records without one of their own.
	DefaultTTL uint32 `json:"default_ttl" binding:"required"`

	LastModified time.Time `json:"last_modified" format:"date-time" readonly:"true" binding:"required"`
}

// IsGlobal tells whether the template belongs to no one, and is offered to
// every user.
func (t *ZoneTemplateMeta) IsGlobal() bool {
	return t.UserId == nil || t.UserId.IsEmpty()
}

// ZoneTemplateOrigin is the domain the services of every template are shown
// under, and the records given as text are relative to. Being reserved, no
// record of a real zone points to it. The names of the services are relative
// to it, so applied to a domain they name that domain; free texts are kept as
// they are.
const ZoneTemplateOrigin = "example.com."

// ZoneTemplate is a zone a user saved to fill other domains with.
type ZoneTemplate struct {
	ZoneTemplateMeta
	Services map[Subdomain][]*Service `json:"services" binding:"required"`
}

// ZoneTemplateMessage is the intermediate struct for parsing zone templates.
type ZoneTemplateMessage struct {
	ZoneTemplateMeta
	Services map[Subdomain][]*ServiceMessage `json:"services"`
}

// Zone presents the template as a zone, for the use cases editing zones.
// Both share the same services.
func (t *ZoneTemplate) Zone() *Zone {
	return &Zone{
		ZoneMeta: ZoneMeta{
			Id:           t.Id,
			DefaultTTL:   t.DefaultTTL,
			LastModified: t.LastModified,
		},
		Services: t.Services,
	}
}
