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
	"time"

	"git.happydns.org/happyDomain/model"
)

type mockServiceBody struct {
	nbResources int
	comment     string
}

func (m *mockServiceBody) GetNbResources() int {
	return m.nbResources
}

func (m *mockServiceBody) GenComment() string {
	return m.comment
}

func (m *mockServiceBody) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	return nil, nil
}

func createTestZone() *happydns.Zone {
	authorID := happydns.Identifier{0x01, 0x02, 0x03}

	service1 := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Type:   "test.Service1",
			Id:     happydns.Identifier{0x10, 0x11, 0x12},
			Domain: "example.com",
		},
		Service: &mockServiceBody{nbResources: 1, comment: "service1"},
	}

	service2 := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Type:   "test.Service2",
			Id:     happydns.Identifier{0x20, 0x21, 0x22},
			Domain: "sub.example.com",
		},
		Service: &mockServiceBody{nbResources: 2, comment: "service2"},
	}

	service3 := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Type:   "test.Service3",
			Id:     happydns.Identifier{0x30, 0x31, 0x32},
			Domain: "example.com",
		},
		Service: &mockServiceBody{nbResources: 1, comment: "service3"},
	}

	return &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{
			Id:         happydns.Identifier{0xaa, 0xbb, 0xcc},
			IdAuthor:   authorID,
			DefaultTTL: 3600,
		},
		Services: map[happydns.Subdomain][]*happydns.Service{
			"":    {service1, service3},
			"sub": {service2},
		},
	}
}

func TestZoneDerivateNew(t *testing.T) {
	originalZone := createTestZone()
	originalTime := originalZone.LastModified

	time.Sleep(10 * time.Millisecond)

	newZone := originalZone.DerivateNew()

	if newZone == nil {
		t.Fatal("DerivateNew() returned nil")
	}

	if !newZone.ParentZone.Equals(originalZone.Id) {
		t.Errorf("DerivateNew().ParentZone = %v; want %v", newZone.ParentZone, originalZone.Id)
	}

	if !newZone.IdAuthor.Equals(originalZone.IdAuthor) {
		t.Errorf("DerivateNew().IdAuthor = %v; want %v", newZone.IdAuthor, originalZone.IdAuthor)
	}

	if newZone.DefaultTTL != originalZone.DefaultTTL {
		t.Errorf("DerivateNew().DefaultTTL = %d; want %d", newZone.DefaultTTL, originalZone.DefaultTTL)
	}

	if newZone.LastModified.Before(originalTime) || newZone.LastModified.Equal(originalTime) {
		t.Errorf("DerivateNew().LastModified should be after original, got %v (original: %v)", newZone.LastModified, originalTime)
	}

	if newZone.Services == nil {
		t.Fatal("DerivateNew().Services should not be nil")
	}

	if len(newZone.Services) != len(originalZone.Services) {
		t.Errorf("DerivateNew() services count = %d; want %d", len(newZone.Services), len(originalZone.Services))
	}

	for subdomain, services := range originalZone.Services {
		newServices, ok := newZone.Services[subdomain]
		if !ok {
			t.Errorf("DerivateNew() missing subdomain %q", subdomain)
			continue
		}

		if len(newServices) != len(services) {
			t.Errorf("DerivateNew() subdomain %q has %d services; want %d", subdomain, len(newServices), len(services))
		}

		for i, svc := range services {
			if !newServices[i].Id.Equals(svc.Id) {
				t.Errorf("DerivateNew() subdomain %q service %d id mismatch", subdomain, i)
			}
		}
	}

	if !newZone.Id.IsEmpty() {
		t.Error("DerivateNew().Id should be empty")
	}
}

