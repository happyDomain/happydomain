// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

package database

import (
	"fmt"
	"log"
	"strings"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

func migrateFrom4(s *LevelDBStorage) (err error) {
	err = migrateFrom4_orphanrecords(s)
	if err != nil {
		return
	}

	err = s.Tidy()
	if err != nil {
		return
	}

	return
}

func migrateFrom4_orphanrecords(s *LevelDBStorage) (err error) {
	iter := s.search("domain.zone-")
	for iter.Next() {
		var tmpid string
		fmt.Sscanf(string(iter.Key()), "domain.zone-%s", &tmpid)

		var id happydns.Identifier
		id, err = happydns.NewIdentifierFromString(tmpid)
		if err != nil {
			return fmt.Errorf("unable to determine identifier of %s: %w", iter.Key(), err)
		}

		var zone *happydns.Zone
		zone, err = s.GetZone(id)
		if err != nil {
			return fmt.Errorf("%s: %w", iter.Key(), err)
		}

		changed := false
		for _, zServices := range zone.Services {
			for _, svc := range zServices {
				if orphan, ok := svc.Service.(*svcs.Orphan); ok {
					tmp := strings.Fields(orphan.RR)

					orphan.Type = tmp[0]
					orphan.RR = strings.TrimSpace(orphan.RR[len(tmp[0]):])

					changed = true
				}
			}
		}

		if changed {
			err = s.UpdateZone(zone)
			if err != nil {
				return fmt.Errorf("unable to write %s: %w", iter.Key(), err)
			}
			log.Printf("Migrating v3 -> v4: %s (contains Orphan)...", iter.Key())
		}
	}

	return nil
}
