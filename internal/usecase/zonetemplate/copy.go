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
	"encoding/json"
	"fmt"

	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
)

// originTypes are the services holding the SOA and the NS records of the
// apex of a zone. They belong to the domain and its provider: templates
// never hold them, and applying one keeps those of the zone.
var originTypes = []string{"abstract.Origin", "abstract.NSOnlyOrigin"}

func isOrigin(subdomain happydns.Subdomain, s *happydns.Service) bool {
	if subdomain != "" && subdomain != "@" {
		return false
	}
	for _, t := range originTypes {
		if s.Type == t {
			return true
		}
	}
	return false
}

func hasOrigin(services map[happydns.Subdomain][]*happydns.Service) bool {
	for subdomain, svcs := range services {
		for _, s := range svcs {
			if isOrigin(subdomain, s) {
				return true
			}
		}
	}
	return false
}

// withoutOrigin is services but for the origin of the zone.
func withoutOrigin(services map[happydns.Subdomain][]*happydns.Service) map[happydns.Subdomain][]*happydns.Service {
	ret := map[happydns.Subdomain][]*happydns.Service{}
	for subdomain, svcs := range services {
		for _, s := range svcs {
			if !isOrigin(subdomain, s) {
				ret[subdomain] = append(ret[subdomain], s)
			}
		}
	}
	return ret
}

func parseServices(msgs map[happydns.Subdomain][]*happydns.ServiceMessage) (map[happydns.Subdomain][]*happydns.Service, error) {
	services := map[happydns.Subdomain][]*happydns.Service{}
	for subdomain, svcs := range msgs {
		for _, msg := range svcs {
			s, err := serviceUC.ParseService(msg)
			if err != nil {
				return nil, fmt.Errorf("under %q, unable to parse service %s: %w", subdomain, msg.Type, err)
			}
			services[subdomain] = append(services[subdomain], s)
		}
	}
	return services, nil
}

// copyServices makes a deep copy of services for origin. The names of the
// services are relative to their origin: copied as they are, they name the
// domain they are applied to. Each copy is checked to give records under
// origin, and identified by their fingerprint.
func (s *Service) copyServices(services map[happydns.Subdomain][]*happydns.Service, origin string) (map[happydns.Subdomain][]*happydns.Service, error) {
	ret := map[happydns.Subdomain][]*happydns.Service{}
	for subdomain, svcs := range services {
		for _, orig := range svcs {
			c, err := s.copyService(subdomain, orig, origin)
			if err != nil {
				return nil, err
			}
			ret[subdomain] = append(ret[subdomain], c)
		}
	}
	return ret, nil
}

func (s *Service) copyService(subdomain happydns.Subdomain, orig *happydns.Service, origin string) (*happydns.Service, error) {
	raw, err := json.Marshal(orig)
	if err != nil {
		return nil, happydns.InternalError{Err: fmt.Errorf("unable to copy the service %s under %q: %w", orig.Type, subdomain, err), UserMessage: "Sorry, we are unable to copy the services."}
	}

	var msg happydns.ServiceMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, happydns.InternalError{Err: fmt.Errorf("unable to copy the service %s under %q: %w", orig.Type, subdomain, err), UserMessage: "Sorry, we are unable to copy the services."}
	}
	// A copy has never been published, and belongs to where it goes.
	msg.PropagatedAt = nil
	msg.OwnerId = nil

	c, err := serviceUC.ParseService(&msg)
	if err != nil {
		return nil, happydns.InternalError{Err: fmt.Errorf("unable to copy the service %s under %q: %w", orig.Type, subdomain, err), UserMessage: "Sorry, we are unable to copy the services."}
	}

	if err := s.identify(subdomain, c, origin); err != nil {
		return nil, err
	}
	return c, nil
}

// identify checks that service gives records under origin, and identifies it
// by their fingerprint, as a service added to a zone is.
func (s *Service) identify(subdomain happydns.Subdomain, service *happydns.Service, origin string) error {
	if service.Service == nil {
		return happydns.ValidationError{Msg: "Unable to parse the given service."}
	}

	hash, err := s.validator.Validate(service.Service, subdomain, happydns.Origin(origin))
	if err != nil {
		return happydns.ValidationError{Msg: fmt.Sprintf("The service %s of %s is invalid: %s", service.Type, displaySubdomain(subdomain), err.Error())}
	}

	service.Id = hash
	service.Domain = string(subdomain)
	return nil
}

func displaySubdomain(subdomain happydns.Subdomain) string {
	if subdomain == "" {
		return "the apex"
	}
	return fmt.Sprintf("%q", string(subdomain))
}
