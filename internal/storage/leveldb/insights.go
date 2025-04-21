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
	"time"

	"github.com/syndtr/goleveldb/leveldb"

	"git.happydns.org/happyDomain/model"
)

func (s *LevelDBStorage) InsightsRun() error {
	return s.put("insights", time.Now())
}

func (s *LevelDBStorage) LastInsightsRun() (t *time.Time, instance happydns.Identifier, err error) {
	err = s.get("insights.instance-id", &instance)
	if err == leveldb.ErrNotFound {
		// No instance ID defined, set one
		instance, err = happydns.NewRandomIdentifier()
		if err != nil {
			return
		}

		err = s.put("insights.instance-id", instance)
		if err != nil {
			return
		}
	} else if err != nil {
		return
	}

	t = new(time.Time)
	err = s.get("insights", &t)
	if err == leveldb.ErrNotFound {
		t = nil
		err = nil
	} else if err != nil {
		return
	}

	return
}
