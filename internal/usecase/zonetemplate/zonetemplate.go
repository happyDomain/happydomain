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
	"fmt"
	"slices"
	"strings"
	"time"

	"git.happydns.org/happyDomain/model"
)

const (
	// defaultTemplateTTL is the TTL of the records of a template made from
	// nothing, nor from a zone telling one.
	defaultTemplateTTL = 3600

	maxNameLength        = 100
	maxDescriptionLength = 1000
)

// ServiceValidator checks that a service gives records, and fingerprints
// them.
type ServiceValidator interface {
	Validate(svc happydns.ServiceBody, subdomain happydns.Subdomain, origin happydns.Origin) ([]byte, error)
}

// RecordEditor adds records to the services of a zone, and removes some.
type RecordEditor interface {
	AddRecord(zone *happydns.Zone, origin string, record happydns.Record) error
	DeleteRecord(zone *happydns.Zone, origin string, record happydns.Record) error
}

// Service manages the zone templates.
type Service struct {
	store      ZoneTemplateStorage
	validator  ServiceValidator
	records    RecordEditor
	maxPerUser int

	nowFn func() time.Time
}

var _ happydns.ZoneTemplateUsecase = (*Service)(nil)

// NewService builds the zone template service. maxPerUser caps the number of
// templates of a user, 0 meaning unlimited; it does not apply to the global
// templates.
func NewService(store ZoneTemplateStorage, validator ServiceValidator, records RecordEditor, maxPerUser int) *Service {
	return &Service{
		store:      store,
		validator:  validator,
		records:    records,
		maxPerUser: maxPerUser,
		nowFn:      time.Now,
	}
}

// ownerOf is the owner of the templates a user creates: none for an
// administrator, whose templates are the global ones.
func ownerOf(user *happydns.User) *happydns.Identifier {
	if user == nil {
		return nil
	}
	id := user.Id
	return &id
}

func parseTemplate(msg *happydns.ZoneTemplateMessage) (*happydns.ZoneTemplate, error) {
	services, err := parseServices(msg.Services)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to parse the zone template %s: %w", msg.Id.String(), err),
			UserMessage: "Sorry, we are unable to read this template.",
		}
	}
	return &happydns.ZoneTemplate{ZoneTemplateMeta: msg.ZoneTemplateMeta, Services: services}, nil
}

func parseTemplates(msgs []*happydns.ZoneTemplateMessage) ([]*happydns.ZoneTemplate, error) {
	templates := make([]*happydns.ZoneTemplate, 0, len(msgs))
	for _, msg := range msgs {
		t, err := parseTemplate(msg)
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	slices.SortFunc(templates, func(a, b *happydns.ZoneTemplate) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return templates, nil
}

func (s *Service) ListZoneTemplates(user *happydns.User) ([]*happydns.ZoneTemplate, error) {
	// Never nil: without user nor template, answer an empty list, not null.
	templates := []*happydns.ZoneTemplate{}

	if user != nil {
		msgs, err := s.store.ListZoneTemplatesByUser(user.Id)
		if err != nil {
			return nil, happydns.InternalError{Err: fmt.Errorf("unable to list the zone templates: %w", err), UserMessage: "Unable to list your templates."}
		}
		if templates, err = parseTemplates(msgs); err != nil {
			return nil, err
		}
	}

	msgs, err := s.store.ListGlobalZoneTemplates()
	if err != nil {
		return nil, happydns.InternalError{Err: fmt.Errorf("unable to list the global zone templates: %w", err), UserMessage: "Unable to list the templates."}
	}
	globals, err := parseTemplates(msgs)
	if err != nil {
		return nil, err
	}

	return append(templates, globals...), nil
}

// load finds a template the user can read: one of theirs, or a global one.
func (s *Service) load(user *happydns.User, id happydns.Identifier) (*happydns.ZoneTemplate, error) {
	msg, err := s.store.GetZoneTemplate(id)
	if err != nil {
		return nil, err
	}
	// Not found rather than forbidden: do not leak the templates of others.
	if !msg.IsGlobal() && (user == nil || !msg.UserId.Equals(user.Id)) {
		return nil, happydns.ErrZoneTemplateNotFound
	}
	return parseTemplate(msg)
}

// loadEditable finds a template the user can change: one of theirs, or a
// global one for an administrator.
func (s *Service) loadEditable(user *happydns.User, id happydns.Identifier) (*happydns.ZoneTemplate, error) {
	t, err := s.load(user, id)
	if err != nil {
		return nil, err
	}
	if t.IsGlobal() && user != nil {
		return nil, happydns.ForbiddenError{Msg: "This template is offered to everyone: only the administrators can change it."}
	}
	return t, nil
}

func (s *Service) GetZoneTemplate(user *happydns.User, id happydns.Identifier) (*happydns.ZoneTemplate, error) {
	return s.load(user, id)
}

// checkInput validates what describes a template, normalizing its origin,
// example.com. when empty.
func checkInput(input *happydns.ZoneTemplateInput) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)

	if input.Name == "" {
		return happydns.ValidationError{Msg: "A template needs a name."}
	}
	if len(input.Name) > maxNameLength {
		return happydns.ValidationError{Msg: fmt.Sprintf("The name of a template cannot exceed %d characters.", maxNameLength)}
	}
	if len(input.Description) > maxDescriptionLength {
		return happydns.ValidationError{Msg: fmt.Sprintf("The description of a template cannot exceed %d characters.", maxDescriptionLength)}
	}

	return nil
}

