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
	"fmt"

	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/model"
)

// ChannelService creates the notification channels users submit.
type ChannelService struct {
	store    NotificationChannelStorage
	registry *notifPkg.Registry
}

func NewChannelService(store NotificationChannelStorage, registry *notifPkg.Registry) *ChannelService {
	return &ChannelService{store: store, registry: registry}
}

// CreateChannel stores ch, submitted by user, as a new channel of theirs.
//
// Every creation goes through here, because the storage keeps the identifier
// a channel carries: ch gets a fresh one and user as owner, whatever it held,
// so that no client chooses where its channel is stored nor whose it is.
//
// What is wrong with ch itself comes back as a happydns.ValidationError; any
// other error is not the user's to see.
func (s *ChannelService) CreateChannel(ctx context.Context, user *happydns.User, ch *happydns.NotificationChannel) error {
	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		return fmt.Errorf("unable to generate the channel identifier: %w", err)
	}
	ch.Id = id
	ch.UserId = user.Id

	if _, err := s.registry.AcceptChannelConfig(ctx, ch); err != nil {
		return happydns.ValidationError{Msg: err.Error()}
	}

	if err := s.store.CreateChannel(ch); err != nil {
		return fmt.Errorf("unable to store the channel: %w", err)
	}

	return nil
}
