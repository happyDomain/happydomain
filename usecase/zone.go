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

package usecase

import (
	"fmt"
	"net/http"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/adapters"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type zoneUsecase struct {
	providerService happydns.ProviderUsecase
	serviceService  happydns.ServiceUsecase
	store           storage.Storage
}

func NewZoneUsecase(pu happydns.ProviderUsecase, su happydns.ServiceUsecase, store storage.Storage) happydns.ZoneUsecase {
	return &zoneUsecase{
		providerService: pu,
		serviceService:  su,
		store:           store,
	}
}

func (zu *zoneUsecase) AppendService(zone *happydns.Zone, subdomain, origin string, service *happydns.Service) error {
	if service.Service == nil {
		return happydns.InternalError{
			Err:        fmt.Errorf("Unable to parse the given service."),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	hash, err := zu.serviceService.ValidateService(service.Service, subdomain, origin)
	if err != nil {
		return err
	}

	service.Id = hash
	service.Domain = subdomain
	service.NbResources = service.Service.GetNbResources()
	service.Comment = service.Service.GenComment()

	zone.Services[subdomain] = append(zone.Services[subdomain], service)

	err = zu.store.UpdateZone(zone)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("Unable to UpdateZone in AppendService: %w", err),
			UserMessage: "Sorry, we are currently unable to update your zone. Please retry later.",
			HTTPStatus:  http.StatusInternalServerError,
		}
	}

	return nil
}

func (zu *zoneUsecase) CreateZone(zone *happydns.Zone) error {
	return zu.store.CreateZone(zone)
}

func (zu *zoneUsecase) DeleteService(zone *happydns.Zone, subdomain string, serviceid happydns.Identifier) error {
	err := zone.EraseService(subdomain, serviceid, nil)
	if err != nil {
		return happydns.InternalError{
			Err:        fmt.Errorf("unable to delete service: %w", err),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	zone.LastModified = time.Now()

	err = zu.store.UpdateZone(zone)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateZone in DeleteService: %w", err),
			UserMessage: "Sorry, we are currently unable to update your zone. Please retry later.",
			HTTPStatus:  http.StatusBadRequest,
		}
	}

	return nil
}

func (zu *zoneUsecase) DeleteZone(id happydns.Identifier) error {
	return zu.store.DeleteZone(id)
}

func (zu *zoneUsecase) DiffZones(domain *happydns.Domain, newzone *happydns.Zone, oldzoneid happydns.Identifier) ([]*happydns.Correction, error) {
	oldzone, err := zu.LoadZoneFromId(domain, oldzoneid)
	if err != nil {
		return nil, err
	}

	oldrecords, err := zu.GenerateRecords(domain, oldzone)
	if err != nil {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("unable to retrieve records for old zone: %w", err),
			HTTPStatus: http.StatusInternalServerError,
		}
	}

	newrecords, err := zu.GenerateRecords(domain, newzone)
	if err != nil {
		return nil, err
	}

	corrections, err := adapter.DNSControlDiffByRecord(oldrecords, newrecords, domain.DomainName)
	if err != nil {
		return nil, err
	}

	return corrections, nil
}

func (zu *zoneUsecase) FlattenZoneFile(domain *happydns.Domain, zone *happydns.Zone) (string, error) {
	records, err := zu.GenerateRecords(domain, zone)
	if err != nil {
		return "", happydns.InternalError{
			Err:        fmt.Errorf("unable to retrieve records for old zone: %w", err),
			HTTPStatus: http.StatusInternalServerError,
		}
	}

	var ret string

	for _, rr := range records {
		ret += rr.String() + "\n"
	}

	return ret, nil
}

