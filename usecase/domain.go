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
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type domainUsecase struct {
	domainLogService happydns.DomainLogUsecase
	providerService  happydns.ProviderUsecase
	store            storage.Storage
	zoneService      happydns.ZoneUsecase
}

func NewDomainUsecase(store storage.Storage, domainLogService happydns.DomainLogUsecase, providerService happydns.ProviderUsecase, zoneService happydns.ZoneUsecase) happydns.DomainUsecase {
	return &domainUsecase{
		domainLogService: domainLogService,
		providerService:  providerService,
		store:            store,
		zoneService:      zoneService,
	}
}

func (du *domainUsecase) ApplyZoneCorrection(user *happydns.User, domain *happydns.Domain, zone *happydns.Zone, form *happydns.ApplyZoneForm) (*happydns.Zone, error) {
	provider, err := du.getUserProvider(user, domain)
	if err != nil {
		return nil, err
	}

	records, err := du.zoneService.GenerateRecords(domain, zone)
	if err != nil {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("unable to retrieve records for zone: %w", err),
			HTTPStatus: http.StatusInternalServerError,
		}
	}

	nbcorrections := len(form.WantedCorrections)
	corrections, err := du.providerService.GetZoneCorrections(provider, domain, records)
	if err != nil {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("unable to compute domain corrections: %w", err),
			HTTPStatus: http.StatusInternalServerError,
		}
	}

	var errs error
corrections:
	for i, cr := range corrections {
		for ic, wc := range form.WantedCorrections {
			if wc.Equals(cr.Id) {
				log.Printf("%s: apply correction: %s", domain.DomainName, cr.Msg)
				err := cr.F()

				if err != nil {
					log.Printf("%s: unable to apply correction: %s", domain.DomainName, err.Error())
					du.store.CreateDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("Failed record update (%s): %s", cr.Msg, err.Error())))
					errs = errors.Join(errs, fmt.Errorf("%s: %w", cr.Msg, err))
					// Stop the zone update if we didn't change it yet
					if i == 0 {
						break corrections
					}
				} else {
					form.WantedCorrections = append(form.WantedCorrections[:ic], form.WantedCorrections[ic+1:]...)
				}
				break
			}
		}
	}

	if errs != nil {
		du.store.CreateDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("Failed zone publishing (%s): %d corrections were not applied due to errors.", zone.Id.String(), nbcorrections)))
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("unable to update the zone: %w", errs),
			HTTPStatus: http.StatusBadRequest,
		}
	} else if len(form.WantedCorrections) > 0 {
		du.store.CreateDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("Failed zone publishing (%s): %d corrections were not applied.", zone.Id.String(), nbcorrections)))
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("unable to perform the following changes: %s", form.WantedCorrections),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	du.store.CreateDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ACK, fmt.Sprintf("Zone published (%s), %d corrections applied with success", zone.Id.String(), nbcorrections)))

	// Create a new zone in history for futher updates
	newZone := zone.DerivateNew()
	err = du.store.CreateZone(newZone)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to CreateZone: %w", err),
			HTTPStatus:  http.StatusInternalServerError,
			UserMessage: "Sorry, we are unable to create the zone now.",
		}
	}

	domain.ZoneHistory = append(
		[]happydns.Identifier{newZone.Id}, domain.ZoneHistory...)

	err = du.store.UpdateDomain(domain)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateDomain: %w", err),
			HTTPStatus:  http.StatusInternalServerError,
			UserMessage: "Sorry, we are unable to create the zone now.",
		}
	}

	// Commit changes in previous zone
	now := time.Now()
	zone.ZoneMeta.IdAuthor = user.Id
	zone.CommitMsg = &form.CommitMsg
	zone.ZoneMeta.CommitDate = &now
	zone.ZoneMeta.Published = &now

	zone.LastModified = time.Now()

	err = du.store.UpdateZone(zone)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateZone: %w", err),
			HTTPStatus:  http.StatusInternalServerError,
			UserMessage: "Sorry, we are unable to create the zone now.",
		}
	}

	return newZone, nil
}

