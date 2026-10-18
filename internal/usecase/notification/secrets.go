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
	"context"
	"errors"
	"fmt"
	"sync"

	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
)

// channelLocks serializes the writes to one channel, so that a reseal never
// writes back a config read before a user's update.
var channelLocks sync.Map

// LockChannel locks the channel id and returns the function unlocking it.
// Every write of a channel takes it.
func LockChannel(id happydns.Identifier) func() {
	mu, _ := channelLocks.LoadOrStore(id.String(), &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	return mu.(*sync.Mutex).Unlock
}

// ChannelSecrets lets the administrator follow and migrate how the secrets of
// notification channels are stored.
type ChannelSecrets struct {
	store    NotificationChannelStorage
	registry *notifPkg.Registry
}

func NewChannelSecrets(store NotificationChannelStorage, registry *notifPkg.Registry) *ChannelSecrets {
	return &ChannelSecrets{store: store, registry: registry}
}

// InspectSecrets tells how the secrets of every channel are stored.
func (cs *ChannelSecrets) InspectSecrets(ctx context.Context) (secret.Counts, error) {
	var counts secret.Counts

	iter, err := cs.store.ListAllChannels()
	if err != nil {
		return counts, err
	}
	defer iter.Close()

	// One channel that cannot be looked at must not hide the others. It
	// is counted: a type this version does not know may hold sealed
	// secrets all the same.
	for iter.NextWithError() {
		if iter.Err() != nil {
			counts.Undecodable++
			continue
		}
		if err := cs.registry.InspectChannelConfig(ctx, iter.Item(), &counts); err != nil {
			counts.Undecodable++
		}
	}

	return counts, iter.Err()
}

// ResealSecrets stores the secrets of every channel the way the current policy
// stores new ones. A channel that fails is reported and skipped; run it again
// to resume.
func (cs *ChannelSecrets) ResealSecrets(ctx context.Context) (secret.ResealReport, error) {
	report := secret.ResealReport{ObjectType: notifPkg.SecretObjectType}

	iter, err := cs.store.ListAllChannels()
	if err != nil {
		return report, err
	}

	// Collected first: writing while iterating is not safe on every storage.
	// A record that does not decode is reported, and the others resealed.
	var ids []happydns.Identifier
	for iter.NextWithError() {
		if err := iter.Err(); err != nil {
			report.Processed++
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("record %s: %s", iter.Key(), err.Error()))
			continue
		}
		ids = append(ids, iter.Item().Id)
	}
	err = iter.Err()
	iter.Close()
	if err != nil {
		return report, err
	}

	for _, id := range ids {
		report.Processed++

		changed, err := cs.resealChannel(ctx, id)
		if err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("channel %s: %s", id.String(), err.Error()))
			continue
		}
		if changed {
			report.Changed++
		}
	}

	return report, nil
}

func (cs *ChannelSecrets) resealChannel(ctx context.Context, id happydns.Identifier) (bool, error) {
	defer LockChannel(id)()

	ch, err := cs.store.GetChannel(id)
	if errors.Is(err, happydns.ErrNotificationChannelNotFound) {
		// Deleted since it was listed.
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// A type this version does not know fails: its secrets, if any,
	// cannot be told apart.
	changed, err := cs.registry.ResealChannelConfig(ctx, ch)
	if err != nil || !changed {
		return false, err
	}

	return true, cs.store.UpdateChannel(ch)
}
