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
	"fmt"
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

func TestParseTwoIdKey(t *testing.T) {
	const prefix = "some.index|"

	first, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatalf("NewRandomIdentifier: %v", err)
	}
	second, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatalf("NewRandomIdentifier: %v", err)
	}

	key := fmt.Sprintf("%s%s|%s", prefix, first.String(), second.String())

	gotFirst, gotSecond, err := parseTwoIdKey(key, prefix)
	if err != nil {
		t.Fatalf("parseTwoIdKey(%q, %q): unexpected error: %v", key, prefix, err)
	}
	if !gotFirst.Equals(first) {
		t.Errorf("first = %s, want %s", gotFirst.String(), first.String())
	}
	if !gotSecond.Equals(second) {
		t.Errorf("second = %s, want %s", gotSecond.String(), second.String())
	}
}

func TestParseTwoIdKeyErrors(t *testing.T) {
	const prefix = "some.index|"

	validId, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatalf("NewRandomIdentifier: %v", err)
	}

	tests := []struct {
		name string
		key  string
	}{
		{"no separator", prefix + "onlyonesegment"},
		{"empty rest", prefix},
		{"invalid first segment", prefix + "not base64!|" + validId.String()},
		{"invalid second segment", prefix + validId.String() + "|not base64!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := parseTwoIdKey(tt.key, prefix); err == nil {
				t.Errorf("parseTwoIdKey(%q, %q): expected error, got nil", tt.key, prefix)
			}
		})
	}
}

// TestParseTwoIdKeyRoundTripsIndexKeyBuilders checks parseTwoIdKey against the
// actual two-identifier index key builders that rely on it.
func TestParseTwoIdKeyRoundTripsIndexKeyBuilders(t *testing.T) {
	domainId, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatalf("NewRandomIdentifier: %v", err)
	}
	granteeId, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatalf("NewRandomIdentifier: %v", err)
	}

	shareKey := domainShareIndexKey(domainId, granteeId)
	gotDomainId, gotGranteeId, err := parseTwoIdKey(shareKey, domainShareIndexPrefix)
	if err != nil {
		t.Fatalf("parseTwoIdKey(domainShareIndexKey(...)): unexpected error: %v", err)
	}
	if !gotDomainId.Equals(domainId) || !gotGranteeId.Equals(granteeId) {
		t.Errorf("domain share key round trip = (%s, %s), want (%s, %s)", gotDomainId.String(), gotGranteeId.String(), domainId.String(), granteeId.String())
	}

	grantKey := domainGrantIndexKey(granteeId, domainId)
	gotGranteeId, gotDomainId, err = parseTwoIdKey(grantKey, domainGrantIndexPrefix)
	if err != nil {
		t.Fatalf("parseTwoIdKey(domainGrantIndexKey(...)): unexpected error: %v", err)
	}
	if !gotGranteeId.Equals(granteeId) || !gotDomainId.Equals(domainId) {
		t.Errorf("domain grant key round trip = (%s, %s), want (%s, %s)", gotGranteeId.String(), gotDomainId.String(), granteeId.String(), domainId.String())
	}

	providerId, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatalf("NewRandomIdentifier: %v", err)
	}
	providerShareKey := providerShareIndexKey(providerId, granteeId)
	gotProviderId, gotGranteeId, err := parseTwoIdKey(providerShareKey, providerShareIndexPrefix)
	if err != nil {
		t.Fatalf("parseTwoIdKey(providerShareIndexKey(...)): unexpected error: %v", err)
	}
	if !gotProviderId.Equals(providerId) || !gotGranteeId.Equals(granteeId) {
		t.Errorf("provider share key round trip = (%s, %s), want (%s, %s)", gotProviderId.String(), gotGranteeId.String(), providerId.String(), granteeId.String())
	}
}
