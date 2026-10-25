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
	"bytes"
	"fmt"
	"slices"

	"github.com/miekg/dns"

	svc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	"git.happydns.org/happyDomain/model"
)

func (s *Service) ApplyZoneTemplate(user *happydns.User, domain *happydns.Domain, zone *happydns.Zone, form happydns.ApplyZoneTemplateForm) (*happydns.Zone, error) {
	if form.Mode != happydns.ZoneTemplateMerge && form.Mode != happydns.ZoneTemplateOverwrite {
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("Unknown way to apply a template: %q.", form.Mode)}
	}

	t, err := s.load(user, form.TemplateId)
	if err != nil {
		return nil, err
	}

	services, err := s.copyServices(t.Services, domain.DomainName)
	if err != nil {
		return nil, err
	}
	// Services without TTL of their own followed the default TTL of the
	// template, which the zone may not share: pin it.
	if t.DefaultTTL != zone.DefaultTTL {
		for _, svcs := range services {
			for _, svc := range svcs {
				if svc.Ttl == 0 {
					svc.Ttl = t.DefaultTTL
				}
			}
		}
	}

	return s.zones.ActionOnEditableZone(user, domain, zone, func(z *happydns.Zone) error {
		if form.Mode == happydns.ZoneTemplateOverwrite {
			z.Services = overwrite(z.Services, services)
		} else {
			z.Services = s.merge(z.Services, services, domain.DomainName)
		}
		z.LastModified = s.nowFn()

		if err := s.zoneStore.UpdateZone(z); err != nil {
			return happydns.InternalError{
				Err:         fmt.Errorf("unable to UpdateZone in ApplyZoneTemplate: %w", err),
				UserMessage: "Sorry, we are currently unable to update your zone. Please retry later.",
			}
		}
		return nil
	})
}

// overwrite keeps the origin of zone only, and adds the services of the
// template.
func overwrite(zone, template map[happydns.Subdomain][]*happydns.Service) map[happydns.Subdomain][]*happydns.Service {
	ret := map[happydns.Subdomain][]*happydns.Service{}
	for subdomain, svcs := range zone {
		for _, svc := range svcs {
			if isOrigin(subdomain, svc) {
				ret[subdomain] = append(ret[subdomain], svc)
			}
		}
	}
	for subdomain, svcs := range template {
		ret[subdomain] = append(ret[subdomain], svcs...)
	}
	return ret
}

// merge adds the services of the template to those of zone, without
// changing the slices of zone, which a derived zone shares with the
// published one. A service of the template already in the zone is not added
// again; any other replaces, on its name, the services of the zone:
//   - of its type, when there is only one of its type per name;
//   - of a type it cannot stand next to, or that cannot stand next to it;
//   - all of its name, when it has to be alone there, as a CNAME;
//   - the one alone there, unless it can stand next to it.
//
// The origin of the zone is always kept.
func (s *Service) merge(zone, template map[happydns.Subdomain][]*happydns.Service, origin string) map[happydns.Subdomain][]*happydns.Service {
	ret := map[happydns.Subdomain][]*happydns.Service{}
	for subdomain, svcs := range zone {
		ret[subdomain] = slices.Clone(svcs)
	}

	for subdomain, svcs := range template {
		current := ret[subdomain]
		for _, add := range svcs {
			if slices.ContainsFunc(current, func(existing *happydns.Service) bool {
				return s.sameService(subdomain, existing, add, origin)
			}) {
				continue
			}

			addR := s.restrictionsOf(add.Type)
			addAlone := addR.Alone || isCNAME(subdomain, add, origin)
			current = slices.DeleteFunc(current, func(existing *happydns.Service) bool {
				if isOrigin(subdomain, existing) {
					return false
				}
				existingR := s.restrictionsOf(existing.Type)
				switch {
				case existing.Type == add.Type && addR.Single:
					return true
				case slices.Contains(addR.ExclusiveRR, existing.Type) || slices.Contains(existingR.ExclusiveRR, add.Type):
					return true
				case addAlone && !existingR.NearAlone:
					return true
				case (existingR.Alone || isCNAME(subdomain, existing, origin)) && !addR.NearAlone:
					return true
				}
				return false
			})
			current = append(current, add)
		}
		ret[subdomain] = current
	}
	return ret
}

// registeredRestrictions are the restrictions the type of service declares.
func registeredRestrictions(svctype string) happydns.ServiceRestrictions {
	if s, ok := (*svc.ListServices())[svc.CanonicalServiceType(svctype)]; ok {
		return s.Infos.Restrictions
	}
	return happydns.ServiceRestrictions{}
}

// sameService tells whether a and b give the same records.
func (s *Service) sameService(subdomain happydns.Subdomain, a, b *happydns.Service, origin string) bool {
	if a.Type != b.Type {
		return false
	}
	ha, err := s.validator.Validate(a.Service, subdomain, happydns.Origin(origin))
	if err != nil {
		return false
	}
	hb, err := s.validator.Validate(b.Service, subdomain, happydns.Origin(origin))
	if err != nil {
		return false
	}
	return bytes.Equal(ha, hb)
}

// isCNAME tells whether the service puts a CNAME on its subdomain, which
// leaves no room for other records there.
func isCNAME(subdomain happydns.Subdomain, svc *happydns.Service, origin string) bool {
	if subdomain == "" || svc.Service == nil {
		return false
	}
	rrs, err := svc.Service.GetRecords(string(subdomain), 0, origin)
	if err != nil {
		return false
	}
	// The records of a service are named relative to its subdomain: "" or
	// "@" is the subdomain itself.
	for _, rr := range rrs {
		if rr.Header().Rrtype == dns.TypeCNAME && (rr.Header().Name == "" || rr.Header().Name == "@") {
			return true
		}
	}
	return false
}
