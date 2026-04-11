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

	notifPkg "git.happydns.org/happyDomain/internal/notification"
	"git.happydns.org/happyDomain/model"
)

// Synchronous and bypasses preferences/state/quiet-hours — user explicitly verifying one channel.
type Tester struct {
	registry *notifPkg.Registry
}

func NewTester(registry *notifPkg.Registry) *Tester {
	return &Tester{registry: registry}
}

func (t *Tester) Send(ch *happydns.NotificationChannel, user *happydns.User) error {
	sender, ok := t.registry.Get(ch.Type)
	if !ok {
		return notifPkg.ErrUnknownChannelType
	}
	cfg, err := sender.DecodeConfig(ch.Config)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	return sender.SendTest(ctx, cfg, user)
}
