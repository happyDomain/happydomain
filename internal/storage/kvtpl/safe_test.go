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

func TestReplaceSafe(t *testing.T) {
	s := newStorage(t)
	owner, _ := happydns.NewRandomIdentifier()
	safe := newSafe(owner, "instance")
	if err := s.CreateSafe(safe); err != nil {
		t.Fatal(err)
	}

	err := s.ReplaceSafe(safe.Id, func(got *happydns.Safe) (*happydns.Safe, error) {
		if !got.Id.Equals(safe.Id) || len(got.Keyring) != 1 {
			t.Errorf("update got %+v, want the stored safe", got)
		}
		got.Keyring[0].Blob = []byte{9}
		return got, nil
	})
	if err != nil {
		t.Fatalf("ReplaceSafe: %v", err)
	}
	if got, _ := s.GetSafe(safe.Id); len(got.Keyring) != 1 || got.Keyring[0].Blob[0] != 9 {
		t.Errorf("stored = %+v, want the replaced keyring", got)
	}

	// Nothing to do, or a failing update: nothing written.
	if err := s.ReplaceSafe(safe.Id, func(*happydns.Safe) (*happydns.Safe, error) { return nil, nil }); err != nil {
		t.Errorf("ReplaceSafe with nothing to do = %v", err)
	}
	boom := errors.New("boom")
	err = s.ReplaceSafe(safe.Id, func(got *happydns.Safe) (*happydns.Safe, error) {
		got.Keyring = nil
		return nil, boom
	})
	if !errors.Is(err, boom) {
		t.Errorf("ReplaceSafe with a failing update = %v, want its error", err)
	}
	if got, _ := s.GetSafe(safe.Id); len(got.Keyring) != 1 {
		t.Errorf("a failed update was written: %+v", got)
	}

	// The owner and kind cannot change.
	other, _ := happydns.NewRandomIdentifier()
	err = s.ReplaceSafe(safe.Id, func(got *happydns.Safe) (*happydns.Safe, error) {
		got.Owner = other
		return got, nil
	})
	if err == nil {
		t.Error("ReplaceSafe changed the owner of a safe")
	}

	// Missing: update is not even called.
	missing, _ := happydns.NewRandomIdentifier()
	err = s.ReplaceSafe(missing, func(*happydns.Safe) (*happydns.Safe, error) {
		t.Error("update called for a missing safe")
		return nil, nil
	})
	if !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("ReplaceSafe on a missing safe = %v, want ErrSafeNotFound", err)
	}
}

// What happens between the read and the write wins: the replacement is
// dropped, and a deleted safe is not brought back.
func TestReplaceSafeLosesToConcurrentWrites(t *testing.T) {
	s := newStorage(t)
	owner, _ := happydns.NewRandomIdentifier()
	safe := newSafe(owner, "instance")
	if err := s.CreateSafe(safe); err != nil {
		t.Fatal(err)
	}

	err := s.ReplaceSafe(safe.Id, func(got *happydns.Safe) (*happydns.Safe, error) {
		concurrent := *got
		concurrent.Keyring = []happydns.WrappedKeyset{{KEK: "instance", Blob: []byte{7}}}
		if err := s.UpdateSafe(&concurrent); err != nil {
			t.Fatal(err)
		}
		got.Keyring[0].Blob = []byte{9}
		return got, nil
	})
	if !errors.Is(err, happydns.ErrChangedMeanwhile) {
		t.Errorf("ReplaceSafe over a concurrent update = %v, want ErrChangedMeanwhile", err)
	}
	if got, _ := s.GetSafe(safe.Id); got.Keyring[0].Blob[0] != 7 {
		t.Errorf("stored = %+v, want the concurrent update kept", got)
	}

	err = s.ReplaceSafe(safe.Id, func(got *happydns.Safe) (*happydns.Safe, error) {
		if err := s.DeleteSafe(safe.Id); err != nil {
			t.Fatal(err)
		}
		return got, nil
	})
	if !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("ReplaceSafe over a concurrent delete = %v, want ErrSafeNotFound", err)
	}
	if _, err := s.GetSafe(safe.Id); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("a deleted safe was brought back: %v", err)
	}
}

// A safe coming from a backup is stored beside the one its owner already has,
// rather than refused: it opens what was sealed in it, while new secrets keep
// going to the one in place.
func TestRestoreSafe(t *testing.T) {
	s := newStorage(t)
	owner, _ := happydns.NewRandomIdentifier()

	// Alone, a restored safe becomes the one of its owner.
	first := newSafe(owner, "instance")
	if err := s.RestoreSafe(first); err != nil {
		t.Fatalf("RestoreSafe: %v", err)
	}
	if got, err := s.GetSafeByOwner(owner, "instance"); err != nil || !got.Id.Equals(first.Id) {
		t.Errorf("GetSafeByOwner = %v, %v; want the restored safe", got, err)
	}

	// Beside it, another one is stored without taking its place.
	second := newSafe(owner, "instance")
	if err := s.RestoreSafe(second); err != nil {
		t.Fatalf("RestoreSafe beside an existing safe: %v", err)
	}
	if _, err := s.GetSafe(second.Id); err != nil {
		t.Errorf("the second restored safe was not stored: %v", err)
	}
	if got, err := s.GetSafeByOwner(owner, "instance"); err != nil || !got.Id.Equals(first.Id) {
		t.Errorf("GetSafeByOwner = %v, %v; want the safe in place", got, err)
	}

	if err := s.RestoreSafe(second); !errors.Is(err, happydns.ErrAlreadyExists) {
		t.Errorf("RestoreSafe with a taken identifier = %v, want ErrAlreadyExists", err)
	}
	bad := newSafe(owner, "instance")
	bad.Id = happydns.Identifier{0x01}
	if err := s.RestoreSafe(bad); !errors.Is(err, happydns.ErrInvalidIdentifier) {
		t.Errorf("RestoreSafe with a malformed identifier = %v, want ErrInvalidIdentifier", err)
	}
}

// Deleting a safe the owner index does not point to leaves the index alone:
// the owner keeps sealing into the safe in place.
func TestDeleteSafeKeepsTheIndexOfAnother(t *testing.T) {
	s := newStorage(t)
	owner, _ := happydns.NewRandomIdentifier()

	inPlace := newSafe(owner, "instance")
	beside := newSafe(owner, "instance")
	for _, safe := range []*happydns.Safe{inPlace, beside} {
		if err := s.RestoreSafe(safe); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.DeleteSafe(beside.Id); err != nil {
		t.Fatalf("DeleteSafe: %v", err)
	}
	if got, err := s.GetSafeByOwner(owner, "instance"); err != nil || !got.Id.Equals(inPlace.Id) {
		t.Errorf("GetSafeByOwner after deleting the other safe = %v, %v; want the safe in place", got, err)
	}

	if err := s.DeleteSafe(inPlace.Id); err != nil {
		t.Fatalf("DeleteSafe: %v", err)
	}
	if _, err := s.GetSafeByOwner(owner, "instance"); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("GetSafeByOwner after deleting the safe in place = %v, want ErrSafeNotFound", err)
	}
}