func TestZoneFindService(t *testing.T) {
	zone := createTestZone()

	tests := []struct {
		name              string
		serviceId         happydns.Identifier
		expectedSubdomain happydns.Subdomain
		expectFound       bool
	}{
		{
			name:              "find service in root",
			serviceId:         happydns.Identifier{0x10, 0x11, 0x12},
			expectedSubdomain: "",
			expectFound:       true,
		},
		{
			name:              "find service in subdomain",
			serviceId:         happydns.Identifier{0x20, 0x21, 0x22},
			expectedSubdomain: "sub",
			expectFound:       true,
		},
		{
			name:              "find second service in root",
			serviceId:         happydns.Identifier{0x30, 0x31, 0x32},
			expectedSubdomain: "",
			expectFound:       true,
		},
		{
			name:        "service not found",
			serviceId:   happydns.Identifier{0xff, 0xff, 0xff},
			expectFound: false,
		},
		{
			name:        "empty service id",
			serviceId:   happydns.Identifier{},
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subdomain, service := zone.FindService(tt.serviceId)

			if tt.expectFound {
				if service == nil {
					t.Fatalf("FindService() expected to find service but got nil")
				}
				if subdomain != tt.expectedSubdomain {
					t.Errorf("FindService() subdomain = %q; want %q", subdomain, tt.expectedSubdomain)
				}
				if !service.Id.Equals(tt.serviceId) {
					t.Errorf("FindService() service.Id = %v; want %v", service.Id, tt.serviceId)
				}
			} else {
				if service != nil {
					t.Errorf("FindService() expected nil but found service: %v", service)
				}
			}
		})
	}
}

func TestZoneFindSubdomainService(t *testing.T) {
	zone := createTestZone()

	tests := []struct {
		name          string
		subdomain     happydns.Subdomain
		serviceId     happydns.Identifier
		expectedIndex int
		expectFound   bool
	}{
		{
			name:          "find first service in root",
			subdomain:     "",
			serviceId:     happydns.Identifier{0x10, 0x11, 0x12},
			expectedIndex: 0,
			expectFound:   true,
		},
		{
			name:          "find second service in root",
			subdomain:     "",
			serviceId:     happydns.Identifier{0x30, 0x31, 0x32},
			expectedIndex: 1,
			expectFound:   true,
		},
		{
			name:          "find service in subdomain",
			subdomain:     "sub",
			serviceId:     happydns.Identifier{0x20, 0x21, 0x22},
			expectedIndex: 0,
			expectFound:   true,
		},
		{
			name:          "@ alias for root subdomain",
			subdomain:     "@",
			serviceId:     happydns.Identifier{0x10, 0x11, 0x12},
			expectedIndex: 0,
			expectFound:   true,
		},
		{
			name:        "service not in subdomain",
			subdomain:   "sub",
			serviceId:   happydns.Identifier{0x10, 0x11, 0x12},
			expectFound: false,
		},
		{
			name:        "nonexistent subdomain",
			subdomain:   "nonexistent",
			serviceId:   happydns.Identifier{0x10, 0x11, 0x12},
			expectFound: false,
		},
		{
			name:        "nonexistent service",
			subdomain:   "",
			serviceId:   happydns.Identifier{0xff, 0xff, 0xff},
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index, service := zone.FindSubdomainService(tt.subdomain, tt.serviceId)

			if tt.expectFound {
				if service == nil {
					t.Fatalf("FindSubdomainService() expected to find service but got nil")
				}
				if index != tt.expectedIndex {
					t.Errorf("FindSubdomainService() index = %d; want %d", index, tt.expectedIndex)
				}
				if !service.Id.Equals(tt.serviceId) {
					t.Errorf("FindSubdomainService() service.Id = %v; want %v", service.Id, tt.serviceId)
				}
			} else {
				if service != nil {
					t.Errorf("FindSubdomainService() expected nil but found service: %v", service)
				}
				if index != -1 {
					t.Errorf("FindSubdomainService() index = %d; want -1", index)
				}
			}
		})
	}
}

func TestZoneEraseService(t *testing.T) {
	zone := createTestZone()
	serviceId := happydns.Identifier{0x20, 0x21, 0x22}

	subdomain := happydns.Subdomain("sub")
	originalCount := len(zone.Services[subdomain])

	err := zone.EraseService(subdomain, serviceId, nil)
	if err != nil {
		t.Fatalf("EraseService() error = %v", err)
	}

	if _, ok := zone.Services[subdomain]; ok {
		t.Error("EraseService() should have removed the subdomain when last service was deleted")
	}

	if originalCount != 1 {
		t.Errorf("Test assumption failed: expected 1 service in subdomain, got %d", originalCount)
	}
}

