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
