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

package domaininfo

import (
	"errors"
	"testing"
	"time"

	"git.happydns.org/happyDomain/pkg/domaininfo/types"
)

func TestPhaseOf(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	future := now.AddDate(0, 3, 0)
	past := now.AddDate(0, 0, -3)

	cases := []struct {
		name   string
		info   *DomainInfo
		err    error
		want   Phase
		wantOk bool
	}{
		{"registered", &DomainInfo{ExpirationDate: &future, Status: []string{"clientTransferProhibited"}}, nil, PhaseRegistered, true},
		{"registered without expiration date", &DomainInfo{Status: []string{"active"}}, nil, PhaseRegistered, true},
		{"just renewed", &DomainInfo{ExpirationDate: &future, Status: []string{"renewPeriod"}}, nil, PhaseRegistered, true},
		{"expiring right now", &DomainInfo{ExpirationDate: &now}, nil, PhaseRegistered, true},
		{"expired", &DomainInfo{ExpirationDate: &past}, nil, PhaseGrace, true},
		{"auto-renewed by the registry", &DomainInfo{ExpirationDate: &future, Status: []string{"autoRenewPeriod", "clientHold"}}, nil, PhaseRegistered, true},
		{"redemption, EPP spelling", &DomainInfo{ExpirationDate: &past, Status: []string{"redemptionPeriod"}}, nil, PhaseRedemption, true},
		{"redemption, RDAP spelling", &DomainInfo{Status: []string{"redemption period"}}, nil, PhaseRedemption, true},
		{"pending restore", &DomainInfo{Status: []string{"pendingRestore"}}, nil, PhaseRedemption, true},
		{"pending restore, RDAP spelling", &DomainInfo{Status: []string{"pending restore"}}, nil, PhaseRedemption, true},
		{"pending delete wins", &DomainInfo{Status: []string{"redemptionPeriod", "pending delete"}}, nil, PhasePendingDelete, true},
		{"pending delete before expiration", &DomainInfo{ExpirationDate: &future, Status: []string{"pendingDelete"}}, nil, PhasePendingDelete, true},
		{"available", nil, types.ErrDomainDoesNotExist, PhaseAvailable, true},
		{"available, wrapped", nil, errors.Join(errors.New("lookup"), types.ErrDomainDoesNotExist), PhaseAvailable, true},
		{"lookup failure", nil, errors.New("timeout"), "", false},
		{"no answer", nil, nil, "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := PhaseOf(tc.info, tc.err, now)
			if got != tc.want || ok != tc.wantOk {
				t.Errorf("PhaseOf() = %q, %v, want %q, %v", got, ok, tc.want, tc.wantOk)
			}
		})
	}
}
