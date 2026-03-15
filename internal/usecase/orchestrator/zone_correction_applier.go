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

	domainlogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

// ZoneCorrectionApplierUsecase applies a user-selected subset of zone
// corrections to the provider and, on success, snapshots the zone into the
// domain history.  It embeds ZoneCorrectionListerUsecase to compute the full
// diff before filtering to the requested corrections.
type ZoneCorrectionApplierUsecase struct {
	*ZoneCorrectionListerUsecase
	appendDomainLog domainlogUC.DomainLogAppender
	domainUpdater   DomainUpdater
	zoneCreator     *zoneUC.CreateZoneUsecase
	zoneUpdater     *zoneUC.UpdateZoneUsecase
	clock           func() time.Time
}

// NewZoneCorrectionApplierUsecase creates a ZoneCorrectionApplierUsecase with
// the given dependencies.  The lister is embedded so that Apply can compute
// the full correction diff in a single call.
func NewZoneCorrectionApplierUsecase(
	appendDomainLog domainlogUC.DomainLogAppender,
	domainUpdater DomainUpdater,
	lister *ZoneCorrectionListerUsecase,
	zoneCreator *zoneUC.CreateZoneUsecase,
	zoneUpdater *zoneUC.UpdateZoneUsecase,
) *ZoneCorrectionApplierUsecase {
	return &ZoneCorrectionApplierUsecase{
		ZoneCorrectionListerUsecase: lister,
		appendDomainLog:             appendDomainLog,
		domainUpdater:               domainUpdater,
		zoneCreator:                 zoneCreator,
		zoneUpdater:                 zoneUpdater,
		clock:                       time.Now,
	}
}

// Apply executes the corrections listed in form.WantedCorrections against the
// provider.  Each correction is matched by ID against the live diff; unmatched
// or failed corrections abort the operation.  On success the applied zone is
// committed (publish date, author, commit message recorded) and a new derived
// zone is prepended to the domain's history so future edits start from the
// published state.  Returns the newly created zone or a descriptive error.
func (uc *ZoneCorrectionApplierUsecase) Apply(
	user *happydns.User,
	domain *happydns.Domain,
	zone *happydns.Zone,
	form *happydns.ApplyZoneForm,
) (*happydns.Zone, error) {
	corrections, _, err := uc.List(user, domain, zone)
	if err != nil {
		return nil, err
	}

	nbcorrections := len(form.WantedCorrections)

	// Track which wanted corrections were successfully applied, without mutating the input.
	matched := make([]bool, len(form.WantedCorrections))
	var errs error
	appliedCount := 0

corrections:
	for _, cr := range corrections {
		for i, wc := range form.WantedCorrections {
			if matched[i] {
				continue
			}
			if wc.Equals(cr.Id) {
				log.Printf("%s: apply correction: %s", domain.DomainName, cr.Msg)
				corrErr := cr.F()

				if corrErr != nil {
					log.Printf("%s: unable to apply correction: %s", domain.DomainName, corrErr.Error())
					uc.appendDomainLog.AppendDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("Failed record update (%s): %s", cr.Msg, corrErr.Error())))
					errs = errors.Join(errs, fmt.Errorf("%s: %w", cr.Msg, corrErr))
					// Stop if no corrections have been successfully applied yet
					if appliedCount == 0 {
						break corrections
					}
				} else {
					appliedCount++
					matched[i] = true
				}
				break
			}
		}
	}

	unmatchedCount := 0
	for _, m := range matched {
		if !m {
			unmatchedCount++
		}
	}

	if errs != nil {
		uc.appendDomainLog.AppendDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("Failed zone publishing (%s): %d of %d corrections applied, errors occurred.", zone.Id.String(), appliedCount, nbcorrections)))
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("unable to update the zone (%d of %d corrections applied): %s", appliedCount, nbcorrections, errs.Error())}
	} else if unmatchedCount > 0 {
		uc.appendDomainLog.AppendDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("Failed zone publishing (%s): %d corrections were not found in the current diff.", zone.Id.String(), unmatchedCount)))
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("unable to perform %d corrections that were not found in the current diff", unmatchedCount)}
	}

	uc.appendDomainLog.AppendDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ACK, fmt.Sprintf("Zone published (%s), %d corrections applied with success", zone.Id.String(), nbcorrections)))

	// Create a new zone in history for further updates
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
			UserMessage: "Sorry, we are unable to update the domain history now.",
		}
	}

	// Commit changes in previous zone
	now := uc.clock()
	err = uc.zoneUpdater.Update(zone.ZoneMeta.Id, func(zone *happydns.Zone) {
		zone.ZoneMeta.IdAuthor = user.Id
		zone.CommitMsg = &form.CommitMsg
		zone.ZoneMeta.CommitDate = &now
		zone.ZoneMeta.Published = &now
		zone.LastModified = now
	})
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to commit zone changes: %w", err),
			UserMessage: "Sorry, we are unable to commit the zone changes now.",
		}
	}

	return newZone, nil
}
