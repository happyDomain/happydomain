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

// AcknowledgeIssue marks an issue as acknowledged, suppressing repeat
// notifications until the next state change.
func (d *Dispatcher) AcknowledgeIssue(userId happydns.Identifier, checkerID string, target happydns.CheckTarget, acknowledgedBy string, annotation string) error {
	state, err := d.stateStore.GetState(checkerID, target, userId)
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

	now := time.Now()
	state.Acknowledged = true
	state.AcknowledgedAt = &now
	state.AcknowledgedBy = acknowledgedBy
	state.Annotation = annotation

	return d.stateStore.PutState(state)
}

// ClearAcknowledgement removes the acknowledgement from an issue.
func (d *Dispatcher) ClearAcknowledgement(userId happydns.Identifier, checkerID string, target happydns.CheckTarget) error {
	state, err := d.stateStore.GetState(checkerID, target, userId)
	if err != nil {
		return err
	}

	state.Acknowledged = false
	state.AcknowledgedAt = nil
	state.AcknowledgedBy = ""
	state.Annotation = ""

	return d.stateStore.PutState(state)
}

// GetState returns the current notification state for a checker/target/user.
func (d *Dispatcher) GetState(userId happydns.Identifier, checkerID string, target happydns.CheckTarget) (*happydns.NotificationState, error) {
	return d.stateStore.GetState(checkerID, target, userId)
}

// ListStatesByUser returns all notification states for a user.
func (d *Dispatcher) ListStatesByUser(userId happydns.Identifier) ([]*happydns.NotificationState, error) {
	return d.stateStore.ListStatesByUser(userId)
}
