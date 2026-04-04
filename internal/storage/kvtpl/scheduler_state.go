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

package database

import (
	"errors"
	"time"

	"git.happydns.org/happyDomain/model"
)

const schedulerLastRunKey = "scheduler-lastrun"

func (s *KVStorage) GetLastSchedulerRun() (time.Time, error) {
	var t time.Time
	err := s.db.Get(schedulerLastRunKey, &t)
	if errors.Is(err, happydns.ErrNotFound) {
		return time.Time{}, nil
	}
	return t, err
}

func (s *KVStorage) SetLastSchedulerRun(t time.Time) error {
	return s.db.Put(schedulerLastRunKey, t)
}
