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

package orchestrator

import (
	"errors"
	"fmt"
	"log"
	"time"

	domainUC "git.happydns.org/happyDomain/internal/usecase/domain"
	domainlogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

type ZoneCorrectionApplierUsecase struct {
	appendDomainLog *domainlogUC.CreateDomainLogUsecase
	domainUpdater   *domainUC.UpdateDomainUsecase
	getProvider     *providerUC.GetProviderUsecase
	listRecords     *zoneUC.ListRecordsUsecase
	zoneCorrector   *providerUC.ZoneCorrectorUsecase
	zoneCreator     *zoneUC.CreateZoneUsecase
	zoneUpdater     *zoneUC.UpdateZoneUsecase
}

func NewZoneCorrectionApplierUsecase(
	appendDomainLog *domainlogUC.CreateDomainLogUsecase,
	domainUpdater *domainUC.UpdateDomainUsecase,
	getProvider *providerUC.GetProviderUsecase,
	listRecords *zoneUC.ListRecordsUsecase,
	zoneCorrector *providerUC.ZoneCorrectorUsecase,
	zoneCreator *zoneUC.CreateZoneUsecase,
	zoneUpdater *zoneUC.UpdateZoneUsecase,
) *ZoneCorrectionApplierUsecase {
	return &ZoneCorrectionApplierUsecase{
		appendDomainLog: appendDomainLog,
		domainUpdater:   domainUpdater,
		getProvider:     getProvider,
		listRecords:     listRecords,
		zoneCorrector:   zoneCorrector,
		zoneCreator:     zoneCreator,
		zoneUpdater:     zoneUpdater,
	}
}

func (uc *ZoneCorrectionApplierUsecase) Apply(user *happydns.User, domain *happydns.Domain, zone *happydns.Zone, form *happydns.ApplyZoneForm) (*happydns.Zone, error) {
	provider, err := uc.getProvider.Get(user, domain.ProviderId)
	if err != nil {
		return nil, err
	}

	records, err := uc.listRecords.List(domain, zone)
	if err != nil {
		return nil, happydns.InternalError{
			Err: fmt.Errorf("unable to retrieve records for zone: %w", err),
		}
	}

	nbcorrections := len(form.WantedCorrections)
	corrections, err := uc.zoneCorrector.List(provider, domain, records)
	if err != nil {
		return nil, happydns.InternalError{
			Err: fmt.Errorf("unable to compute domain corrections: %w", err),
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
					uc.appendDomainLog.Create(domain, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("Failed record update (%s): %s", cr.Msg, err.Error())))
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
		uc.appendDomainLog.Create(domain, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("Failed zone publishing (%s): %d corrections were not applied due to errors.", zone.Id.String(), nbcorrections)))
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("unable to update the zone: %s", errs.Error())}
	} else if len(form.WantedCorrections) > 0 {
		uc.appendDomainLog.Create(domain, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("Failed zone publishing (%s): %d corrections were not applied.", zone.Id.String(), nbcorrections)))
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("unable to perform the following changes: %s", form.WantedCorrections)}
	}

	uc.appendDomainLog.Create(domain, happydns.NewDomainLog(user, happydns.LOG_ACK, fmt.Sprintf("Zone published (%s), %d corrections applied with success", zone.Id.String(), nbcorrections)))

	// Create a new zone in history for futher updates
	newZone := zone.DerivateNew()
	err = uc.zoneCreator.Create(newZone)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to CreateZone: %w", err),
			UserMessage: "Sorry, we are unable to create the zone now.",
		}
	}

	err = uc.domainUpdater.Update(domain.Id, user, func(domain *happydns.Domain) {
		domain.ZoneHistory = append(
			[]happydns.Identifier{newZone.Id}, domain.ZoneHistory...)
	})
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateDomain: %w", err),
			UserMessage: "Sorry, we are unable to create the zone now.",
		}
	}

	// Commit changes in previous zone
	err = uc.zoneUpdater.Update(zone.ZoneMeta.Id, func(zone *happydns.Zone) {
		now := time.Now()
		zone.ZoneMeta.IdAuthor = user.Id
		zone.CommitMsg = &form.CommitMsg
		zone.ZoneMeta.CommitDate = &now
		zone.ZoneMeta.Published = &now

		zone.LastModified = time.Now()
	})
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateZone: %w", err),
			UserMessage: "Sorry, we are unable to create the zone now.",
		}
	}

	return newZone, nil
}

func (uc *ZoneCorrectionApplierUsecase) List(user *happydns.User, domain *happydns.Domain, zone *happydns.Zone) ([]*happydns.Correction, error) {
	provider, err := uc.getProvider.Get(user, domain.ProviderId)
	if err != nil {
		return nil, err
	}

	records, err := uc.listRecords.List(domain, zone)
	if err != nil {
		return nil, err
	}

	return uc.zoneCorrector.List(provider, domain, records)
}
