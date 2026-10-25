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
	"io"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	svc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	"git.happydns.org/happyDomain/model"
)

func (s *Service) ImportZoneTemplate(user *happydns.User, input happydns.ZoneTemplateInput, origin string, zoneFile io.Reader) (*happydns.ZoneTemplate, error) {
	if err := checkInput(&input); err != nil {
		return nil, err
	}

	if strings.TrimSpace(origin) == "" {
		origin = happydns.ZoneTemplateOrigin
	} else {
		normalized, err := happydns.NormalizeDomainName(origin)
		if err != nil {
			return nil, happydns.ValidationError{Msg: fmt.Sprintf("%q is not a valid origin: %s", origin, err.Error())}
		}
		origin = strings.ToLower(normalized)
	}

	zp := dns.NewZoneParser(zoneFile, origin, "")
	// $INCLUDE would read files of the server.
	zp.SetIncludeAllowed(false)

	var rrs []happydns.Record
	for rr, ok := zp.Next(); ok; rr, ok = zp.Next() {
		// Kept, it would name a subdomain of the domains the template is
		// applied to.
		if !dns.IsSubDomain(origin, dns.CanonicalName(rr.Header().Name)) {
			return nil, happydns.ValidationError{Msg: fmt.Sprintf("Record %q is not part of the template, written for %s", rr.Header().String(), origin)}
		}
		rrs = append(rrs, rr)
	}
	if err := zp.Err(); err != nil {
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("Unable to read your zone file: %s", err.Error())}
	}

	services, defaultTTL, err := svc.AnalyzeZone(origin, rrs)
	if err != nil {
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("Unable to perform the analysis of your zone: %s", err.Error())}
	}

	// The names of the services are relative to origin: copied as they
	// are, they name the origin of the templates.
	services, err = s.copyServices(withoutOrigin(services), happydns.ZoneTemplateOrigin)
	if err != nil {
		return nil, err
	}
	if defaultTTL == 0 {
		defaultTTL = defaultTemplateTTL
	}

	return s.create(user, input, defaultTTL, services)
}

// checkService refuses the services a template cannot hold.
func checkService(subdomain happydns.Subdomain, service *happydns.Service) error {
	if service == nil || service.Service == nil {
		return happydns.ValidationError{Msg: "Unable to parse the given service."}
	}
	if isOrigin(subdomain, service) {
		return happydns.ValidationError{Msg: "A template cannot hold the origin of a zone: it belongs to the domain it is applied to."}
	}
	return nil
}

func normalizeSubdomain(subdomain happydns.Subdomain) happydns.Subdomain {
	if subdomain == "@" {
		return ""
	}
	return subdomain
}

func (s *Service) AddZoneTemplateService(user *happydns.User, id happydns.Identifier, subdomain happydns.Subdomain, service *happydns.Service) (*happydns.ZoneTemplate, error) {
	t, err := s.loadEditable(user, id)
	if err != nil {
		return nil, err
	}

	subdomain = normalizeSubdomain(subdomain)
	if err := checkService(subdomain, service); err != nil {
		return nil, err
	}
	if err := s.identify(subdomain, service, happydns.ZoneTemplateOrigin); err != nil {
		return nil, err
	}

	t.Services[subdomain] = append(t.Services[subdomain], service)
	return t, s.save(t)
}

func (s *Service) UpdateZoneTemplateService(user *happydns.User, id happydns.Identifier, subdomain happydns.Subdomain, serviceId happydns.Identifier, service *happydns.Service) (*happydns.ZoneTemplate, error) {
	t, err := s.loadEditable(user, id)
	if err != nil {
		return nil, err
	}

	subdomain = normalizeSubdomain(subdomain)
	if err := checkService(subdomain, service); err != nil {
		return nil, err
	}
	if _, err := s.validator.Validate(service.Service, subdomain, happydns.Origin(happydns.ZoneTemplateOrigin)); err != nil {
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("The service is invalid: %s", err.Error())}
	}
	// Keeps its identifier, as a service updated in a zone does.
	service.Id = serviceId
	service.Domain = string(subdomain)

	if err := t.Zone().EraseService(subdomain, serviceId, service); err != nil {
		return nil, happydns.NotFoundError{Msg: fmt.Sprintf("Unable to update the service: %s", err.Error())}
	}
	return t, s.save(t)
}

func (s *Service) DeleteZoneTemplateService(user *happydns.User, id happydns.Identifier, subdomain happydns.Subdomain, serviceId happydns.Identifier) (*happydns.ZoneTemplate, error) {
	t, err := s.loadEditable(user, id)
	if err != nil {
		return nil, err
	}

	if err := t.Zone().EraseService(normalizeSubdomain(subdomain), serviceId, nil); err != nil {
		return nil, happydns.NotFoundError{Msg: fmt.Sprintf("Unable to delete the service: %s", err.Error())}
	}
	return t, s.save(t)
}

// parseRecord reads a record, given as text, relative to the origin of the
// templates.
func parseRecord(record string) (happydns.Record, error) {
	rr, err := helpers.ParseRecord(record, happydns.ZoneTemplateOrigin)
	if err != nil {
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("Unable to parse the record %q: %s", record, err.Error())}
	}

	rr = helpers.RRRelative(rr, happydns.ZoneTemplateOrigin)
	if strings.HasSuffix(rr.Header().Name, ".") {
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("Record %q is not part of the template, written for %s", rr.Header().String(), happydns.ZoneTemplateOrigin)}
	}
	return rr, nil
}

// editRecords runs edit on the template seen as a zone, then saves it.
func (s *Service) editRecords(user *happydns.User, id happydns.Identifier, edit func(zone *happydns.Zone) error) (*happydns.ZoneTemplate, error) {
	t, err := s.loadEditable(user, id)
	if err != nil {
		return nil, err
	}

	zone := t.Zone()
	if err := edit(zone); err != nil {
		return nil, err
	}

	t.Services = zone.Services
	return t, s.save(t)
}

func (s *Service) AddZoneTemplateRecords(user *happydns.User, id happydns.Identifier, records []string) (*happydns.ZoneTemplate, error) {
	return s.editRecords(user, id, func(zone *happydns.Zone) error {
		for _, record := range records {
			rr, err := parseRecord(record)
			if err != nil {
				return err
			}
			if err := s.records.AddRecord(zone, happydns.ZoneTemplateOrigin, rr); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) DeleteZoneTemplateRecords(user *happydns.User, id happydns.Identifier, records []string) (*happydns.ZoneTemplate, error) {
	return s.editRecords(user, id, func(zone *happydns.Zone) error {
		for _, record := range records {
			rr, err := parseRecord(record)
			if err != nil {
				return err
			}
			if err := s.records.DeleteRecord(zone, happydns.ZoneTemplateOrigin, rr); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) UpdateZoneTemplateRecord(user *happydns.User, id happydns.Identifier, form happydns.UpdateRecordForm) (*happydns.ZoneTemplate, error) {
	return s.editRecords(user, id, func(zone *happydns.Zone) error {
		oldRR, err := parseRecord(form.OldRR)
		if err != nil {
			return err
		}
		newRR, err := parseRecord(form.NewRR)
		if err != nil {
			return err
		}

		if err := s.records.DeleteRecord(zone, happydns.ZoneTemplateOrigin, oldRR); err != nil {
			return err
		}
		return s.records.AddRecord(zone, happydns.ZoneTemplateOrigin, newRR)
	})
}
