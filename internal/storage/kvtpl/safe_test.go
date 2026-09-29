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
	"sync"
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
	if err := s.CreateSafe(safe); !errors.Is(err, happydns.ErrAlreadyExists) {
		t.Errorf("CreateSafe twice with the same id = %v, want ErrAlreadyExists", err)
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

	// One safe of a kind per owner.
	if err := s.CreateSafe(newSafe(alice, "instance")); !errors.Is(err, happydns.ErrAlreadyExists) {
		t.Errorf("a second instance safe for the same owner = %v, want ErrAlreadyExists", err)
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

// Like providers and channels, the storage never chooses the identifier of a
// safe, and refuses one that is missing or malformed.
func TestCreateSafeRefusesInvalidId(t *testing.T) {
	for name, id := range map[string]happydns.Identifier{
		"missing":   nil,
		"too short": {0x01},
		"too long":  make(happydns.Identifier, happydns.IDENTIFIER_LEN+1),
	} {
		t.Run(name, func(t *testing.T) {
			s := newStorage(t)
			owner, _ := happydns.NewRandomIdentifier()
			safe := newSafe(owner, "instance")
			safe.Id = id

			if err := s.CreateSafe(safe); !errors.Is(err, happydns.ErrInvalidIdentifier) {
				t.Fatalf("CreateSafe = %v, want ErrInvalidIdentifier", err)
			}
			if _, err := s.GetSafeByOwner(owner, "instance"); !errors.Is(err, happydns.ErrSafeNotFound) {
				t.Errorf("a refused creation stored something: %v", err)
			}
		})
	}
}

// Replicas sharing a database each create the first safe of an owner at the
// same time: only one safe may result, and every other creation must fail
// with ErrAlreadyExists so that its caller reads the winner back.
func TestCreateSafeOnePerOwnerUnderConcurrency(t *testing.T) {
	s := newStorage(t)
	owner, _ := happydns.NewRandomIdentifier()

	const n = 32
	safes := make([]*happydns.Safe, n)
	errs := make([]error, n)
	start := make(chan struct{})

	var wg sync.WaitGroup
	for i := range n {
		safes[i] = newSafe(owner, "instance")
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = s.CreateSafe(safes[i])
		}()
	}
	close(start)
	wg.Wait()

	var winner *happydns.Safe
	for i, err := range errs {
		switch {
		case err == nil && winner == nil:
			winner = safes[i]
		case err == nil:
			t.Fatalf("two safes created for the same owner: %s and %s", winner.Id.String(), safes[i].Id.String())
		case !errors.Is(err, happydns.ErrAlreadyExists):
			t.Fatalf("CreateSafe = %v, want nil or ErrAlreadyExists", err)
		}
	}
	if winner == nil {
		t.Fatal("no safe created")
	}

	got, err := s.GetSafeByOwner(owner, "instance")
	if err != nil || !got.Id.Equals(winner.Id) {
		t.Errorf("GetSafeByOwner = %v, %v; want the only safe created %s", got, err, winner.Id.String())
	}

	// The losers leave nothing behind.
	iter, err := s.ListAllSafes()
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	count := 0
	for iter.Next() {
		count++
	}
	if count != 1 {
		t.Errorf("ListAllSafes = %d safes, want 1", count)
	}
}
