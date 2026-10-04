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
	"fmt"
	"log"
	"time"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type purgeUsecase struct {
	store storage.Storage
}

func NewPurgeUsecase(store storage.Storage) happydns.PurgeUsecase {
	return &purgeUsecase{store: store}
}

func (pu *purgeUsecase) Purge(opts happydns.PurgeOptions) (*happydns.PurgeReport, error) {
	report := &happydns.PurgeReport{}
	var errs []error

	if opts.Checkers {
		n, err := pu.store.PurgeCheckerHistory()
		report.Checkers = n
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to purge checker data: %w", err))
		}
	}

	if opts.ExpiredSessions {
		n, err := pu.purgeExpiredSessions(time.Now())
		report.ExpiredSessions = n
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to purge expired sessions: %w", err))
		}
	}

	if opts.NotificationRecords {
		n, err := pu.store.ClearRecords()
		report.NotificationRecords = n
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to purge notification records: %w", err))
		}
	}

	if opts.Compact {
		compacted, err := pu.store.Compact()
		report.Compacted = compacted && err == nil
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to compact the database: %w", err))
		}
	}

	log.Printf("Purge: deleted %d checker keys, %d expired sessions, %d notification record keys; compacted: %v",
		report.Checkers, report.ExpiredSessions, report.NotificationRecords, report.Compacted)

	return report, errors.Join(errs...)
}

// purgeExpiredSessions deletes the sessions expired before now, then the
// user index entries they leave behind.
func (pu *purgeUsecase) purgeExpiredSessions(now time.Time) (int, error) {
	iter, err := pu.store.ListAllSessions()
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	n := 0
	err = iterateTidy(iter, false, func(session *happydns.Session) error {
		if session.ExpiresOn.IsZero() || !session.ExpiresOn.Before(now) {
			return nil
		}
		if err := iter.DropItem(); err != nil {
			return err
		}
		n++
		return nil
	})
	if err != nil {
		return n, err
	}

	return n, pu.store.TidySessionIndexes()
}
