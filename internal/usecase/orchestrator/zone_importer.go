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
	"time"

	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type ZoneImporterUsecase struct {
	domainUpdater DomainUpdater
	zoneCreator   *zoneUC.CreateZoneUsecase
}

func NewZoneImporterUsecase(domainUpdater DomainUpdater, zoneCreator *zoneUC.CreateZoneUsecase) *ZoneImporterUsecase {
	return &ZoneImporterUsecase{
		domainUpdater: domainUpdater,
		zoneCreator:   zoneCreator,
	}
}

func (uc *ZoneImporterUsecase) Import(user *happydns.User, domain *happydns.Domain, rrs []happydns.Record) (*happydns.Zone, error) {
	services, defaultTTL, err := svcs.AnalyzeZone(domain.DomainName, rrs)
	if err != nil {
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("unable to perform the analysis of your zone: %s", err.Error())}
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
