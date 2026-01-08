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
	"encoding/json"
	"testing"

	"git.happydns.org/happyDomain/model"
)

func TestServiceMeta(t *testing.T) {
	serviceId := happydns.Identifier{0x01, 0x02, 0x03}
	ownerId := happydns.Identifier{0x04, 0x05, 0x06}

	service := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Type:        "test.Service",
			Id:          serviceId,
			OwnerId:     ownerId,
			Domain:      "example.com",
			Ttl:         3600,
			Comment:     "test comment",
			UserComment: "user comment",
			Aliases:     []string{"alias1", "alias2"},
			NbResources: 5,
		},
		Service: &mockServiceBody{nbResources: 5, comment: "test"},
	}

	meta := service.Meta()

	if meta == nil {
		t.Fatal("Meta() returned nil")
	}

	if meta.Type != "test.Service" {
		t.Errorf("Meta().Type = %q; want %q", meta.Type, "test.Service")
	}

	if !meta.Id.Equals(serviceId) {
		t.Errorf("Meta().Id = %v; want %v", meta.Id, serviceId)
	}

	if !meta.OwnerId.Equals(ownerId) {
		t.Errorf("Meta().OwnerId = %v; want %v", meta.OwnerId, ownerId)
	}

	if meta.Domain != "example.com" {
		t.Errorf("Meta().Domain = %q; want %q", meta.Domain, "example.com")
	}

	if meta.Ttl != 3600 {
		t.Errorf("Meta().Ttl = %d; want 3600", meta.Ttl)
	}

	if meta.Comment != "test comment" {
		t.Errorf("Meta().Comment = %q; want %q", meta.Comment, "test comment")
	}

	if meta.UserComment != "user comment" {
		t.Errorf("Meta().UserComment = %q; want %q", meta.UserComment, "user comment")
	}

	if len(meta.Aliases) != 2 {
		t.Errorf("Meta().Aliases length = %d; want 2", len(meta.Aliases))
	}

	if meta.NbResources != 5 {
		t.Errorf("Meta().NbResources = %d; want 5", meta.NbResources)
	}
}

func TestServiceMessageMeta(t *testing.T) {
	serviceId := happydns.Identifier{0x01, 0x02, 0x03}
	ownerId := happydns.Identifier{0x04, 0x05, 0x06}

	msg := &happydns.ServiceMessage{
		ServiceMeta: happydns.ServiceMeta{
			Type:    "test.Service",
			Id:      serviceId,
			OwnerId: ownerId,
			Domain:  "example.com",
			Ttl:     7200,
		},
		Service: json.RawMessage(`{"field":"value"}`),
	}

	meta := msg.Meta()

	if meta == nil {
		t.Fatal("Meta() returned nil")
	}

	if meta.Type != "test.Service" {
		t.Errorf("Meta().Type = %q; want %q", meta.Type, "test.Service")
	}

	if !meta.Id.Equals(serviceId) {
		t.Errorf("Meta().Id = %v; want %v", meta.Id, serviceId)
	}

	if !meta.OwnerId.Equals(ownerId) {
		t.Errorf("Meta().OwnerId = %v; want %v", meta.OwnerId, ownerId)
	}

	if meta.Domain != "example.com" {
		t.Errorf("Meta().Domain = %q; want %q", meta.Domain, "example.com")
	}

	if meta.Ttl != 7200 {
		t.Errorf("Meta().Ttl = %d; want 7200", meta.Ttl)
	}
}

func TestServiceMetaPointerEquality(t *testing.T) {
	service := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Type: "test.Service",
			Id:   happydns.Identifier{0x01},
		},
		Service: &mockServiceBody{},
	}

	meta1 := service.Meta()
	meta2 := service.Meta()

	if meta1.Type != meta2.Type {
		t.Error("Multiple calls to Meta() should return same metadata values")
	}

	if !meta1.Id.Equals(meta2.Id) {
		t.Error("Multiple calls to Meta() should return same ID")
	}
}