func (zu *zoneUsecase) GenerateRecords(domain *happydns.Domain, zone *happydns.Zone) (rrs []happydns.Record, err error) {
	var svc_rrs []happydns.Record

	for subdomain, svcs := range zone.Services {
		if subdomain == "" || subdomain == "@" {
			subdomain = domain.DomainName
		} else {
			subdomain += "." + domain.DomainName
		}

		for _, svc := range svcs {
			var ttl uint32
			if svc.Ttl == 0 {
				ttl = zone.DefaultTTL
			} else {
				ttl = svc.Ttl
			}

			svc_rrs, err = svc.Service.GetRecords(subdomain, ttl, domain.DomainName)
			if err != nil {
				return
			}
			rrs = append(rrs, svc_rrs...)
		}

		// Ensure SOA is the first record
		for i, rr := range rrs {
			if rr.Header().Rrtype == dns.TypeSOA {
				rrs[0], rrs[i] = rrs[i], rrs[0]
				break
			}
		}
	}

	return
}

func (zu *zoneUsecase) GetZone(id happydns.Identifier) (*happydns.Zone, error) {
	zonemsg, err := zu.store.GetZone(id)
	if err != nil {
		return nil, err
	}

	return ParseZone(zonemsg)
}

func (zu *zoneUsecase) GetZoneCorrections(user *happydns.User, domain *happydns.Domain, zone *happydns.Zone) ([]*happydns.Correction, error) {
	provider, err := zu.providerService.GetUserProvider(user, domain.IdProvider)
	if err != nil {
		return nil, err
	}

	records, err := zu.GenerateRecords(domain, zone)
	if err != nil {
		return nil, err
	}

	return zu.providerService.GetZoneCorrections(provider, domain, records)
}

func (zu *zoneUsecase) GetZoneMeta(id happydns.Identifier) (*happydns.ZoneMeta, error) {
	zonemsg, err := zu.store.GetZone(id)
	if err != nil {
		return nil, err
	}

	return &zonemsg.ZoneMeta, nil
}

func (zu *zoneUsecase) LoadZoneFromId(domain *happydns.Domain, zoneid happydns.Identifier) (*happydns.Zone, error) {
	// Check that the zoneid exists in the domain history
	if !domain.HasZone(zoneid) {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("zone not found: %q", zoneid.String()),
			HTTPStatus: http.StatusNotFound,
		}
	}

	zmsg, err := zu.store.GetZone(zoneid)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to retrieve user zone (id=%s): %w", zoneid.String(), err),
			UserMessage: fmt.Sprintf("zone not found: %q", zoneid.String()),
			HTTPStatus:  http.StatusNotFound,
		}
	}

	return ParseZone(zmsg)
}

func ParseZone(msg *happydns.ZoneMessage) (*happydns.Zone, error) {
	var z happydns.Zone

	z.ZoneMeta = msg.ZoneMeta
	z.Services = map[string][]*happydns.Service{}

	for subdn, svcs := range msg.Services {
		for _, svc := range svcs {
			s, err := ParseService(svc)
			if err != nil {
				return nil, fmt.Errorf("under %q, unable to parse service %q: %w", subdn, svc, err)
			}

			z.Services[subdn] = append(z.Services[subdn], s)
		}

	}

	return &z, nil
}

func (zu *zoneUsecase) UpdateService(zone *happydns.Zone, subdomain string, serviceid happydns.Identifier, newservice *happydns.Service) error {
	err := zone.EraseService(subdomain, serviceid, newservice)
	if err != nil {
		return happydns.InternalError{
			Err:        fmt.Errorf("unable to delete service: %w", err),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	zone.LastModified = time.Now()

	err = zu.store.UpdateZone(zone)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateZone in UpdateService: %w", err),
			UserMessage: "Sorry, we are currently unable to update your zone. Please retry later.",
			HTTPStatus:  http.StatusBadRequest,
		}
	}

	return nil
}

func (zu *zoneUsecase) UpdateZone(id happydns.Identifier, upd func(*happydns.Zone)) error {
	zone, err := zu.GetZone(id)
	if err != nil {
		return err
	}

	upd(zone)

	if !zone.Id.Equals(id) {
		return happydns.InternalError{
			Err:        fmt.Errorf("you cannot change the zone identifier"),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	err = zu.store.UpdateZone(zone)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateZone in UpdateZone: %w", err),
			UserMessage: "Sorry, we are currently unable to update your zone. Please retry later.",
		}
	}

	return nil
}
