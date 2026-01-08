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

type mockProviderBody struct {
	TestField string `json:"test_field"`
}

func (m *mockProviderBody) InstantiateProvider() (happydns.ProviderActuator, error) {
	return nil, nil
}

func TestProviderMeta(t *testing.T) {
	providerId := happydns.Identifier{0x01, 0x02, 0x03}
	ownerId := happydns.Identifier{0x04, 0x05, 0x06}

	provider := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{
			Type:    "test.Provider",
			Id:      providerId,
			Owner:   ownerId,
			Comment: "test comment",
		},
		Provider: &mockProviderBody{TestField: "test"},
	}

	meta := provider.Meta()

	if meta == nil {
		t.Fatal("Meta() returned nil")
	}

	if meta.Type != "test.Provider" {
		t.Errorf("Meta().Type = %q; want %q", meta.Type, "test.Provider")
	}

	if !meta.Id.Equals(providerId) {
		t.Errorf("Meta().Id = %v; want %v", meta.Id, providerId)
	}

	if !meta.Owner.Equals(ownerId) {
		t.Errorf("Meta().Owner = %v; want %v", meta.Owner, ownerId)
	}

	if meta.Comment != "test comment" {
		t.Errorf("Meta().Comment = %q; want %q", meta.Comment, "test comment")
	}
}

func TestProviderToMessage(t *testing.T) {
	providerId := happydns.Identifier{0x01, 0x02, 0x03}
	ownerId := happydns.Identifier{0x04, 0x05, 0x06}

	provider := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{
			Type:    "test.Provider",
			Id:      providerId,
			Owner:   ownerId,
			Comment: "test comment",
		},
		Provider: &mockProviderBody{TestField: "test value"},
	}

	msg, err := provider.ToMessage()
	if err != nil {
		t.Fatalf("ToMessage() error = %v", err)
	}

	if msg.Type != "test.Provider" {
		t.Errorf("ToMessage().Type = %q; want %q", msg.Type, "test.Provider")
	}

	if !msg.Id.Equals(providerId) {
		t.Errorf("ToMessage().Id = %v; want %v", msg.Id, providerId)
	}

	if !msg.Owner.Equals(ownerId) {
		t.Errorf("ToMessage().Owner = %v; want %v", msg.Owner, ownerId)
	}

	if msg.Comment != "test comment" {
		t.Errorf("ToMessage().Comment = %q; want %q", msg.Comment, "test comment")
	}

	if len(msg.Provider) == 0 {
		t.Error("ToMessage().Provider should not be empty")
	}

	var providerBody mockProviderBody
	err = json.Unmarshal(msg.Provider, &providerBody)
	if err != nil {
		t.Fatalf("Unmarshal provider body error = %v", err)
	}

	if providerBody.TestField != "test value" {
		t.Errorf("ToMessage() provider body TestField = %q; want %q", providerBody.TestField, "test value")
	}
}

func TestProviderMessageMeta(t *testing.T) {
	providerId := happydns.Identifier{0x01, 0x02, 0x03}
	ownerId := happydns.Identifier{0x04, 0x05, 0x06}

	msg := &happydns.ProviderMessage{
		ProviderMeta: happydns.ProviderMeta{
			Type:    "test.Provider",
			Id:      providerId,
			Owner:   ownerId,
			Comment: "test comment",
		},
		Provider: json.RawMessage(`{"test_field":"test"}`),
	}

	meta := msg.Meta()

	if meta == nil {
		t.Fatal("Meta() returned nil")
	}

	if meta.Type != "test.Provider" {
		t.Errorf("Meta().Type = %q; want %q", meta.Type, "test.Provider")
	}

	if !meta.Id.Equals(providerId) {
		t.Errorf("Meta().Id = %v; want %v", meta.Id, providerId)
	}

	if !meta.Owner.Equals(ownerId) {
		t.Errorf("Meta().Owner = %v; want %v", meta.Owner, ownerId)
	}
}

func TestProviderMessagesMetas(t *testing.T) {
	providerId1 := happydns.Identifier{0x01, 0x02, 0x03}
	providerId2 := happydns.Identifier{0x04, 0x05, 0x06}
	ownerId := happydns.Identifier{0x07, 0x08, 0x09}

	messages := happydns.ProviderMessages{
		{
			ProviderMeta: happydns.ProviderMeta{
				Type:    "test.Provider1",
				Id:      providerId1,
				Owner:   ownerId,
				Comment: "provider 1",
			},
			Provider: json.RawMessage(`{}`),
		},
		{
			ProviderMeta: happydns.ProviderMeta{
				Type:    "test.Provider2",
				Id:      providerId2,
				Owner:   ownerId,
				Comment: "provider 2",
			},
			Provider: json.RawMessage(`{}`),
		},
	}

	metas := messages.Metas()

	if len(metas) != 2 {
		t.Fatalf("Metas() length = %d; want 2", len(metas))
	}

	if metas[0].Type != "test.Provider1" {
		t.Errorf("Metas()[0].Type = %q; want %q", metas[0].Type, "test.Provider1")
	}

	if !metas[0].Id.Equals(providerId1) {
		t.Errorf("Metas()[0].Id = %v; want %v", metas[0].Id, providerId1)
	}

	if metas[1].Type != "test.Provider2" {
		t.Errorf("Metas()[1].Type = %q; want %q", metas[1].Type, "test.Provider2")
	}

	if !metas[1].Id.Equals(providerId2) {
		t.Errorf("Metas()[1].Id = %v; want %v", metas[1].Id, providerId2)
	}
}

func TestProviderMessagesMetasEmpty(t *testing.T) {
	messages := happydns.ProviderMessages{}

	metas := messages.Metas()

	if len(metas) != 0 {
		t.Errorf("Metas() length = %d; want 0", len(metas))
	}
}

func TestProviderInstantiateProvider(t *testing.T) {
	provider := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{
			Type: "test.Provider",
			Id:   happydns.Identifier{0x01, 0x02},
		},
		Provider: &mockProviderBody{TestField: "test"},
	}

	actuator, err := provider.InstantiateProvider()
	if err != nil {
		t.Fatalf("InstantiateProvider() error = %v", err)
	}

	if actuator != nil {
		t.Error("mockProviderBody.InstantiateProvider() should return nil (mock implementation)")
	}
}

func TestProviderRoundTripToMessage(t *testing.T) {
	providerId := happydns.Identifier{0xaa, 0xbb, 0xcc}
	ownerId := happydns.Identifier{0xdd, 0xee, 0xff}

	originalProvider := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{
			Type:    "test.Provider",
			Id:      providerId,
			Owner:   ownerId,
			Comment: "original comment",
		},
		Provider: &mockProviderBody{TestField: "original value"},
	}

	msg, err := originalProvider.ToMessage()
	if err != nil {
		t.Fatalf("ToMessage() error = %v", err)
	}

	if !msg.Id.Equals(providerId) {
		t.Error("Round trip: provider ID mismatch")
	}

	if !msg.Owner.Equals(ownerId) {
		t.Error("Round trip: owner ID mismatch")
	}

	if msg.Type != "test.Provider" {
		t.Error("Round trip: type mismatch")
	}

	if msg.Comment != "original comment" {
		t.Error("Round trip: comment mismatch")
	}

	var body mockProviderBody
	err = json.Unmarshal(msg.Provider, &body)
	if err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if body.TestField != "original value" {
		t.Error("Round trip: provider body field mismatch")
	}
}
