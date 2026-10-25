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

package happydns_test

import (
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

func TestZoneTemplateIsGlobal(t *testing.T) {
	empty := happydns.Identifier{}
	owner := happydns.Identifier{0x01}
	for _, tc := range []struct {
		name   string
		userId *happydns.Identifier
		want   bool
	}{
		{"without owner", nil, true},
		{"with an empty owner", &empty, true},
		{"with an owner", &owner, false},
	} {
		meta := happydns.ZoneTemplateMeta{UserId: tc.userId}
		if got := meta.IsGlobal(); got != tc.want {
			t.Errorf("IsGlobal %s = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// The use cases edit the services of a template through its zone.
func TestZoneTemplateZoneSharesTheServices(t *testing.T) {
	tpl := &happydns.ZoneTemplate{
		ZoneTemplateMeta: happydns.ZoneTemplateMeta{Id: happydns.Identifier{0x02}, DefaultTTL: 300},
		Services:         map[happydns.Subdomain][]*happydns.Service{},
	}

	zone := tpl.Zone()
	if !zone.Id.Equals(tpl.Id) || zone.DefaultTTL != 300 {
		t.Errorf("zone of the template = %+v, want its identifier and TTL", zone.ZoneMeta)
	}
	zone.Services["www"] = []*happydns.Service{{}}
	if len(tpl.Services["www"]) != 1 {
		t.Error("a service added to the zone is not in the template")
	}
}