func (du *domainUsecase) ActionOnEditableZone(user *happydns.User, domain *happydns.Domain, zone *happydns.Zone, act func(zone *happydns.Zone) error) (*happydns.Zone, error) {
	var err error
	newZone := zone

	if zone.CommitDate != nil || zone.Published != nil {
		// Create a new zone if the current one is in archived state
		newZone = zone.DerivateNew()

		err = du.zoneService.CreateZone(newZone)
		if err != nil {
			return nil, happydns.InternalError{
				Err:         fmt.Errorf("unable to CreateZone in importZone: %s\n", err),
				UserMessage: "Sorry, we are unable to create your zone.",
			}
		}

		domain.ZoneHistory = append(
			[]happydns.Identifier{newZone.Id}, domain.ZoneHistory...)

		err = du.UpdateDomain(domain.Id, user, func(dn *happydns.Domain) {
			dn.ZoneHistory = domain.ZoneHistory
		})
		if err != nil {
			return nil, happydns.InternalError{
				Err:         fmt.Errorf("unable to UpdateDomain in importZone: %s\n", err),
				UserMessage: "Sorry, we are unable to create your zone.",
			}
		}
	}

	err = act(newZone)
	if err != nil {
		return nil, err
	}

	return newZone, nil
}

func (du *domainUsecase) AppendZoneService(user *happydns.User, domain *happydns.Domain, zone *happydns.Zone, subdomain string, origin string, service *happydns.Service) (*happydns.Zone, error) {
	return du.ActionOnEditableZone(user, domain, zone, func(zone *happydns.Zone) error {
		return du.zoneService.AppendService(zone, subdomain, origin, service)
	})
}

func (du *domainUsecase) CreateDomain(user *happydns.User, uz *happydns.Domain) error {
	if len(uz.DomainName) <= 2 {
		return happydns.InternalError{
			Err:        fmt.Errorf("the given domain is invalid"),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	uz.Owner = user.Id
	uz.DomainName = dns.Fqdn(uz.DomainName)

	if _, ok := dns.IsDomainName(uz.DomainName); !ok {
		return happydns.InternalError{
			Err:        fmt.Errorf("%q is not a valid domain name", uz.DomainName),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	provider, err := du.providerService.GetUserProvider(user, uz.IdProvider)
	if err != nil {
		return happydns.InternalError{
			Err:        fmt.Errorf("unable to find the provider."),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	if err = du.providerService.TestDomainExistence(provider, uz.DomainName); err != nil {
		return happydns.InternalError{
			Err:        err,
			HTTPStatus: http.StatusNotFound,
		}
	}

	if err := du.store.CreateDomain(uz); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to CreateDomain: %s", err),
			UserMessage: "Sorry, we are unable to create your domain now.",
		}
	}

	// Add a log entry
	if du.domainLogService != nil {
		du.domainLogService.AppendDomainLog(uz, happydns.NewDomainLog(user, happydns.LOG_INFO, fmt.Sprintf("Domain name %s added.", uz.DomainName)))
	}

	return nil
}

func (du *domainUsecase) DeleteDomain(did happydns.Identifier) error {
	err := du.store.DeleteDomain(did)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to DeleteDomain: %w", err),
			UserMessage: fmt.Sprintf("unable to delete your domain: %s", err.Error()),
		}
	}

	return nil
}

func (du *domainUsecase) DeleteZoneService(user *happydns.User, domain *happydns.Domain, zone *happydns.Zone, subdomain string, serviceid happydns.Identifier) (*happydns.Zone, error) {
	return du.ActionOnEditableZone(user, domain, zone, func(zone *happydns.Zone) error {
		return du.zoneService.DeleteService(zone, subdomain, serviceid)
	})
}

func (du *domainUsecase) ExtendsDomainWithZoneMeta(domain *happydns.Domain) (*happydns.DomainWithZoneMetadata, error) {
	var errs error
	ret := map[string]*happydns.ZoneMeta{}

	for _, zm := range domain.ZoneHistory {
		zoneMeta, err := du.zoneService.GetZoneMeta(zm)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("unable to retrieve zone meta history for %q: %w", domain.DomainName, err))
		} else {
			ret[zm.String()] = zoneMeta
		}
	}

	return &happydns.DomainWithZoneMetadata{
		Domain:   domain,
		ZoneMeta: ret,
	}, errs
}

func (du *domainUsecase) GetUserDomain(user *happydns.User, did happydns.Identifier) (*happydns.Domain, error) {
	domain, err := du.store.GetDomain(did)
	if err != nil {
		return nil, err
	}

	if !user.Id.Equals(domain.Owner) {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("domain not found"),
			HTTPStatus: http.StatusNotFound,
		}
	}

	return domain, nil
}

func (du *domainUsecase) GetUserDomainByFQDN(user *happydns.User, fqdn string) ([]*happydns.Domain, error) {
	return du.store.GetDomainByDN(user, fqdn)
}

