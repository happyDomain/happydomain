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

// Per-resource tidy methods live in tidy_*.go siblings (users, sessions,
// providers, notifications, domains, zones, checks). This file holds the type, the shared
// iterator helper, and the TidyAll orchestrator.

package usecase

import (
	"errors"
	"fmt"
	"log"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type tidyUpUsecase struct {
	store storage.Storage

	unlocked ScopePurger
}

// ScopePurger forgets what was unlocked during the sessions for which alive
// returns false.
type ScopePurger interface {
	PurgeScopes(alive func(scope string) bool)
}

// TidyOption configures the tidy use case.
type TidyOption func(*tidyUpUsecase)

// TidyUnlockedKeystores has the tidy of the sessions also forget the
// keystores unlocked during those gone.
func TidyUnlockedKeystores(p ScopePurger) TidyOption {
	return func(tu *tidyUpUsecase) { tu.unlocked = p }
}

func NewTidyUpUsecase(store storage.Storage, opts ...TidyOption) happydns.TidyUpUseCase {
	tu := &tidyUpUsecase{
		store: store,
	}
	for _, opt := range opts {
		opt(tu)
	}
	return tu
}

// iterateTidy drives an iterator using NextWithError so Tidy can decide
// whether to delete undecodable records (via DropItem) or just log them.
// handle is only invoked for successfully decoded items.
func iterateTidy[T any](iter happydns.Iterator[T], dropInvalid bool, handle func(*T) error) error {
	for iter.NextWithError() {
		item := iter.Item()
		if item == nil {
			key := iter.Key()
			log.Printf("KVIterator: error decoding item at key %q: %s", key, iter.Err())
			if dropInvalid {
				if err := iter.DropItem(); err != nil {
					log.Printf("KVIterator: failed to delete invalid item at key %q: %s", key, err)
				} else {
					log.Printf("KVIterator: dropped invalid item at key %q", key)
				}
			}
			continue
		}
		if err := handle(item); err != nil {
			return err
		}
	}
	return iter.Err()
}

// tidyUserOrphans deletes, with remove, the items of iter whose user, told by
// userOf, no longer exists. describe names an item in the logs. A removal
// failing with notFound, when not nil, is ignored: the item is already gone.
// iter is closed before anything is deleted.
func tidyUserOrphans[T any](tu *tidyUpUsecase, iter happydns.Iterator[T], dropInvalid bool, userOf func(*T) happydns.Identifier, describe func(*T) string, remove func(*T) error, notFound error) error {
	var orphans []*T
	err := iterateTidy(iter, dropInvalid, func(item *T) error {
		_, err := tu.store.GetUser(userOf(item))
		if errors.Is(err, happydns.ErrUserNotFound) {
			orphans = append(orphans, item)
			return nil
		}
		return err
	})
	iter.Close()
	if err != nil {
		return err
	}

	for _, item := range orphans {
		log.Printf("Deleting orphan %s (user not found)", describe(item))
		if err := remove(item); err != nil && (notFound == nil || !errors.Is(err, notFound)) {
			return fmt.Errorf("unable to delete orphan %s: %w", describe(item), err)
		}
	}

	return nil
}

func (tu *tidyUpUsecase) TidyAll(dropInvalid bool) error {
	for _, tidy := range []func(bool) error{
		tu.TidySessions,
		tu.TidyAuthUsers,
		tu.TidyUsers,
		tu.TidyProviders,
		tu.TidyNotificationChannels,
		tu.TidyNotificationPreferences,
		tu.TidyNotificationStates,
		tu.TidyNotificationRecords,
		tu.TidySafes,
		tu.TidyDomains,
		tu.TidyZones,
		tu.TidyDomainLogs,
		tu.TidyCheckPlans,
		tu.TidyCheckerConfigurations,
		tu.TidyExecutions,
		tu.TidyCheckEvaluations,
		tu.TidySnapshots,
		tu.TidyObservationCache,
	} {
		if err := tidy(dropInvalid); err != nil {
			return err
		}
	}
	return nil
}
