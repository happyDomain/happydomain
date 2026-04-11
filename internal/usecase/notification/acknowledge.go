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

package notification

import (
	"errors"
	"time"

	"git.happydns.org/happyDomain/model"
)

// Depends only on the state store — no senders, no preferences.
type AckService struct {
	stateStore NotificationStateStorage
	locker     *StateLocker
	// Overridable for tests.
	nowFn func() time.Time
}

// locker is shared with the Dispatcher to avoid racing on the same state record.
func NewAckService(stateStore NotificationStateStorage, locker *StateLocker) *AckService {
	return &AckService{stateStore: stateStore, locker: locker, nowFn: time.Now}
}

// Requires an existing dispatcher-created state so clients cannot materialize states by guessing IDs.
func (a *AckService) AcknowledgeIssue(userId happydns.Identifier, checkerID string, target happydns.CheckTarget, acknowledgedBy string, annotation string) error {
	unlock := a.locker.Lock(checkerID, target, userId)
	defer unlock()

	state, err := a.stateStore.GetState(checkerID, target, userId)
	if errors.Is(err, happydns.ErrNotificationStateNotFound) {
		// Create a new state if one doesn't exist yet.
		state = &happydns.NotificationState{
			CheckerID:  checkerID,
			Target:     target,
			UserId:     userId,
			LastStatus: happydns.StatusUnknown,
		}
	} else if err != nil {
		return err
	}
	// Defense in depth: storage key already encodes userId.
	if !state.UserId.Equals(userId) {
		return happydns.ErrNotificationStateNotFound
	}

	now := a.nowFn()
	state.Acknowledged = true
	state.AcknowledgedAt = &now
	state.AcknowledgedBy = acknowledgedBy
	state.Annotation = annotation

	return a.stateStore.PutState(state)
}

func (a *AckService) ClearAcknowledgement(userId happydns.Identifier, checkerID string, target happydns.CheckTarget) error {
	unlock := a.locker.Lock(checkerID, target, userId)
	defer unlock()

	state, err := a.stateStore.GetState(checkerID, target, userId)
	if err != nil {
		return err
	}

	state.ClearAcknowledgement()
	return a.stateStore.PutState(state)
}

func (a *AckService) GetState(userId happydns.Identifier, checkerID string, target happydns.CheckTarget) (*happydns.NotificationState, error) {
	return a.stateStore.GetState(checkerID, target, userId)
}

func (a *AckService) ListStatesByUser(userId happydns.Identifier) ([]*happydns.NotificationState, error) {
	return a.stateStore.ListStatesByUser(userId)
}
