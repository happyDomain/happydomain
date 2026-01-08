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

package happydns_test

import (
	"testing"

	"git.happydns.org/happyDomain/model"
)

func TestNewDomain(t *testing.T) {
	user := &happydns.User{
		Id:    happydns.Identifier{0x01, 0x02, 0x03},
		Email: "test@example.com",
	}
	providerID := happydns.Identifier{0x04, 0x05, 0x06}

	tests := []struct {
		name        string
		domainName  string
		expectError bool
	}{
		{
			name:        "valid domain",
			domainName:  "example.com",
			expectError: false,
		},
		{
			name:        "valid domain with trailing dot",
			domainName:  "example.com.",
			expectError: false,
		},
		{
			name:        "valid subdomain",
			domainName:  "sub.example.com",
			expectError: false,
		},
		{
			name:        "valid domain with spaces trimmed",
			domainName:  "  example.com  ",
			expectError: false,
		},
		{
			name:        "empty domain name",
			domainName:  "",
			expectError: true,
		},
		{
			name:        "whitespace only",
			domainName:  "   ",
			expectError: true,
		},
		{
			name:        "domain with underscore",
			domainName:  "domain_with_underscore.com",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain, err := happydns.NewDomain(user, tt.domainName, providerID)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewDomain() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewDomain() error = %v", err)
			}

			if domain == nil {
				t.Fatal("NewDomain() returned nil domain")
			}

			if !domain.Owner.Equals(user.Id) {
				t.Errorf("NewDomain().Owner = %v; want %v", domain.Owner, user.Id)
			}

			if !domain.ProviderId.Equals(providerID) {
				t.Errorf("NewDomain().ProviderId = %v; want %v", domain.ProviderId, providerID)
			}

			if domain.DomainName == "" {
				t.Error("NewDomain().DomainName should not be empty")
			}

			if domain.DomainName[len(domain.DomainName)-1] != '.' {
				t.Errorf("NewDomain().DomainName should end with '.', got %q", domain.DomainName)
			}
		})
	}
}

func TestDomainHasZone(t *testing.T) {
	zoneId1 := happydns.Identifier{0x01, 0x02, 0x03}
	zoneId2 := happydns.Identifier{0x04, 0x05, 0x06}
	zoneId3 := happydns.Identifier{0x07, 0x08, 0x09}

	domain := &happydns.Domain{
		ZoneHistory: []happydns.Identifier{zoneId1, zoneId2},
	}

	tests := []struct {
		name     string
		zoneId   happydns.Identifier
		expected bool
	}{
		{
			name:     "zone exists in history (first)",
			zoneId:   zoneId1,
			expected: true,
		},
		{
			name:     "zone exists in history (second)",
			zoneId:   zoneId2,
			expected: true,
		},
		{
			name:     "zone does not exist in history",
			zoneId:   zoneId3,
			expected: false,
		},
		{
			name:     "empty zone id",
			zoneId:   happydns.Identifier{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.HasZone(tt.zoneId)
			if result != tt.expected {
				t.Errorf("HasZone() = %v; want %v", result, tt.expected)
			}
		})
	}
}

func TestDomainHasZoneEmptyHistory(t *testing.T) {
	domain := &happydns.Domain{
		ZoneHistory: []happydns.Identifier{},
	}

	zoneId := happydns.Identifier{0x01, 0x02, 0x03}

	if domain.HasZone(zoneId) {
		t.Error("HasZone() should return false for empty zone history")
	}
}

func TestDomainHasZoneNilHistory(t *testing.T) {
	domain := &happydns.Domain{
		ZoneHistory: nil,
	}

	zoneId := happydns.Identifier{0x01, 0x02, 0x03}

	if domain.HasZone(zoneId) {
		t.Error("HasZone() should return false for nil zone history")
	}
}

func TestNewDomainFQDN(t *testing.T) {
	user := &happydns.User{
		Id:    happydns.Identifier{0x01, 0x02, 0x03},
		Email: "test@example.com",
	}
	providerID := happydns.Identifier{0x04, 0x05, 0x06}

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "example.com",
			expected: "example.com.",
		},
		{
			input:    "example.com.",
			expected: "example.com.",
		},
		{
			input:    "sub.example.com",
			expected: "sub.example.com.",
		},
		{
			input:    "  example.com  ",
			expected: "example.com.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			domain, err := happydns.NewDomain(user, tt.input, providerID)
			if err != nil {
				t.Fatalf("NewDomain() error = %v", err)
			}

			if domain.DomainName != tt.expected {
				t.Errorf("NewDomain().DomainName = %q; want %q", domain.DomainName, tt.expected)
			}
		})
	}
}


func TestNewDomainInitialization(t *testing.T) {
	user := &happydns.User{
		Id:    happydns.Identifier{0x01, 0x02, 0x03},
		Email: "test@example.com",
	}
	providerID := happydns.Identifier{0x04, 0x05, 0x06}

	domain, err := happydns.NewDomain(user, "example.com", providerID)
	if err != nil {
		t.Fatalf("NewDomain() error = %v", err)
	}

	if !domain.Id.IsEmpty() {
		t.Error("NewDomain() should initialize with empty Id")
	}

	if domain.Group != "" {
		t.Errorf("NewDomain() should initialize with empty Group, got %q", domain.Group)
	}

	if domain.ZoneHistory != nil {
		t.Error("NewDomain() should initialize with nil ZoneHistory")
	}
}
