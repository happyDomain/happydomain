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
	"errors"
	"testing"
	"time"

	happydns "git.happydns.org/happyDomain/model"
)

func newSafe(owner happydns.Identifier, kind string) *happydns.Safe {
	id, _ := happydns.NewRandomIdentifier()
	return &happydns.Safe{
		Id:        id,
		Owner:     owner,
		Kind:      kind,
		Keyring:   []happydns.WrappedKeyset{{KEK: "instance", Blob: []byte{1, 2, 3}}},
		CreatedAt: time.Unix(1700000000, 0).UTC(),
	}
}

func TestSafeCRUD(t *testing.T) {
	s := newStorage(t)
	owner, _ := happydns.NewRandomIdentifier()
	safe := newSafe(owner, "instance")

	if err := s.CreateSafe(safe); err != nil {
		t.Fatalf("CreateSafe: %v", err)
	}
	if err := s.CreateSafe(safe); err == nil {
		t.Error("CreateSafe twice with the same id succeeded")
	}

	got, err := s.GetSafe(safe.Id)
	if err != nil {
		t.Fatalf("GetSafe: %v", err)
	}
	if !got.Owner.Equals(owner) || got.Kind != "instance" || len(got.Keyring) != 1 || got.Keyring[0].KEK != "instance" || !got.CreatedAt.Equal(safe.CreatedAt) {
		t.Errorf("GetSafe = %+v, want %+v", got, safe)
	}

	safe.Keyring = append(safe.Keyring, happydns.WrappedKeyset{KEK: "other", Blob: []byte{4}})
	if err := s.UpdateSafe(safe); err != nil {
		t.Fatalf("UpdateSafe: %v", err)
	}
	if got, _ := s.GetSafe(safe.Id); len(got.Keyring) != 2 {
		t.Errorf("UpdateSafe not applied: %+v", got)
	}

	if err := s.DeleteSafe(safe.Id); err != nil {
		t.Fatalf("DeleteSafe: %v", err)
	}
	if _, err := s.GetSafe(safe.Id); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("GetSafe after delete = %v, want ErrSafeNotFound", err)
	}
	if _, err := s.GetSafeByOwner(owner, "instance"); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("GetSafeByOwner after delete = %v, want ErrSafeNotFound", err)
	}
}

func TestSafeOwnerIndex(t *testing.T) {
	s := newStorage(t)
	alice, _ := happydns.NewRandomIdentifier()
	bob, _ := happydns.NewRandomIdentifier()

	aliceSafe := newSafe(alice, "instance")
	bobSafe := newSafe(bob, "instance")
	for _, safe := range []*happydns.Safe{aliceSafe, bobSafe} {
		if err := s.CreateSafe(safe); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.GetSafeByOwner(alice, "instance")
	if err != nil || !got.Id.Equals(aliceSafe.Id) {
		t.Errorf("GetSafeByOwner(alice) = %v, %v; want alice's safe", got, err)
	}
	if _, err := s.GetSafeByOwner(alice, "other"); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("GetSafeByOwner(alice, other) = %v, want ErrSafeNotFound", err)
	}

	iter, err := s.ListAllSafes()
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	n := 0
	for iter.Next() {
		n++
	}
	if n != 2 {
		t.Errorf("ListAllSafes = %d safes, want 2 (index entries must not be listed)", n)
	}
}

func TestSafeSeveralOfAKind(t *testing.T) {
	s := newStorage(t)
	alice, _ := happydns.NewRandomIdentifier()
	bob, _ := happydns.NewRandomIdentifier()

	first := newSafe(alice, "personal")
	first.Name = "first"
	second := newSafe(alice, "personal")
	second.Name = "second"
	second.CreatedAt = first.CreatedAt.Add(time.Hour)
	other := newSafe(alice, "instance")
	bobs := newSafe(bob, "personal")
	// Created out of order: the oldest is found whatever the order.
	for _, safe := range []*happydns.Safe{second, other, first, bobs} {
		if err := s.CreateSafe(safe); err != nil {
			t.Fatalf("CreateSafe(%s): %v", safe.Name, err)
		}
	}

	got, err := s.ListOwnerSafes(alice)
	if err != nil {
		t.Fatalf("ListOwnerSafes: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("ListOwnerSafes(alice) = %d safes, want 3", len(got))
	}
	for _, safe := range got {
		if !safe.Owner.Equals(alice) {
			t.Errorf("ListOwnerSafes(alice) lists a safe of %s", safe.Owner.String())
		}
	}

	if safe, err := s.GetSafe(second.Id); err != nil || safe.Name != "second" {
		t.Errorf("GetSafe(second) = %+v, %v; want its name kept", safe, err)
	}

	// The safes an owner created are no safe of the instance, older or not.
	if safe, err := s.GetSafeByOwner(alice, "personal"); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("GetSafeByOwner(alice, personal) = %+v, %v; want ErrSafeNotFound", safe, err)
	}
	unnamed := newSafe(alice, "personal")
	unnamed.CreatedAt = second.CreatedAt.Add(time.Hour)
	if err := s.CreateSafe(unnamed); err != nil {
		t.Fatal(err)
	}
	if safe, err := s.GetSafeByOwner(alice, "personal"); err != nil || !safe.Id.Equals(unnamed.Id) {
		t.Errorf("GetSafeByOwner(alice, personal) = %+v, %v; want the one without a name", safe, err)
	}

	if err := s.DeleteSafe(first.Id); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.ListOwnerSafes(alice); len(got) != 3 {
		t.Errorf("ListOwnerSafes(alice) after a delete = %d safes, want 3", len(got))
	}

	if got, err := s.ListOwnerSafes(happydns.Identifier{0x42}); err != nil || len(got) != 0 {
		t.Errorf("ListOwnerSafes(nobody) = %v, %v; want none", got, err)
	}
}

