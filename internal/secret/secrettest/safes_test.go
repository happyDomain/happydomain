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

package secrettest_test

import (
	"errors"
	"slices"
	"testing"

	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/model"
)

// id returns a well-formed identifier starting with b.
func id(b byte) happydns.Identifier {
	i := make(happydns.Identifier, happydns.IDENTIFIER_LEN)
	i[0] = b
	return i
}

func newSafe(i, owner byte) *happydns.Safe {
	return &happydns.Safe{
		Id:      id(i),
		Owner:   id(owner),
		Kind:    "instance",
		Keyring: []happydns.WrappedKeyset{{KEK: "instance", Blob: []byte("blob")}},
	}
}

// A second safe of a kind for one owner is refused with the error the real
// storage returns, which the code creating safes concurrently relies on.
func TestCreateSafeRefusesASecondSafeOfAKind(t *testing.T) {
	safes := secrettest.NewSafes()

	if err := safes.CreateSafe(newSafe(0x10, 0x01)); err != nil {
		t.Fatal(err)
	}
	err := safes.CreateSafe(newSafe(0x11, 0x01))
	if !errors.Is(err, happydns.ErrAlreadyExists) {
		t.Errorf("second CreateSafe = %v, want ErrAlreadyExists", err)
	}
	if safes.Len() != 1 || safes.Creates() != 1 {
		t.Errorf("Len, Creates = %d, %d; want 1, 1", safes.Len(), safes.Creates())
	}
}

// An identifier already taken, even by the safe of another owner, is refused
// rather than overwritten.
func TestCreateSafeRefusesATakenIdentifier(t *testing.T) {
	safes := secrettest.NewSafes()

	if err := safes.CreateSafe(newSafe(0x10, 0x01)); err != nil {
		t.Fatal(err)
	}
	err := safes.CreateSafe(newSafe(0x10, 0x02))
	if !errors.Is(err, happydns.ErrAlreadyExists) {
		t.Errorf("CreateSafe on a taken identifier = %v, want ErrAlreadyExists", err)
	}
	if got, _ := safes.GetSafe(id(0x10)); got == nil || !got.Owner.Equals(id(0x01)) {
		t.Errorf("safe 0x10 = %+v, want the first owner's", got)
	}
	if safes.Creates() != 1 {
		t.Errorf("Creates = %d, want 1", safes.Creates())
	}
}

// A safe without owner, kind or well-formed identifier is refused, as the
// real storage does.
func TestCreateSafeRefusesAnIncompleteSafe(t *testing.T) {
	for name, tc := range map[string]struct {
		safe *happydns.Safe
		want error
	}{
		"no owner":         {&happydns.Safe{Id: id(0x10), Kind: "instance"}, nil},
		"no kind":          {&happydns.Safe{Id: id(0x10), Owner: id(0x01)}, nil},
		"no identifier":    {&happydns.Safe{Owner: id(0x01), Kind: "instance"}, happydns.ErrInvalidIdentifier},
		"short identifier": {&happydns.Safe{Id: happydns.Identifier{0x10}, Owner: id(0x01), Kind: "instance"}, happydns.ErrInvalidIdentifier},
	} {
		t.Run(name, func(t *testing.T) {
			safes := secrettest.NewSafes()
			err := safes.CreateSafe(tc.safe)
			if err == nil || (tc.want != nil && !errors.Is(err, tc.want)) {
				t.Errorf("CreateSafe = %v, want an error (%v)", err, tc.want)
			}
			if safes.Len() != 0 || safes.Creates() != 0 {
				t.Errorf("Len, Creates = %d, %d; want nothing stored", safes.Len(), safes.Creates())
			}
		})
	}
}

// GetSafeByOwner follows the owner index, as the real storage does, rather
// than returning any safe of the owner and kind.
func TestGetSafeByOwnerFollowsTheIndex(t *testing.T) {
	safes := secrettest.NewSafes()
	if err := safes.CreateSafe(newSafe(0x10, 0x01)); err != nil {
		t.Fatal(err)
	}
	// Restored or concurrent: a second safe of the kind, not indexed.
	for b := byte(0x11); b < 0x20; b++ {
		safes.Put(*newSafe(b, 0x01))
	}

	for range 20 {
		got, err := safes.GetSafeByOwner(id(0x01), "instance")
		if err != nil || !got.Id.Equals(id(0x10)) {
			t.Fatalf("GetSafeByOwner = %v, %v; want the indexed safe 0x10", got, err)
		}
	}

	if _, err := safes.GetSafeByOwner(id(0x02), "instance"); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("GetSafeByOwner(unknown owner) = %v, want ErrSafeNotFound", err)
	}
}

