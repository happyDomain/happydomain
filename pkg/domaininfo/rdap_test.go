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

package domaininfo_test

import (
	"errors"
	"testing"

	happydns "git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/pkg/domaininfo"
)

func TestGetDomainRDAPInfo_KnownDomain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live RDAP integration test")
	}

	info, err := domaininfo.GetDomainRDAPInfo(happydns.Origin("example.com"))
	if err != nil {
		t.Fatalf("unexpected error for example.com: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil DomainInfo")
	}
	if info.Name == "" {
		t.Error("expected Name to be set")
	}
	if len(info.Nameservers) == 0 {
		t.Error("expected at least one nameserver")
	}
	if info.ExpirationDate == nil {
		t.Error("expected ExpirationDate to be set")
	}
	if info.CreationDate == nil {
		t.Error("expected CreationDate to be set")
	}
	if info.Registrar == "" {
		t.Error("expected Registrar to be set")
	}
}

func TestGetDomainRDAPInfo_NonExistentDomain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live RDAP integration test")
	}

	_, err := domaininfo.GetDomainRDAPInfo(happydns.Origin("this-domain-definitely-does-not-exist-xyz987654321.com"))
	if !errors.Is(err, happydns.DomainDoesNotExist) {
		t.Errorf("expected DomainDoesNotExist error, got: %v", err)
	}
}

