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

package orchestrator_test

import (
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/usecase/orchestrator"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

func TestZoneImporter_StoresPartial(t *testing.T) {
	for _, partial := range []bool{false, true} {
		storage := newInMemoryZoneStorage()
		domain := &happydns.Domain{
			Id:         happydns.Identifier([]byte("test-domain")),
			DomainName: "example.com.",
		}
		uc := orchestrator.NewZoneImporterUsecase(
			&mockDomainUpdater{domain: domain},
			zoneUC.NewCreateZoneUsecase(storage),
			zoneUC.NewGetZoneUsecase(storage),
		)

		records := []happydns.Record{
			&dns.A{
				Hdr: dns.RR_Header{Name: "www.example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
				A:   []byte{192, 0, 2, 1},
			},
		}
		zone, err := uc.Import(&happydns.User{}, domain, records, partial)
		if err != nil {
			t.Fatalf("Import(partial=%v): %v", partial, err)
		}
		if zone.Partial != partial {
			t.Errorf("Import(partial=%v) returned a zone with Partial = %v", partial, zone.Partial)
		}

		if len(domain.ZoneHistory) != 1 {
			t.Fatalf("Import(partial=%v) left a history of %d zones; want 1", partial, len(domain.ZoneHistory))
		}
		stored, err := storage.GetZoneMeta(domain.ZoneHistory[0])
		if err != nil {
			t.Fatalf("GetZoneMeta: %v", err)
		}
		if stored.Partial != partial {
			t.Errorf("Import(partial=%v) stored a zone with Partial = %v", partial, stored.Partial)
		}
	}
}
