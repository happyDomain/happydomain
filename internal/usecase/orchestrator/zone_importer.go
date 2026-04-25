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
	"fmt"
	"log"
	"time"

	svc "git.happydns.org/happyDomain/internal/service"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

// ZoneImporterUsecase converts a flat slice of DNS records into a structured
// happyDomain zone, preserving metadata from the previous zone when available,
// and persists the result as the newest entry in the domain's zone history.
type ZoneImporterUsecase struct {
	domainUpdater DomainUpdater
	zoneCreator   *zoneUC.CreateZoneUsecase
	zoneGetter    *zoneUC.GetZoneUsecase
}

// NewZoneImporterUsecase creates a ZoneImporterUsecase with the given domain
// updater, zone creator, and zone getter.
func NewZoneImporterUsecase(domainUpdater DomainUpdater, zoneCreator *zoneUC.CreateZoneUsecase, zoneGetter *zoneUC.GetZoneUsecase) *ZoneImporterUsecase {
	return &ZoneImporterUsecase{
		domainUpdater: domainUpdater,
		zoneCreator:   zoneCreator,
		zoneGetter:    zoneGetter,
	}
}

// Import analyzes rrs into services, optionally carries over metadata from the
// domain's most recent zone, persists the new zone, and prepends its ID to the
// domain's history.  Returns the created zone or an error.
func (uc *ZoneImporterUsecase) Import(user *happydns.User, domain *happydns.Domain, rrs []happydns.Record) (*happydns.Zone, error) {
	services, defaultTTL, err := svc.AnalyzeZone(domain.DomainName, rrs)
	if err != nil {
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("unable to perform the analysis of your zone: %s", err.Error())}
	}

	if len(domain.ZoneHistory) > 0 {
		prevZone, err := uc.zoneGetter.Get(domain.ZoneHistory[0])
		if err != nil {
			log.Printf("ReassociateMetadata: unable to load previous zone %s: %s (metadata will not be transferred)", domain.ZoneHistory[0].String(), err)
		} else {
			zoneUC.ReassociateMetadata(prevZone.Services, services, domain.DomainName, defaultTTL)
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
	err = uc.zoneCreator.Create(myZone)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to CreateZone in importZone: %s\n", err),
			UserMessage: "Sorry, we are unable to create your zone.",
		}
	}
	domain.ZoneHistory = append(
		[]happydns.Identifier{myZone.Id}, domain.ZoneHistory...)

	// Save domain modifications
	err = uc.domainUpdater.Update(domain.Id, user, func(dn *happydns.Domain) {
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
