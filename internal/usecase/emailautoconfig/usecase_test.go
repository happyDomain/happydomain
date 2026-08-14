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

package emailautoconfig

import (
	"errors"
	"testing"

	happydns "git.happydns.org/happyDomain/model"

	"git.happydns.org/happyDomain/internal/usecase/hostedservice/hostedservicetest"
)

// newUsecase wires an Usecase over a single domain owning a single zone that
// holds the given service.
func newUsecase(t *testing.T, fqdn string, service happydns.ServiceBody) *Usecase {
	t.Helper()

	domains, zones := hostedservicetest.NewStore(t, fqdn, service)

	return NewUsecase(domains, zones)
}

func TestMozillaConfig(t *testing.T) {
	uc := newUsecase(t, "example.com.", sampleService())

	for _, fqdn := range []string{"autoconfig.example.com.", "example.com."} {
		body, err := uc.MozillaConfig(fqdn, "user@example.com")
		if err != nil {
			t.Fatalf("MozillaConfig(%q): %v", fqdn, err)
		}
		if len(body) == 0 {
			t.Errorf("MozillaConfig(%q) returned empty body", fqdn)
		}
	}
}

func TestMozillaConfig_UnknownDomain(t *testing.T) {
	uc := newUsecase(t, "example.com.", sampleService())

	if _, err := uc.MozillaConfig("autoconfig.unknown.example.", "user@unknown.example"); !errors.Is(err, happydns.ErrNotFound) {
		t.Errorf("MozillaConfig(unknown) error = %v; want ErrNotFound", err)
	}
}

func TestAutodiscoverConfig(t *testing.T) {
	uc := newUsecase(t, "example.com.", sampleService())

	body, err := uc.AutodiscoverConfig("autodiscover.example.com.", "user@example.com")
	if err != nil {
		t.Fatalf("AutodiscoverConfig: %v", err)
	}
	if len(body) == 0 {
		t.Errorf("AutodiscoverConfig returned empty body")
	}
}

func TestIsManaged(t *testing.T) {
	uc := newUsecase(t, "example.com.", sampleService())

	for fqdn, want := range map[string]bool{
		"autoconfig.example.com.":     true,
		"autodiscover.example.com.":   true,
		"autoconfig.unknown.example.": false,
		// Never authorise a certificate for a name we do not answer on, even
		// when the domain itself is hosted here.
		"example.com.":         false,
		"mta-sts.example.com.": false,
	} {
		got, err := uc.IsManaged(fqdn)
		if err != nil {
			t.Fatalf("IsManaged(%q): %v", fqdn, err)
		}
		if got != want {
			t.Errorf("IsManaged(%q) = %v; want %v", fqdn, got, want)
		}
	}
}
