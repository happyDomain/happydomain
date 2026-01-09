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

func (s *KVStorage) InsightsRun() error {
	return s.db.Put("insights", time.Now())
}

func (s *KVStorage) LastInsightsRun() (t *time.Time, instance happydns.Identifier, err error) {
	err = s.db.Get("insights.instance-id", &instance)
	if errors.Is(err, happydns.ErrNotFound) {
		// No instance ID defined, set one
		instance, err = happydns.NewRandomIdentifier()
		if err != nil {
			return
		}

		err = s.db.Put("insights.instance-id", instance)
		if err != nil {
			return
		}
	} else if err != nil {
		return
	}

	t = new(time.Time)
	err = s.db.Get("insights", &t)
	if errors.Is(err, happydns.ErrNotFound) {
		t = nil
		err = nil
	} else if err != nil {
		return
	}

	return
}
