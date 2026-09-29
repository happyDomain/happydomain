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

	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
)

// ChannelSecrets lets the administrator follow and migrate how the secrets of
// notification channels are stored.
type ChannelSecrets struct {
	store    NotificationChannelStorage
	registry *notifPkg.Registry
}

func NewChannelSecrets(store NotificationChannelStorage, registry *notifPkg.Registry) *ChannelSecrets {
	return &ChannelSecrets{store: store, registry: registry}
}

// channelName names a channel in the reports.
func channelName(ch *happydns.NotificationChannel) string {
	return "channel " + ch.Id.String()
}

// InspectSecrets tells how the secrets of every channel are stored. A channel
// of a type no sender handles counts as undecodable, as it may hold sealed
// secrets nobody can look at.
func (cs *ChannelSecrets) InspectSecrets(ctx context.Context) (secret.Counts, error) {
	iter, err := cs.store.ListAllChannels()
	if err != nil {
		return secret.Counts{}, err
	}

	return secret.InspectAll(iter, channelName, func(ch *happydns.NotificationChannel, c *secret.Counts) error {
		return cs.registry.InspectChannelConfig(ctx, ch, c)
	})
}

// ResealSecrets stores the secrets of every channel the way the current policy
// stores new ones. A channel that fails is reported and skipped; run it again
// to resume.
func (cs *ChannelSecrets) ResealSecrets(ctx context.Context) (secret.ResealReport, error) {
	iter, err := cs.store.ListAllChannels()
	if err != nil {
		return secret.ResealReport{ObjectType: notifPkg.SecretObjectType}, err
	}

	return secret.ResealAll(notifPkg.SecretObjectType, iter, channelName, func(ch *happydns.NotificationChannel) (bool, error) {
		return cs.resealChannel(ctx, ch.Id)
	})
}

// resealChannel reseals the channel id as stored now. The write is
// conditional: whatever another writer did in between wins, and the channel
// is left for the next run.
func (cs *ChannelSecrets) resealChannel(ctx context.Context, id happydns.Identifier) (bool, error) {
	changed := false
	err := cs.store.ReplaceChannel(id, func(ch *happydns.NotificationChannel) (*happydns.NotificationChannel, error) {
		c, err := cs.registry.ResealChannelConfig(ctx, ch)
		if err != nil || !c {
			return nil, err
		}
		changed = true
		return ch, nil
	})
	if errors.Is(err, happydns.ErrNotificationChannelNotFound) {
		// Deleted since it was listed.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return changed, nil
}