// checkQuota refuses a new template to a user who has too many.
func (s *Service) checkQuota(user *happydns.User) error {
	if user == nil || s.maxPerUser <= 0 {
		return nil
	}

	templates, err := s.store.ListZoneTemplatesByUser(user.Id)
	if err != nil {
		return happydns.InternalError{Err: fmt.Errorf("unable to count the zone templates: %w", err), UserMessage: "Unable to create your template."}
	}
	if len(templates) >= s.maxPerUser {
		return happydns.ForbiddenError{Msg: fmt.Sprintf("You cannot keep more than %d templates.", s.maxPerUser)}
	}
	return nil
}

func (s *Service) create(user *happydns.User, input happydns.ZoneTemplateInput, defaultTTL uint32, services map[happydns.Subdomain][]*happydns.Service) (*happydns.ZoneTemplate, error) {
	if err := s.checkQuota(user); err != nil {
		return nil, err
	}

	t := &happydns.ZoneTemplate{
		ZoneTemplateMeta: happydns.ZoneTemplateMeta{
			UserId:       ownerOf(user),
			Name:         input.Name,
			Description:  input.Description,
			DefaultTTL:   defaultTTL,
			LastModified: s.nowFn(),
		},
		Services: services,
	}

	if err := s.store.CreateZoneTemplate(t); err != nil {
		return nil, happydns.InternalError{Err: fmt.Errorf("unable to create the zone template: %w", err), UserMessage: "Sorry, we are unable to create your template."}
	}
	return t, nil
}

func (s *Service) CreateZoneTemplate(user *happydns.User, input happydns.ZoneTemplateInput) (*happydns.ZoneTemplate, error) {
	if err := checkInput(&input); err != nil {
		return nil, err
	}

	return s.create(user, input, defaultTemplateTTL, map[happydns.Subdomain][]*happydns.Service{})
}

func (s *Service) CreateZoneTemplateFromZone(user *happydns.User, input happydns.ZoneTemplateInput, domain *happydns.Domain, zone *happydns.Zone) (*happydns.ZoneTemplate, error) {
	if err := checkInput(&input); err != nil {
		return nil, err
	}

	// The names of the services are relative to the domain: copied as they
	// are, they name the origin of the templates.
	services, err := s.copyServices(withoutOrigin(zone.Services), happydns.ZoneTemplateOrigin)
	if err != nil {
		return nil, err
	}

	defaultTTL := zone.DefaultTTL
	if defaultTTL == 0 {
		defaultTTL = defaultTemplateTTL
	}

	return s.create(user, input, defaultTTL, services)
}

func (s *Service) UpdateZoneTemplate(user *happydns.User, id happydns.Identifier, input happydns.ZoneTemplateInput) (*happydns.ZoneTemplate, error) {
	t, err := s.loadEditable(user, id)
	if err != nil {
		return nil, err
	}

	if err := checkInput(&input); err != nil {
		return nil, err
	}

	t.Name, t.Description = input.Name, input.Description
	return t, s.save(t)
}

func (s *Service) DeleteZoneTemplate(user *happydns.User, id happydns.Identifier) error {
	if _, err := s.loadEditable(user, id); err != nil {
		return err
	}

	if err := s.store.DeleteZoneTemplate(id); err != nil {
		return happydns.InternalError{Err: fmt.Errorf("unable to delete the zone template %s: %w", id.String(), err), UserMessage: "Sorry, we are unable to delete this template."}
	}
	return nil
}

// save stores the changes made to a template.
func (s *Service) save(t *happydns.ZoneTemplate) error {
	if hasOrigin(t.Services) {
		return happydns.ValidationError{Msg: "A template cannot hold the origin of a zone, its SOA and NS records: they belong to the domain it is applied to."}
	}

	// Removing the last record of a subdomain leaves it without service.
	for subdomain, svcs := range t.Services {
		if len(svcs) == 0 {
			delete(t.Services, subdomain)
		}
	}

	t.LastModified = s.nowFn()
	if err := s.store.UpdateZoneTemplate(t); err != nil {
		return happydns.InternalError{Err: fmt.Errorf("unable to update the zone template %s: %w", t.Id.String(), err), UserMessage: "Sorry, we are unable to save your template."}
	}
	return nil
}
