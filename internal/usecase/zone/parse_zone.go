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

package zone

import (
	"fmt"

	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
)

func ParseZone(msg *happydns.ZoneMessage) (*happydns.Zone, error) {
	var z happydns.Zone

	z.ZoneMeta = msg.ZoneMeta
	z.Services = map[happydns.Subdomain][]*happydns.Service{}

	for subdn, svcs := range msg.Services {
		for _, svc := range svcs {
			s, err := serviceUC.ParseService(svc)
			if err != nil {
				return nil, fmt.Errorf("under %q, unable to parse service %q: %w", subdn, svc, err)
			}

			z.Services[subdn] = append(z.Services[subdn], s)
		}

	}

	return &z, nil
}
