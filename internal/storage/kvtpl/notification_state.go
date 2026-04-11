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

package database

import (
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

func notifStateKey(checkerID string, target happydns.CheckTarget, userId happydns.Identifier) string {
	return fmt.Sprintf("notifstate|%s|%s|%s", userId.String(), checkerID, target.String())
}

func (s *KVStorage) GetState(checkerID string, target happydns.CheckTarget, userId happydns.Identifier) (*happydns.NotificationState, error) {
	state := &happydns.NotificationState{}
	err := s.db.Get(notifStateKey(checkerID, target, userId), state)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrNotificationStateNotFound
	}
	return state, err
}

func (s *KVStorage) PutState(state *happydns.NotificationState) error {
	return s.db.Put(notifStateKey(state.CheckerID, state.Target, state.UserId), state)
}

func (s *KVStorage) DeleteState(checkerID string, target happydns.CheckTarget, userId happydns.Identifier) error {
	return s.db.Delete(notifStateKey(checkerID, target, userId))
}

func (s *KVStorage) ListStatesByUser(userId happydns.Identifier) ([]*happydns.NotificationState, error) {
	prefix := fmt.Sprintf("notifstate|%s|", userId.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	var states []*happydns.NotificationState
	for iter.Next() {
		var state happydns.NotificationState
		if err := s.db.DecodeData(iter.Value(), &state); err != nil {
			continue
		}
		states = append(states, &state)
	}
	return states, nil
}