func TestDefaultKeystore(t *testing.T) {
	s := newStorage(t)
	alice, _ := happydns.NewRandomIdentifier()
	bob, _ := happydns.NewRandomIdentifier()

	if _, err := s.GetDefaultKeystore(alice); !errors.Is(err, happydns.ErrNotFound) {
		t.Errorf("GetDefaultKeystore before any = %v, want ErrNotFound", err)
	}

	if err := s.PutDefaultKeystore(alice, "instance"); err != nil {
		t.Fatal(err)
	}
	if err := s.PutDefaultKeystore(alice, "abc"); err != nil {
		t.Fatal(err)
	}
	if err := s.PutDefaultKeystore(bob, "instance"); err != nil {
		t.Fatal(err)
	}
	if ref, err := s.GetDefaultKeystore(alice); err != nil || ref != "abc" {
		t.Errorf("GetDefaultKeystore(alice) = %q, %v; want the last one set", ref, err)
	}

	if err := s.DeleteDefaultKeystore(alice); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetDefaultKeystore(alice); !errors.Is(err, happydns.ErrNotFound) {
		t.Errorf("GetDefaultKeystore after delete = %v, want ErrNotFound", err)
	}
	if ref, err := s.GetDefaultKeystore(bob); err != nil || ref != "instance" {
		t.Errorf("GetDefaultKeystore(bob) = %q, %v; want it untouched", ref, err)
	}

	// The defaults are not safes.
	iter, err := s.ListAllSafes()
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	if iter.Next() {
		t.Error("ListAllSafes lists a default keystore record")
	}
}

func TestSafeStorageRefusals(t *testing.T) {
	s := newStorage(t)
	owner, _ := happydns.NewRandomIdentifier()

	for name, safe := range map[string]*happydns.Safe{
		"no id":    {Owner: owner, Kind: "instance"},
		"no owner": {Id: happydns.Identifier{0x01}, Kind: "instance"},
		"no kind":  {Id: happydns.Identifier{0x01}, Owner: owner},
	} {
		if err := s.CreateSafe(safe); err == nil {
			t.Errorf("CreateSafe(%s) succeeded", name)
		}
	}

	safe := newSafe(owner, "instance")
	if err := s.CreateSafe(safe); err != nil {
		t.Fatal(err)
	}

	moved := *safe
	moved.Owner, _ = happydns.NewRandomIdentifier()
	if err := s.UpdateSafe(&moved); err == nil {
		t.Error("UpdateSafe changed the owner of a safe")
	}
	rekinded := *safe
	rekinded.Kind = "other"
	if err := s.UpdateSafe(&rekinded); err == nil {
		t.Error("UpdateSafe changed the kind of a safe")
	}
	if got, err := s.GetSafeByOwner(owner, "instance"); err != nil || !got.Id.Equals(safe.Id) {
		t.Errorf("GetSafeByOwner after refused updates = %v, %v", got, err)
	}

	unknown := newSafe(owner, "instance")
	if err := s.UpdateSafe(unknown); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("UpdateSafe(unknown) = %v, want ErrSafeNotFound", err)
	}
	if err := s.DeleteSafe(unknown.Id); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("DeleteSafe(unknown) = %v, want ErrSafeNotFound", err)
	}
}

func TestListDefaultKeystores(t *testing.T) {
	s := newStorage(t)
	alice, _ := happydns.NewRandomIdentifier()
	bob, _ := happydns.NewRandomIdentifier()

	if got, err := s.ListDefaultKeystores(); err != nil || len(got) != 0 {
		t.Fatalf("ListDefaultKeystores before any = %v, %v; want none", got, err)
	}

	if err := s.PutDefaultKeystore(alice, "abc"); err != nil {
		t.Fatal(err)
	}
	if err := s.PutDefaultKeystore(bob, "instance"); err != nil {
		t.Fatal(err)
	}

	// A safe and its index are not defaults.
	if err := s.CreateSafe(&happydns.Safe{Id: happydns.Identifier{1}, Owner: alice, Kind: "instance"}); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListDefaultKeystores()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[alice.String()] != "abc" || got[bob.String()] != "instance" {
		t.Errorf("ListDefaultKeystores = %v, want alice and bob only", got)
	}

	if err := s.DeleteDefaultKeystore(alice); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.ListDefaultKeystores(); len(got) != 1 || got[bob.String()] != "instance" {
		t.Errorf("ListDefaultKeystores after delete = %v, want bob only", got)
	}
}
