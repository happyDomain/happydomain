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
	"log"

	"git.happydns.org/happyDomain/model"
)

func (tu *tidyUpUsecase) TidyZones(dropInvalid bool) error {
	iterdn, err := tu.store.ListAllDomains()
	if err != nil {
		return err
	}
	defer iterdn.Close()

	var referencedZones []happydns.Identifier
	if err = iterateTidy(iterdn, dropInvalid, func(domain *happydns.Domain) error {
		referencedZones = append(referencedZones, domain.ZoneHistory...)
		return nil
	}); err != nil {
		return err
	}

	iter, err := tu.store.ListAllZones()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(zone *happydns.ZoneMessage) error {
		foundZone := false
		for _, zid := range referencedZones {
			if zid.Equals(zone.Id) {
				foundZone = true
				break
			}
		}

		if !foundZone {
			// Drop orphan zones
			log.Printf("Deleting orphan zone: %s\n", zone.Id.String())
			if err := iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}