func TestZoneEraseServiceNotFound(t *testing.T) {
	zone := createTestZone()
	serviceId := happydns.Identifier{0xff, 0xff, 0xff}

	err := zone.EraseService("", serviceId, nil)
	if err == nil {
		t.Error("EraseService() expected error for non-existent service")
	}
}

func TestZoneEraseServiceWithReplacement(t *testing.T) {
	zone := createTestZone()
	serviceId := happydns.Identifier{0x10, 0x11, 0x12}

	newService := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Type:   "test.NewService",
			Id:     serviceId,
			Domain: "example.com",
		},
		Service: &mockServiceBody{nbResources: 5, comment: "new service"},
	}

	err := zone.EraseService("", serviceId, newService)
	if err != nil {
		t.Fatalf("EraseService() error = %v", err)
	}

	_, service := zone.FindSubdomainService("", serviceId)
	if service == nil {
		t.Fatal("EraseService() should have replaced the service, not deleted it")
	}

	if service.Type != "test.NewService" {
		t.Errorf("EraseService() service.Type = %q; want %q", service.Type, "test.NewService")
	}

	if service.NbResources != 5 {
		t.Errorf("EraseService() service.NbResources = %d; want 5", service.NbResources)
	}

	if service.Comment != "new service" {
		t.Errorf("EraseService() service.Comment = %q; want %q", service.Comment, "new service")
	}
}

func TestZoneEraseServiceMultipleServices(t *testing.T) {
	zone := createTestZone()
	serviceId := happydns.Identifier{0x10, 0x11, 0x12}

	originalCount := len(zone.Services[""])

	err := zone.EraseService("", serviceId, nil)
	if err != nil {
		t.Fatalf("EraseService() error = %v", err)
	}

	if len(zone.Services[""]) != originalCount-1 {
		t.Errorf("EraseService() service count = %d; want %d", len(zone.Services[""]), originalCount-1)
	}

	_, service := zone.FindSubdomainService("", serviceId)
	if service != nil {
		t.Error("EraseService() service should have been deleted")
	}
}

func TestZoneEraseServiceWithoutMeta(t *testing.T) {
	zone := createTestZone()
	serviceId := happydns.Identifier{0x10, 0x11, 0x12}

	newServiceBody := &mockServiceBody{nbResources: 7, comment: "updated service"}

	err := zone.EraseServiceWithoutMeta("", serviceId, newServiceBody)
	if err != nil {
		t.Fatalf("EraseServiceWithoutMeta() error = %v", err)
	}

	_, service := zone.FindSubdomainService("", serviceId)
	if service == nil {
		t.Fatal("EraseServiceWithoutMeta() service should exist")
	}

	if !service.Id.Equals(serviceId) {
		t.Error("EraseServiceWithoutMeta() should preserve service ID")
	}

	if service.NbResources != 7 {
		t.Errorf("EraseServiceWithoutMeta() service.NbResources = %d; want 7", service.NbResources)
	}

	if service.Comment != "updated service" {
		t.Errorf("EraseServiceWithoutMeta() service.Comment = %q; want %q", service.Comment, "updated service")
	}
}

func TestZoneEraseServiceOriginProtection(t *testing.T) {
	zone := createTestZone()

	originService := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Type:   "abstract.Origin",
			Id:     happydns.Identifier{0x99, 0x99, 0x99},
			Domain: "",
		},
		Service: &mockServiceBody{nbResources: 1, comment: "origin"},
	}

	zone.Services[""] = []*happydns.Service{originService}

	err := zone.EraseService("", originService.Id, nil)
	if err == nil {
		t.Error("EraseService() should return error when trying to delete Origin service")
	}

	_, service := zone.FindSubdomainService("", originService.Id)
	if service == nil {
		t.Error("Origin service should not have been deleted")
	}
}
