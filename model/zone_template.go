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
	"io"
	"time"
)

// ZoneTemplateApplyMode tells how a template is applied to a zone.
type ZoneTemplateApplyMode string

const (
	// ZoneTemplateMerge adds the services of the template to those of the
	// zone, replacing those that cannot stand next to them.
	ZoneTemplateMerge ZoneTemplateApplyMode = "merge"
	// ZoneTemplateOverwrite replaces the services of the zone by those of
	// the template, but for the origin of the zone, kept as it is.
	ZoneTemplateOverwrite ZoneTemplateApplyMode = "overwrite"
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

// ZoneTemplateInput is what a user tells about a template.
type ZoneTemplateInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ZoneTemplateCreationForm creates a template, empty or from a zone of a
// domain of the user.
type ZoneTemplateCreationForm struct {
	ZoneTemplateInput
	// DomainId and ZoneId designate the zone to save as a template. A
	// template without them starts empty.
	DomainId *Identifier `json:"domain_id,omitempty" swaggertype:"string"`
	ZoneId   *Identifier `json:"zone_id,omitempty" swaggertype:"string"`
}

// ApplyZoneTemplateForm tells which template to apply to a zone, and how.
type ApplyZoneTemplateForm struct {
	TemplateId Identifier            `json:"template_id" swaggertype:"string" binding:"required"`
	Mode       ZoneTemplateApplyMode `json:"mode" binding:"required" enums:"merge,overwrite"`
}

// ZoneTemplateUsecase manages zone templates. A nil user stands for an
// administrator, who manages the global templates.
type ZoneTemplateUsecase interface {
	// ListZoneTemplates lists the templates of the user, then the global
	// ones; only the global ones for a nil user.
	ListZoneTemplates(user *User) ([]*ZoneTemplate, error)
	GetZoneTemplate(user *User, id Identifier) (*ZoneTemplate, error)
	// CreateZoneTemplate stores an empty template.
	CreateZoneTemplate(user *User, input ZoneTemplateInput) (*ZoneTemplate, error)
	// CreateZoneTemplateFromZone saves the services of a zone of domain,
	// but for its origin, as a template.
	CreateZoneTemplateFromZone(user *User, input ZoneTemplateInput, domain *Domain, zone *Zone) (*ZoneTemplate, error)
	// ImportZoneTemplate makes a template from a zone file written for
	// origin, ZoneTemplateOrigin when empty.
	ImportZoneTemplate(user *User, input ZoneTemplateInput, origin string, zoneFile io.Reader) (*ZoneTemplate, error)
	// UpdateZoneTemplate changes the name and the description of a
	// template.
	UpdateZoneTemplate(user *User, id Identifier, input ZoneTemplateInput) (*ZoneTemplate, error)
	DeleteZoneTemplate(user *User, id Identifier) error

	AddZoneTemplateService(user *User, id Identifier, subdomain Subdomain, service *Service) (*ZoneTemplate, error)
	UpdateZoneTemplateService(user *User, id Identifier, subdomain Subdomain, serviceId Identifier, service *Service) (*ZoneTemplate, error)
	DeleteZoneTemplateService(user *User, id Identifier, subdomain Subdomain, serviceId Identifier) (*ZoneTemplate, error)

	// AddZoneTemplateRecords adds records, given as text relative to
	// ZoneTemplateOrigin, to its services.
	AddZoneTemplateRecords(user *User, id Identifier, records []string) (*ZoneTemplate, error)
	DeleteZoneTemplateRecords(user *User, id Identifier, records []string) (*ZoneTemplate, error)
	UpdateZoneTemplateRecord(user *User, id Identifier, form UpdateRecordForm) (*ZoneTemplate, error)

	// ApplyZoneTemplate applies a template to the zone of domain, deriving
	// an editable zone first when that one is already published. It
	// returns the zone changed.
	ApplyZoneTemplate(user *User, domain *Domain, zone *Zone, form ApplyZoneTemplateForm) (*Zone, error)
}
