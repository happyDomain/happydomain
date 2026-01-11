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

package database

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/services/providers/google"
)

func migrateFrom8(s *KVStorage) (err error) {
	migrateFrom7SvcType = make(map[string]func(json.RawMessage) (json.RawMessage, error))

	// google.GSuite
	migrateFrom7SvcType["google.GSuite"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var gsuite google.GSuite
		gsuite.Initialize()

		if code, ok := val["validationCode"]; ok {
			rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN MX 15 %s", code.(string)))
			if err != nil {
				return nil, err
			}
			if rr != nil {
				gsuite.ValidationMX = helpers.RRRelative(rr, "zZzZ").(*dns.MX)
			}
		}

		return json.Marshal(gsuite)
	}

	zones, err := s.ListAllZones()
	if err != nil {
		return err
	}

	for zones.Next() {
		zone := zones.Item()
		for _, svcs := range zone.Services {
			changed := false

			for i, svc := range svcs {
				if m, ok := migrateFrom7SvcType[svc.Type]; ok {
					svcs[i].Service, err = m(svc.Service)
					if err != nil {
						return err
					}

					changed = true
				}
			}

			if changed {
				// Save zone
				err = s.UpdateZoneMessage(zone)
				if err != nil {
					return err
				}
				log.Printf("Migrated zone %s", zone.Id.String())
			}
		}
	}

	return nil
}
