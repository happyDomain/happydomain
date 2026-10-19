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
	"context"
	"errors"
	"testing"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/providers"
)

// readOnlyFixture sets up a domain on a provider limited to zone transfers,
// with one correction pending, and reports whether it was executed.
func readOnlyFixture(t *testing.T) (apply func() error, prepare func() (*happydns.PrepareZoneResponse, error), executed *bool) {
	t.Helper()

	provider := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{Type: "DDNSServer"},
		Provider:     &providers.DDNSServer{TransferOnly: true},
	}

	storage := newInMemoryZoneStorage()
	wipZoneID := happydns.Identifier([]byte("wip-zone"))
	wipZone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{Id: wipZoneID, DefaultTTL: 3600},
		Services: map[happydns.Subdomain][]*happydns.Service{},
	}
	storage.zones[wipZoneID.String()] = wipZone

	domain := &happydns.Domain{
		Id:          happydns.Identifier([]byte("test-domain")),
		ProviderId:  happydns.Identifier([]byte("test-provider")),
		DomainName:  "example.com.",
		ZoneHistory: []happydns.Identifier{wipZoneID},
	}

	executed = new(bool)
	correction := &happydns.Correction{
		Msg: "a pending change",
		F: func() error {
			*executed = true
			return nil
		},
	}

	uc := buildTestApplier(
		&mockProviderGetter{provider: provider},
		&mockZoneCorrector{corrections: []*happydns.Correction{correction}, nbDiff: 1},
		&mockZoneRetriever{},
		&mockDomainUpdater{domain: domain},
		storage,
	)
	user := &happydns.User{Id: happydns.Identifier([]byte("test-user"))}

	apply = func() error {
		_, err := uc.Apply(context.Background(), user, domain, wipZone, &happydns.ApplyZoneForm{CommitMsg: "test deploy"})
		return err
	}
	prepare = func() (*happydns.PrepareZoneResponse, error) {
		return uc.Prepare(context.Background(), user, domain, wipZone, &happydns.PrepareZoneForm{})
	}

	return apply, prepare, executed
}

func TestApply_RefusedWhenProviderReadOnly(t *testing.T) {
	apply, _, executed := readOnlyFixture(t)

	err := apply()
	var forbidden happydns.ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Errorf("Apply() = %v, want a ForbiddenError", err)
	}
	if *executed {
		t.Error("Apply() executed a correction on a read-only provider")
	}
}

func TestPrepare_AllowedWhenProviderReadOnly(t *testing.T) {
	_, prepare, executed := readOnlyFixture(t)

	resp, err := prepare()
	if err != nil {
		t.Fatalf("Prepare() = %v, want the differences of a read-only provider", err)
	}
	if len(resp.Corrections) != 1 {
		t.Errorf("Prepare() returned %d corrections, want 1", len(resp.Corrections))
	}
	if *executed {
		t.Error("Prepare() executed a correction")
	}
}
