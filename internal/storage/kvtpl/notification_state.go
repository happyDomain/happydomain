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
	"log"

	"git.happydns.org/happyDomain/model"
)

const (
	notificationStatePrimaryPrefix = "notifstate|"
)

// notifStateKey builds a bounded key for a notification state entry.
//
// Key layout: "notifstate|" (11) + userId (22) + "|" (1) + hash28 (28) = 62 chars.
//
// The (checkerID, target) pair is SHA-256 hashed and truncated to 21 bytes so
// the total key length never exceeds 64 chars regardless of checker name or
// target field lengths.
func notifStateKey(checkerID string, target happydns.CheckTarget, userId happydns.Identifier) string {
	compound := checkerID + "|" + target.UserId + "/" + target.DomainId + "/" + target.ServiceId
	return fmt.Sprintf("%s%s|%s", notificationStatePrimaryPrefix, userId.String(), hash28(compound))
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
	prefix := fmt.Sprintf("%s%s|", notificationStatePrimaryPrefix, userId.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	var states []*happydns.NotificationState
	for iter.Next() {
		var state happydns.NotificationState
		if err := s.db.DecodeData(iter.Value(), &state); err != nil {
			log.Printf("storage: malformed notification state at %q: %v", iter.Key(), err)
			continue
		}
		states = append(states, &state)
	}
	return states, nil
}