// What the storage hands out or keeps is a copy, as decoding a record is:
// changing it in place changes nothing stored.
func TestSafesAreCopies(t *testing.T) {
	safes := secrettest.NewSafes()
	created := newSafe(0x10, 0x01)
	if err := safes.CreateSafe(created); err != nil {
		t.Fatal(err)
	}
	put := newSafe(0x20, 0x02)
	safes.Put(*put)

	// Change what was given to the storage.
	created.Keyring[0].Blob[0] = 'X'
	put.Keyring[0].Blob[0] = 'X'

	// Change what the storage handed out.
	if s, err := safes.GetSafe(id(0x10)); err == nil {
		s.Keyring[0].Blob[0] = 'Y'
		s.Owner[0] = 0xff
	}
	if s, err := safes.GetSafeByOwner(id(0x01), "instance"); err == nil {
		s.Keyring[0].Blob[1] = 'Y'
	}
	if iter, err := safes.ListAllSafes(); err == nil {
		for iter.Next() {
			iter.Item().Keyring[0].Blob[2] = 'Y'
		}
	}
	for _, i := range []happydns.Identifier{id(0x10), id(0x20)} {
		s, err := safes.GetSafe(i)
		if err != nil {
			t.Fatal(err)
		}
		if string(s.Keyring[0].Blob) != "blob" || s.Owner[0] == 0xff {
			t.Errorf("safe %s = %+v, changed in place", i, s)
		}
	}
}

// ListAllSafes lists every record, in the order of their keys, as the real
// storage does.
func TestListAllSafesIsOrdered(t *testing.T) {
	safes := secrettest.NewSafes()
	for _, b := range []byte{0x30, 0x10, 0x20} {
		safes.Put(*newSafe(b, b))
	}

	var got []string
	iter, err := safes.ListAllSafes()
	if err != nil {
		t.Fatal(err)
	}
	for iter.Next() {
		got = append(got, iter.Item().Id.String())
	}
	var want []string
	for _, b := range []byte{0x10, 0x20, 0x30} {
		i := id(b)
		want = append(want, i.String())
	}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("ListAllSafes = %v, want %v", got, want)
	}
}

// UpdateSafe refuses an unknown safe and a change of owner or kind, and
// counts what it writes.
func TestUpdateSafe(t *testing.T) {
	safes := secrettest.NewSafes()
	if err := safes.CreateSafe(newSafe(0x10, 0x01)); err != nil {
		t.Fatal(err)
	}

	if err := safes.UpdateSafe(newSafe(0x11, 0x01)); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("UpdateSafe(unknown) = %v, want ErrSafeNotFound", err)
	}
	otherOwner := newSafe(0x10, 0x02)
	otherKind := newSafe(0x10, 0x01)
	otherKind.Kind = "password"
	for _, s := range []*happydns.Safe{otherOwner, otherKind} {
		if err := safes.UpdateSafe(s); err == nil {
			t.Errorf("UpdateSafe(%+v) accepted a change of owner or kind", s)
		}
	}
	if safes.Updates() != 0 {
		t.Errorf("Updates = %d after refused updates, want 0", safes.Updates())
	}

	updated := newSafe(0x10, 0x01)
	updated.Keyring[0].Blob = []byte("new")
	if err := safes.UpdateSafe(updated); err != nil {
		t.Fatal(err)
	}
	if s, _ := safes.GetSafe(id(0x10)); string(s.Keyring[0].Blob) != "new" {
		t.Errorf("after UpdateSafe, blob = %q", s.Keyring[0].Blob)
	}
	if safes.Updates() != 1 {
		t.Errorf("Updates = %d, want 1", safes.Updates())
	}
}

// DeleteSafe refuses an unknown safe, and removes the owner index with the
// safe.
func TestDeleteSafe(t *testing.T) {
	safes := secrettest.NewSafes()
	if err := safes.CreateSafe(newSafe(0x10, 0x01)); err != nil {
		t.Fatal(err)
	}

	if err := safes.DeleteSafe(id(0x12)); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("DeleteSafe(unknown) = %v, want ErrSafeNotFound", err)
	}

	if err := safes.DeleteSafe(id(0x10)); err != nil {
		t.Fatal(err)
	}
	if _, err := safes.GetSafeByOwner(id(0x01), "instance"); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("after DeleteSafe, GetSafeByOwner = %v, want ErrSafeNotFound", err)
	}
	if err := safes.CreateSafe(newSafe(0x13, 0x01)); err != nil {
		t.Errorf("CreateSafe after deleting the owner's safe = %v", err)
	}
	if safes.Deletes() != 1 || safes.Creates() != 2 {
		t.Errorf("Deletes, Creates = %d, %d; want 1, 2", safes.Deletes(), safes.Creates())
	}
}

// Put stores a record as is, as written behind the storage's back: it
// bypasses the rules and the counters, and leaves the owner index alone.
func TestPut(t *testing.T) {
	safes := secrettest.NewSafes()
	safes.Put(happydns.Safe{Id: happydns.Identifier{0x12}, Kind: "instance"})
	safes.Put(*newSafe(0x13, 0x01))

	if safes.Len() != 2 || safes.Creates() != 0 || safes.Updates() != 0 {
		t.Errorf("Len, Creates, Updates = %d, %d, %d; want 2, 0, 0", safes.Len(), safes.Creates(), safes.Updates())
	}
	if _, err := safes.GetSafe(happydns.Identifier{0x12}); err != nil {
		t.Errorf("GetSafe(put) = %v", err)
	}
	if _, err := safes.GetSafeByOwner(id(0x01), "instance"); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("GetSafeByOwner after Put = %v, want ErrSafeNotFound: Put writes no index", err)
	}
}
