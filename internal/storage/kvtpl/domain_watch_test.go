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

func TestCreateDomainWatch(t *testing.T) {
	s := newStorage(t)
	expiration := time.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC)
	since := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	w := &happydns.DomainWatch{
		UserId:         happydns.Identifier{0x01},
		DomainName:     "xn--bcher-kva.example",
		DisplayName:    "bücher.example",
		Note:           "for the shop",
		Phase:          happydns.DomainWatchRedemption,
		PhaseSince:     &since,
		ExpirationDate: &expiration,
		Status:         []string{"redemptionPeriod"},
		History:        []happydns.DomainWatchEvent{{At: since, From: happydns.DomainWatchGrace, To: happydns.DomainWatchRedemption}},
	}
	if err := s.CreateDomainWatch(w); err != nil {
		t.Fatal(err)
	}
	if w.Id.IsEmpty() {
		t.Fatal("CreateDomainWatch gave the watch no identifier")
	}

	got, err := s.GetDomainWatch(w.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got.DomainName != w.DomainName || got.DisplayName != w.DisplayName || got.Note != w.Note || got.Phase != w.Phase {
		t.Errorf("GetDomainWatch = %+v, want %+v", got, w)
	}
	if got.PhaseSince == nil || !got.PhaseSince.Equal(since) || got.ExpirationDate == nil || !got.ExpirationDate.Equal(expiration) {
		t.Errorf("GetDomainWatch dates = %v, %v; want %v, %v", got.PhaseSince, got.ExpirationDate, since, expiration)
	}
	if len(got.History) != 1 || got.History[0] != w.History[0] || len(got.Status) != 1 || got.Status[0] != "redemptionPeriod" {
		t.Errorf("GetDomainWatch history and status = %+v, %v", got.History, got.Status)
	}
}

func TestUpdateDomainWatch(t *testing.T) {
	s := newStorage(t)
	owner := happydns.Identifier{0x01}
	w := &happydns.DomainWatch{UserId: owner, DomainName: "example.com"}
	if err := s.CreateDomainWatch(w); err != nil {
		t.Fatal(err)
	}

	w.Note = "updated"
	if err := s.UpdateDomainWatch(w); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListDomainWatchesByUser(owner)
	if err != nil || len(got) != 1 || got[0].Note != "updated" {
		t.Errorf("ListDomainWatchesByUser = %v, %v; want the updated watch, once", got, err)
	}
}

func TestListDomainWatchesByUser(t *testing.T) {
	s := newStorage(t)
	alice, bob := happydns.Identifier{0x01}, happydns.Identifier{0x02}
	for _, w := range []*happydns.DomainWatch{
		{UserId: alice, DomainName: "a.example"},
		{UserId: alice, DomainName: "b.example"},
		{UserId: bob, DomainName: "c.example"},
	} {
		if err := s.CreateDomainWatch(w); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.ListDomainWatchesByUser(bob)
	if err != nil || len(got) != 1 || got[0].DomainName != "c.example" {
		t.Errorf("ListDomainWatchesByUser(bob) = %v, %v; want only the watch of bob", got, err)
	}
	if got, err := s.ListDomainWatchesByUser(alice); err != nil || len(got) != 2 {
		t.Errorf("ListDomainWatchesByUser(alice) = %v, %v; want her 2 watches", got, err)
	}
}

func TestListAllDomainWatches(t *testing.T) {
	s := newStorage(t)
	for _, owner := range []happydns.Identifier{{0x01}, {0x02}} {
		if err := s.CreateDomainWatch(&happydns.DomainWatch{UserId: owner, DomainName: "example.com"}); err != nil {
			t.Fatal(err)
		}
	}

	iter, err := s.ListAllDomainWatches()
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	n := 0
	for iter.Next() {
		if iter.Item().DomainName != "example.com" {
			t.Errorf("listed watch = %+v, want the stored one", iter.Item())
		}
		n++
	}
	if n != 2 {
		t.Errorf("ListAllDomainWatches listed %d watches, want 2", n)
	}
}

func TestDeleteDomainWatch(t *testing.T) {
	s := newStorage(t)
	owner := happydns.Identifier{0x01}
	w := &happydns.DomainWatch{UserId: owner, DomainName: "example.com"}
	if err := s.CreateDomainWatch(w); err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteDomainWatch(w.Id); err != nil {
		t.Fatalf("DeleteDomainWatch: %v", err)
	}
	if got, err := s.ListDomainWatchesByUser(owner); err != nil || len(got) != 0 {
		t.Errorf("ListDomainWatchesByUser = %v, %v; want nothing", got, err)
	}
	if _, err := s.GetDomainWatch(w.Id); err != happydns.ErrDomainWatchNotFound {
		t.Errorf("GetDomainWatch of a deleted watch: %v, want ErrDomainWatchNotFound", err)
	}
	if err := s.DeleteDomainWatch(w.Id); !errors.Is(err, happydns.ErrDomainWatchNotFound) {
		t.Errorf("DeleteDomainWatch of a deleted watch: %v, want ErrDomainWatchNotFound", err)
	}
}

// A restore puts a watch back under its identifier, reachable from its user,
// whether it is missing or already there, even under another user.
func TestRestoreDomainWatch(t *testing.T) {
	s := newStorage(t)
	id, _ := happydns.NewRandomIdentifier()
	first, second := happydns.Identifier{0x01}, happydns.Identifier{0x02}

	if err := s.RestoreDomainWatch(&happydns.DomainWatch{Id: id, UserId: first, DomainName: "example.com", Note: "a"}); err != nil {
		t.Fatalf("RestoreDomainWatch(missing): %v", err)
	}
	if got, err := s.ListDomainWatchesByUser(first); err != nil || len(got) != 1 || !got[0].Id.Equals(id) || got[0].Note != "a" {
		t.Fatalf("ListDomainWatchesByUser = %v, %v; want the restored watch", got, err)
	}

	if err := s.RestoreDomainWatch(&happydns.DomainWatch{Id: id, UserId: second, DomainName: "example.com", Note: "b"}); err != nil {
		t.Fatalf("RestoreDomainWatch(existing): %v", err)
	}
	if got, _ := s.ListDomainWatchesByUser(first); len(got) != 0 {
		t.Errorf("the previous user still lists %d watches", len(got))
	}
	if got, _ := s.ListDomainWatchesByUser(second); len(got) != 1 || got[0].Note != "b" {
		t.Errorf("ListDomainWatchesByUser(new user) = %v, want the restored watch", got)
	}

	if err := s.RestoreDomainWatch(&happydns.DomainWatch{UserId: first}); err == nil {
		t.Error("RestoreDomainWatch without an identifier succeeded")
	}
}
