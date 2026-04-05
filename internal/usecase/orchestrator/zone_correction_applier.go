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
	"context"
	"fmt"
	"log"
	"time"

	adapter "git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/provider"
	svc "git.happydns.org/happyDomain/internal/service"
	domainlogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/abstract"
)

// ZoneCorrectionApplierUsecase applies a user-selected subset of zone
// corrections to the provider and, on success, creates a published snapshot
// in the domain history. The WIP zone at ZoneHistory[0] is never modified.
type ZoneCorrectionApplierUsecase struct {
	*ZoneCorrectionListerUsecase
	appendDomainLog    domainlogUC.DomainLogAppender
	domainUpdater      DomainUpdater
	zoneCreator        *zoneUC.CreateZoneUsecase
	zoneGetter         *zoneUC.GetZoneUsecase
	zoneRetriever      ZoneRetriever
	zoneUpdater        *zoneUC.UpdateZoneUsecase
	schedulerNotifier  happydns.SchedulerDomainNotifier
	clock              func() time.Time
}

// NewZoneCorrectionApplierUsecase creates a ZoneCorrectionApplierUsecase with
// the given dependencies. The lister is embedded so that Apply can compute
// the full correction diff in a single call.
func NewZoneCorrectionApplierUsecase(
	appendDomainLog domainlogUC.DomainLogAppender,
	domainUpdater DomainUpdater,
	lister *ZoneCorrectionListerUsecase,
	zoneCreator *zoneUC.CreateZoneUsecase,
	zoneGetter *zoneUC.GetZoneUsecase,
	zoneRetriever ZoneRetriever,
	zoneUpdater *zoneUC.UpdateZoneUsecase,
) *ZoneCorrectionApplierUsecase {
	return &ZoneCorrectionApplierUsecase{
		ZoneCorrectionListerUsecase: lister,
		appendDomainLog:             appendDomainLog,
		domainUpdater:               domainUpdater,
		zoneCreator:                 zoneCreator,
		zoneGetter:                  zoneGetter,
		zoneRetriever:               zoneRetriever,
		zoneUpdater:                 zoneUpdater,
		clock:                       time.Now,
	}
}

// computeExecutableCorrections computes the executable corrections for the
// given selection. It performs the diff, builds the target record set, and asks
// the provider what it would execute to reach that target state.
func (uc *ZoneCorrectionApplierUsecase) computeExecutableCorrections(
	ctx context.Context,
	user *happydns.User,
	domain *happydns.Domain,
	zone *happydns.Zone,
	wantedCorrections []happydns.Identifier,
) (execCorrections []*happydns.Correction, targetRecords []happydns.Record, providerRecords []happydns.Record, nbDiffs int, err error) {
	// Step 1: Compute the diff and get provider/WIP records.
	corrections, providerRecords, _, nbDiffs, err := uc.listWithRecords(ctx, user, domain, zone)
	if err != nil {
		return nil, nil, nil, nbDiffs, err
	}

	// Step 2: Build target records from selected corrections.
	targetRecords = adapter.BuildTargetRecords(providerRecords, corrections, wantedCorrections)

	// Step 3: Get executable corrections from the provider for the target state.
	provider, err := uc.providerService.GetUserProvider(ctx, user, domain.ProviderId)
	if err != nil {
		return nil, nil, nil, nbDiffs, err
	}

	execCorrections, nbDiffs, err = uc.zoneCorrector.ListZoneCorrections(ctx, provider, domain, targetRecords)
	if err != nil {
		return nil, nil, nil, nbDiffs, fmt.Errorf("unable to compute executable corrections: %w", err)
	}

	return execCorrections, targetRecords, providerRecords, nbDiffs, nil
}