func TestServiceMetaWithEmptyFields(t *testing.T) {
	service := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Type:   "test.Service",
			Id:     happydns.Identifier{},
			Domain: "",
		},
		Service: &mockServiceBody{},
	}

	meta := service.Meta()

	if meta == nil {
		t.Fatal("Meta() returned nil")
	}

	if meta.Type != "test.Service" {
		t.Errorf("Meta().Type = %q; want %q", meta.Type, "test.Service")
	}

	if !meta.Id.IsEmpty() {
		t.Error("Meta().Id should be empty")
	}

	if meta.Domain != "" {
		t.Errorf("Meta().Domain = %q; want empty string", meta.Domain)
	}
}

func TestServiceMetaAliases(t *testing.T) {
	tests := []struct {
		name    string
		aliases []string
	}{
		{
			name:    "no aliases",
			aliases: nil,
		},
		{
			name:    "empty aliases",
			aliases: []string{},
		},
		{
			name:    "single alias",
			aliases: []string{"alias1"},
		},
		{
			name:    "multiple aliases",
			aliases: []string{"alias1", "alias2", "alias3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &happydns.Service{
				ServiceMeta: happydns.ServiceMeta{
					Type:    "test.Service",
					Aliases: tt.aliases,
				},
				Service: &mockServiceBody{},
			}

			meta := service.Meta()

			if len(meta.Aliases) != len(tt.aliases) {
				t.Errorf("Meta().Aliases length = %d; want %d", len(meta.Aliases), len(tt.aliases))
			}

			for i, alias := range tt.aliases {
				if meta.Aliases[i] != alias {
					t.Errorf("Meta().Aliases[%d] = %q; want %q", i, meta.Aliases[i], alias)
				}
			}
		})
	}
}

func TestServiceMetaDifferentTTLs(t *testing.T) {
	tests := []struct {
		name string
		ttl  uint32
	}{
		{name: "zero ttl", ttl: 0},
		{name: "small ttl", ttl: 60},
		{name: "medium ttl", ttl: 3600},
		{name: "large ttl", ttl: 86400},
		{name: "very large ttl", ttl: 2592000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &happydns.Service{
				ServiceMeta: happydns.ServiceMeta{
					Type: "test.Service",
					Ttl:  tt.ttl,
				},
				Service: &mockServiceBody{},
			}

			meta := service.Meta()

			if meta.Ttl != tt.ttl {
				t.Errorf("Meta().Ttl = %d; want %d", meta.Ttl, tt.ttl)
			}
		})
	}
}

func TestServiceMessageWithRawJSON(t *testing.T) {
	serviceId := happydns.Identifier{0xaa, 0xbb}

	testJSON := `{"test_field":"test_value","number":42}`

	msg := &happydns.ServiceMessage{
		ServiceMeta: happydns.ServiceMeta{
			Type: "test.Service",
			Id:   serviceId,
		},
		Service: json.RawMessage(testJSON),
	}

	meta := msg.Meta()

	if !meta.Id.Equals(serviceId) {
		t.Error("ServiceMessage Meta() should preserve ID")
	}

	if string(msg.Service) != testJSON {
		t.Error("ServiceMessage should preserve raw JSON")
	}
}

func TestServiceMetaComments(t *testing.T) {
	tests := []struct {
		name        string
		comment     string
		userComment string
	}{
		{
			name:        "both comments empty",
			comment:     "",
			userComment: "",
		},
		{
			name:        "only comment",
			comment:     "auto comment",
			userComment: "",
		},
		{
			name:        "only user comment",
			comment:     "",
			userComment: "user's note",
		},
		{
			name:        "both comments present",
			comment:     "auto comment",
			userComment: "user's note",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &happydns.Service{
				ServiceMeta: happydns.ServiceMeta{
					Type:        "test.Service",
					Comment:     tt.comment,
					UserComment: tt.userComment,
				},
				Service: &mockServiceBody{},
			}

			meta := service.Meta()

			if meta.Comment != tt.comment {
				t.Errorf("Meta().Comment = %q; want %q", meta.Comment, tt.comment)
			}

			if meta.UserComment != tt.userComment {
				t.Errorf("Meta().UserComment = %q; want %q", meta.UserComment, tt.userComment)
			}
		})
	}
}
