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

package inmemory

import (
	"time"

	"git.happydns.org/happyDomain/model"
)

// InsightsRun registers a insights process run just now.
func (s *InMemoryStorage) InsightsRun() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.lastInsightsRun = &now

	return nil
}

// LastInsightsRun gets the last time insights process run.
func (s *InMemoryStorage) LastInsightsRun() (*time.Time, happydns.Identifier, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastInsightsID == nil {
		instance, err := happydns.NewRandomIdentifier()
		if err != nil {
			return nil, nil, err
		}
		s.lastInsightsID = instance
	}

	return s.lastInsightsRun, s.lastInsightsID, nil
}