// Prepare computes the executable corrections for the given selection without
// applying them. This lets the user see exactly what the provider will execute
// before confirming.
func (uc *ZoneCorrectionApplierUsecase) Prepare(
	ctx context.Context,
	user *happydns.User,
	domain *happydns.Domain,
	zone *happydns.Zone,
	form *happydns.PrepareZoneForm,
) (*happydns.PrepareZoneResponse, error) {
	execCorrections, _, _, nbDiffs, err := uc.computeExecutableCorrections(ctx, user, domain, zone, form.WantedCorrections)
	if err != nil {
		return nil, err
	}

	return &happydns.PrepareZoneResponse{
		Corrections: execCorrections,
		NbDiffs:     nbDiffs,
	}, nil
}

// Apply executes the selected corrections against the provider and creates a
// published snapshot zone inserted at ZoneHistory[1] (after the WIP zone at
// position 0). The WIP zone is never modified.
//
// Flow:
//  1. Compute the diff (corrections + provider/WIP records)
//  2. Build the target record set from selected corrections
//  3. Ask the provider to compute executable corrections for the target state
//  4. Execute all returned corrections
//  5. Create a published snapshot zone from the target records
//  6. Insert the snapshot at ZoneHistory[1]
//  7. Return the published snapshot zone
func (uc *ZoneCorrectionApplierUsecase) Apply(
	ctx context.Context,
	user *happydns.User,
	domain *happydns.Domain,
	zone *happydns.Zone,
	form *happydns.ApplyZoneForm,
) (*happydns.Zone, error) {
	executableCorrections, targetRecords, providerRecords, _, err := uc.computeExecutableCorrections(ctx, user, domain, zone, form.WantedCorrections)
	if err != nil {
		return nil, err
	}

	// Step 4: Execute all corrections.
	appliedCount := 0
	for _, cr := range executableCorrections {
		log.Printf("%s: apply correction: %s", domain.DomainName, cr.Msg)
		if corrErr := cr.F(); corrErr != nil {
			log.Printf("%s: unable to apply correction: %s", domain.DomainName, corrErr.Error())
			if logErr := uc.appendDomainLog.AppendDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("Failed record update (%s): %s", cr.Msg, corrErr.Error()))); logErr != nil {
				log.Printf("unable to append domain log for %s: %s", domain.DomainName, logErr.Error())
			}
			if appliedCount == 0 {
				return nil, happydns.ValidationError{Msg: fmt.Sprintf("unable to apply correction: %s", corrErr.Error())}
			}
			if logErr := uc.appendDomainLog.AppendDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("Failed zone publishing (%s): %d of %d corrections applied, errors occurred.", zone.Id.String(), appliedCount, len(executableCorrections)))); logErr != nil {
				log.Printf("unable to append domain log for %s: %s", domain.DomainName, logErr.Error())
			}
			return nil, happydns.ValidationError{Msg: fmt.Sprintf("unable to update the zone (%d of %d corrections applied): %s", appliedCount, len(executableCorrections), corrErr.Error())}
		}
		appliedCount++
	}

	if logErr := uc.appendDomainLog.AppendDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_ACK, fmt.Sprintf("Zone published (%s), %d corrections applied with success", zone.Id.String(), appliedCount))); logErr != nil {
		log.Printf("unable to append domain log for %s: %s", domain.DomainName, logErr.Error())
	}

	// Step 4b: If provider manages SOA serial, re-fetch to get the actual published state.
	publishedRecords := targetRecords
	refetched := false
	provider, provErr := uc.providerService.GetUserProvider(ctx, user, domain.ProviderId)
	if provErr == nil && providerReg.ProviderHasCapability(provider, "manages-soa-serial") {
		fetched, fetchErr := uc.zoneRetriever.RetrieveZone(ctx, provider, domain.DomainName)
		if fetchErr != nil {
			log.Printf("%s: unable to re-fetch zone after deploy, using target records: %s", domain.DomainName, fetchErr)
		} else {
			publishedRecords = fetched
			refetched = true
		}
	}

	// Step 5: Create a published snapshot zone from published records.
	services, defaultTTL, err := svc.AnalyzeZone(domain.DomainName, publishedRecords)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to analyze target zone: %w", err),
			UserMessage: "Sorry, we are unable to analyze the published zone.",
		}
	}

	// Carry over metadata from WIP zone.
	if zone.Services != nil {
		zoneUC.ReassociateMetadata(zone.Services, services, domain.DomainName, defaultTTL)
	}

	// Also carry over metadata from the previous published zone if available.
	if len(domain.ZoneHistory) > 1 {
		prevZone, prevErr := uc.zoneGetter.Get(domain.ZoneHistory[1])
		if prevErr != nil {
			log.Printf("ReassociateMetadata: unable to load previous zone %s: %s (metadata will not be transferred)", domain.ZoneHistory[1], prevErr)
		} else {
			zoneUC.ReassociateMetadata(prevZone.Services, services, domain.DomainName, defaultTTL)
		}
	}

	now := uc.clock()

	// Compute propagation times for changed services on the snapshot.
	SetPropagationTimes(services, providerRecords, domain.DomainName, defaultTTL, now)

	snapshot := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{
			IdAuthor:     user.Id,
			DefaultTTL:   defaultTTL,
			LastModified: now,
			CommitMsg:    &form.CommitMsg,
			CommitDate:   &now,
			Published:    &now,
			ParentZone:   zone.ParentZone,
		},
		Services: services,
	}

	err = uc.zoneCreator.Create(snapshot)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to CreateZone for published snapshot: %w", err),
			UserMessage: "Sorry, we are unable to create the published zone snapshot.",
		}
	}

	// Update the parent zone of the WIP zone
	zone.ParentZone = &snapshot.Id

	// Step 5b: If we re-fetched, update the WIP zone's Origin SOA serial to match.
	if refetched {
		if newSerial, ok := extractOriginSOASerial(snapshot); ok {
			if updateErr := uc.zoneUpdater.Update(zone.Id, func(z *happydns.Zone) {
				if services, exists := z.Services[""]; exists {
					for _, s := range services {
						if s.Type == "abstract.Origin" {
							if origin, ok := s.Service.(*abstract.Origin); ok && origin.SOA != nil {
								origin.SOA.Serial = newSerial
							}
						}
					}
				}
			}); updateErr != nil {
				log.Printf("%s: unable to update WIP zone SOA serial: %s", domain.DomainName, updateErr)
			}
		}
	}

	// Step 6: Insert snapshot at ZoneHistory[1] (after WIP at position 0).
	err = uc.domainUpdater.Update(domain.Id, user, func(domain *happydns.Domain) {
		if len(domain.ZoneHistory) == 0 {
			domain.ZoneHistory = []happydns.Identifier{snapshot.Id}
		} else {
			newHistory := make([]happydns.Identifier, 0, len(domain.ZoneHistory)+1)
			newHistory = append(newHistory, domain.ZoneHistory[0])
			newHistory = append(newHistory, snapshot.Id)
			newHistory = append(newHistory, domain.ZoneHistory[1:]...)
			domain.ZoneHistory = newHistory
		}
	})
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateDomain: %w", err),
			UserMessage: "Sorry, we are unable to update the domain history now.",
		}
	}

	// Update propagation times on the WIP zone as well.
	if updateErr := uc.zoneUpdater.Update(zone.Id, func(wipZone *happydns.Zone) {
		SetPropagationTimes(wipZone.Services, providerRecords, domain.DomainName, wipZone.DefaultTTL, now)
	}); updateErr != nil {
		log.Printf("%s: unable to update WIP zone propagation times: %s", domain.DomainName, updateErr)
	}

	if uc.schedulerNotifier != nil {
		uc.schedulerNotifier.NotifyDomainChange(domain)
	}

	return snapshot, nil
}

// extractOriginSOASerial extracts the SOA serial from the Origin service
// at the zone apex, if present.
func extractOriginSOASerial(zone *happydns.Zone) (uint32, bool) {
	if services, exists := zone.Services[""]; exists {
		for _, s := range services {
			if s.Type == "abstract.Origin" {
				if origin, ok := s.Service.(*abstract.Origin); ok && origin.SOA != nil {
					return origin.SOA.Serial, true
				}
			}
		}
	}
	return 0, false
}
