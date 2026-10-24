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

package usecase

import (
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type orphanItem struct {
	Id     string
	UserId happydns.Identifier
}

type sliceIterator[T any] struct {
	items []*T
	pos   int
}

func (it *sliceIterator[T]) Next() bool          { it.pos++; return it.pos <= len(it.items) }
func (it *sliceIterator[T]) NextWithError() bool { return it.Next() }
func (it *sliceIterator[T]) Item() *T            { return it.items[it.pos-1] }
func (it *sliceIterator[T]) DropItem() error     { return nil }
func (it *sliceIterator[T]) Key() string         { return "" }
func (it *sliceIterator[T]) Raw() any            { return nil }
func (it *sliceIterator[T]) Err() error          { return nil }
func (it *sliceIterator[T]) Close()              {}

// userStore knows the users of alive, and fails to look up the others with
// err.
type userStore struct {
	storage.Storage
	alive happydns.Identifier
	err   error
}

func (s *userStore) GetUser(id happydns.Identifier) (*happydns.User, error) {
	if id.Equals(s.alive) {
		return &happydns.User{Id: id}, nil
	}
	return nil, s.err
}

var errItemNotFound = errors.New("item not found")

func tidyOrphanItems(t *testing.T, err error, items []*orphanItem, remove func(*orphanItem) error, notFound error) error {
	t.Helper()
	tu := &tidyUpUsecase{store: &userStore{alive: items[0].UserId, err: err}}
	return tidyUserOrphans(tu, &sliceIterator[orphanItem]{items: items}, false,
		func(item *orphanItem) happydns.Identifier { return item.UserId },
		func(item *orphanItem) string { return "item " + item.Id },
		remove,
		notFound,
	)
}

func TestTidyUserOrphans(t *testing.T) {
	items := []*orphanItem{
		{Id: "kept", UserId: happydns.Identifier{1}},
		{Id: "orphan", UserId: happydns.Identifier{2}},
	}

	t.Run("removes the items of the users gone", func(t *testing.T) {
		var removed []string
		err := tidyOrphanItems(t, happydns.ErrUserNotFound, items, func(item *orphanItem) error {
			removed = append(removed, item.Id)
			return nil
		}, errItemNotFound)
		if err != nil {
			t.Fatal(err)
		}
		if len(removed) != 1 || removed[0] != "orphan" {
			t.Errorf("removed %v, want [orphan]", removed)
		}
	})

	t.Run("stops on a user lookup failure", func(t *testing.T) {
		boom := errors.New("boom")
		err := tidyOrphanItems(t, boom, items, func(item *orphanItem) error {
			t.Errorf("%s removed although its user could not be looked up", item.Id)
			return nil
		}, errItemNotFound)
		if !errors.Is(err, boom) {
			t.Errorf("err = %v, want %v", err, boom)
		}
	})

	t.Run("ignores an item already gone", func(t *testing.T) {
		err := tidyOrphanItems(t, happydns.ErrUserNotFound, items, func(*orphanItem) error {
			return errItemNotFound
		}, errItemNotFound)
		if err != nil {
			t.Errorf("err = %v, want nil", err)
		}
	})

	t.Run("reports a failed removal", func(t *testing.T) {
		err := tidyOrphanItems(t, happydns.ErrUserNotFound, items, func(*orphanItem) error {
			return errItemNotFound
		}, nil)
		if !errors.Is(err, errItemNotFound) {
			t.Errorf("err = %v, want %v", err, errItemNotFound)
		}
	})
}
