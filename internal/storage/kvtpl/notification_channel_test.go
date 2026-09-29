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

package database_test

import (
	"encoding/json"
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/storage"
	happydns "git.happydns.org/happyDomain/model"
)

func TestCreateChannelKeepsPresetId(t *testing.T) {
	s := newStorage(t)
	owner, id := newIdentifier(t), newIdentifier(t)

	ch := &happydns.NotificationChannel{Id: id, UserId: owner, Type: "webhook", Config: json.RawMessage(`{}`)}
	if err := s.CreateChannel(ch); err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	if !ch.Id.Equals(id) {
		t.Errorf("CreateChannel replaced the preset id")
	}
	if _, err := s.GetChannel(id); err != nil {
		t.Errorf("GetChannel(preset id): %v", err)
	}

	dup := &happydns.NotificationChannel{Id: id, UserId: owner, Type: "webhook"}
	if err := s.CreateChannel(dup); !errors.Is(err, happydns.ErrAlreadyExists) {
		t.Errorf("CreateChannel with an id already used = %v, want ErrAlreadyExists", err)
	}
}

// Like providers, the storage never chooses the identifier of a channel, and
// refuses one that is missing or malformed.
func TestCreateChannelRefusesInvalidId(t *testing.T) {
	for name, id := range map[string]happydns.Identifier{
		"missing":   nil,
		"too short": {0x01},
		"too long":  make(happydns.Identifier, happydns.IDENTIFIER_LEN+1),
	} {
		t.Run(name, func(t *testing.T) {
			s := newStorage(t)
			owner := newIdentifier(t)

			ch := &happydns.NotificationChannel{Id: id, UserId: owner, Type: "webhook", Config: json.RawMessage(`{}`)}
			if err := s.CreateChannel(ch); !errors.Is(err, happydns.ErrInvalidIdentifier) {
				t.Fatalf("CreateChannel = %v, want ErrInvalidIdentifier", err)
			}
			if channels, _ := s.ListChannelsByUser(owner); len(channels) != 0 {
				t.Error("a refused creation stored something")
			}
		})
	}
}

func TestListAllChannels(t *testing.T) {
	s := newStorage(t)
	for _, owner := range []happydns.Identifier{{0x01}, {0x02}} {
		if err := s.CreateChannel(&happydns.NotificationChannel{Id: newIdentifier(t), UserId: owner, Type: "webhook"}); err != nil {
			t.Fatal(err)
		}
	}

	iter, err := s.ListAllChannels()
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	n := 0
	for iter.Next() {
		n++
	}
	if n != 2 {
		t.Errorf("ListAllChannels = %d channels, want 2 (index entries must not be listed)", n)
	}
}

// Like providers, the write itself refuses an identifier taken by a
// concurrent creation that landed after the check.
func TestCreateChannelRefusesTakenIdAtomically(t *testing.T) {
	s, _ := storageOver(t, func(raw storage.KVStorage) storage.KVStorage { return racingKV{raw} })

	owner, other, id := newIdentifier(t), newIdentifier(t), newIdentifier(t)
	if err := s.CreateChannel(&happydns.NotificationChannel{Id: id, UserId: owner, Type: "webhook"}); err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	dup := &happydns.NotificationChannel{Id: append(happydns.Identifier(nil), id...), UserId: other, Type: "webhook"}
	if err := s.CreateChannel(dup); !errors.Is(err, happydns.ErrAlreadyExists) {
		t.Fatalf("CreateChannel racing on a taken id = %v, want ErrAlreadyExists", err)
	}

	got, err := s.GetChannel(id)
	if err != nil {
		t.Fatalf("GetChannel: %v", err)
	}
	if !got.UserId.Equals(owner) {
		t.Error("the first channel was overwritten")
	}
	if channels, _ := s.ListChannelsByUser(other); len(channels) != 0 {
		t.Error("a failed creation left a user index entry")
	}
}

// A channel whose user index could not be written would be stored but listed
// nowhere: the creation fails and leaves nothing behind.
func TestCreateChannelLeavesNothingWhenIndexFails(t *testing.T) {
	s, raw := storageOver(t, func(raw storage.KVStorage) storage.KVStorage {
		return failingIndexKV{raw, "notifch-user|"}
	})

	ch := &happydns.NotificationChannel{Id: newIdentifier(t), UserId: newIdentifier(t), Type: "webhook"}
	if err := s.CreateChannel(ch); !errors.Is(err, errIndexWrite) {
		t.Fatalf("CreateChannel = %v, want the index write error", err)
	}

	iter := raw.Search("notifch")
	defer iter.Release()
	for iter.Next() {
		t.Errorf("key %q left behind", iter.Key())
	}
}