func (du *domainUsecase) getUserProvider(user *happydns.User, domain *happydns.Domain) (*happydns.Provider, error) {
	provider, err := du.providerService.GetUserProvider(user, domain.IdProvider)
	if err != nil {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("unable to find your provider: %w", err),
			HTTPStatus: http.StatusNotFound,
		}
	}

	return provider, nil
}

func (du *domainUsecase) ImportZone(user *happydns.User, domain *happydns.Domain, rrs []happydns.Record) (*happydns.Zone, error) {
	services, defaultTTL, err := svcs.AnalyzeZone(domain.DomainName, rrs)
	if err != nil {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("unable to perform the analysis of your zone: %w", err),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	now := time.Now()
	commit := fmt.Sprintf("Initial zone fetch from %s", domain.DomainName)
	if len(domain.ZoneHistory) > 0 {
		commit = fmt.Sprintf("Zone fetched from %s", domain.DomainName)
	}

	myZone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{
			IdAuthor:     domain.Owner,
			DefaultTTL:   defaultTTL,
			LastModified: now,
			CommitMsg:    &commit,
			CommitDate:   &now,
			Published:    &now,
		},
		Services: services,
	}

	// Create history zone
	err = du.zoneService.CreateZone(myZone)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to CreateZone in importZone: %s\n", err),
			UserMessage: "Sorry, we are unable to create your zone.",
		}
	}
	domain.ZoneHistory = append(
		[]happydns.Identifier{myZone.Id}, domain.ZoneHistory...)

	// Save domain modifications
	err = du.UpdateDomain(domain.Id, user, func(dn *happydns.Domain) {
		dn.ZoneHistory = domain.ZoneHistory
	})
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateDomain in importZone: %s\n", err),
			UserMessage: "Sorry, we are unable to create your zone.",
		}
	}

	return myZone, nil
}

func (du *domainUsecase) ListUserDomains(user *happydns.User) ([]*happydns.Domain, error) {
	domains, err := du.store.GetDomains(user)
	if err != nil {
		return nil, fmt.Errorf("an error occurs when trying to GetUserDomains: %s", err.Error())
	}

	if len(domains) == 0 {
		return []*happydns.Domain{}, nil
	}

	return domains, nil
}

func (du *domainUsecase) PublishZone(*happydns.User, *happydns.Domain, *happydns.Zone) ([]*happydns.Correction, error) {
	return nil, nil
}

func (du *domainUsecase) RetrieveRemoteZone(user *happydns.User, domain *happydns.Domain) (*happydns.Zone, error) {
	provider, err := du.getUserProvider(user, domain)
	if err != nil {
		return nil, err
	}

	zone, err := du.providerService.RetrieveZone(provider, domain.DomainName)
	if err != nil {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("unable to retrieve the zone from server: %w", err),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	// import
	myZone, err := du.ImportZone(user, domain, zone)
	if err != nil {
		return nil, err
	}

	if du.domainLogService != nil {
		du.domainLogService.AppendDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_INFO, fmt.Sprintf("Zone imported from provider API: %s", myZone.Id.String())))
	}

	return myZone, nil
}

func (du *domainUsecase) UpdateDomain(domainid happydns.Identifier, user *happydns.User, upd func(*happydns.Domain)) error {
	domain, err := du.GetUserDomain(user, domainid)
	if err != nil {
		return err
	}

	upd(domain)
	//domain.ModifiedOn = time.Now()

	if !domain.Id.Equals(domainid) {
		return happydns.InternalError{
			Err:        fmt.Errorf("you cannot change the domain identifier"),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	err = du.store.UpdateDomain(domain)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateDomain in UpdateDomain: %w", err),
			UserMessage: "Sorry, we are currently unable to update your domain. Please retry later.",
		}
	}

	// Add a log entry
	if du.domainLogService != nil {
		du.domainLogService.AppendDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_INFO, fmt.Sprintf("Domain name %s properties changed.", domain.DomainName)))
	}

	return nil
}

func (du *domainUsecase) UpdateZoneService(user *happydns.User, domain *happydns.Domain, zone *happydns.Zone, subdomain string, serviceid happydns.Identifier, service *happydns.Service) (*happydns.Zone, error) {
	return du.ActionOnEditableZone(user, domain, zone, func(zone *happydns.Zone) error {
		return du.zoneService.UpdateService(zone, subdomain, serviceid, service)
	})
}
