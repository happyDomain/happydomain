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
	"bytes"
	"context"
	"errors"
	"fmt"

	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
)

// ChannelService creates and updates the notification channels users submit.
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

	if err := s.registry.CheckIncomingChannel(ch); err != nil {
		return happydns.ValidationError{Msg: err.Error()}
	}

	if _, err := s.registry.AcceptChannelConfig(ctx, ch); err != nil {
		return happydns.ValidationError{Msg: err.Error()}
	}

	// After the identifier and owner are set: the secrets are bound to them.
	if err := s.registry.SealChannelConfig(ctx, ch); errors.Is(err, secret.ErrRedactedSecret) {
		// Nothing is stored yet for the placeholder to stand for: the
		// client sent it.
		return happydns.ValidationError{Msg: "the secret placeholder cannot be used to create a channel: enter the secret, or leave it empty"}
	} else if err != nil {
		return fmt.Errorf("unable to seal the channel secrets: %w", err)
	}

	if err := s.store.CreateChannel(ch); err != nil {
		return fmt.Errorf("unable to store the channel: %w", err)
	}

	return nil
}

// UpdateChannel stores the channel id of user as apply changes it from what
// is stored, and returns it. apply gets a copy of the stored channel, so that
// what it leaves out keeps its stored value; the identifier and owner are
// kept whatever it does.
//
// The channel is written only if it is still what apply was given: the
// stored secret carried forward is never written over what another writer,
// such as a reseal, stored in between. Such an update fails with a
// happydns.ConflictError; retrying reads the channel again.
//
// This only guards against a write landing while the update is handled. A
// client submitting a form read before someone else's change is not detected:
// the update is applied to what is stored when writing.
//
// What is wrong with the update comes back as a happydns.ValidationError; a
// channel that is not user's as happydns.ErrNotificationChannelNotFound.
func (s *ChannelService) UpdateChannel(ctx context.Context, user *happydns.User, id happydns.Identifier, apply func(*happydns.NotificationChannel) error) (*happydns.NotificationChannel, error) {
	// What updating the stored channel failed with, told apart from what the
	// storage fails with.
	var updateErr error
	var updated *happydns.NotificationChannel
	err := s.store.ReplaceChannel(id, func(existing *happydns.NotificationChannel) (*happydns.NotificationChannel, error) {
		updated, updateErr = s.updated(ctx, user, existing, apply)
		return updated, updateErr
	})

	switch {
	case updateErr != nil:
		return nil, updateErr
	case errors.Is(err, happydns.ErrChangedMeanwhile):
		return nil, happydns.ConflictError{
			Msg: "This channel was written by another operation at the same moment. Please try again.",
			Err: err,
		}
	case err != nil:
		return nil, err
	}

	return updated, nil
}

// updated returns existing, a stored channel, as apply changes it, checked,
// its stored secrets carried forward, and sealed.
func (s *ChannelService) updated(ctx context.Context, user *happydns.User, existing *happydns.NotificationChannel, apply func(*happydns.NotificationChannel) error) (*happydns.NotificationChannel, error) {
	if !existing.UserId.Equals(user.Id) {
		return nil, happydns.ErrNotificationChannelNotFound
	}

	// A clone, not a copy: decoding reuses the slices it finds, and the
	// merge below reads existing.
	ch := existing.Clone()
	if err := apply(ch); err != nil {
		return nil, happydns.ValidationError{Msg: err.Error()}
	}
	ch.Id = existing.Id
	ch.UserId = existing.UserId

	// Each type reads its own config: another type would inherit the stored
	// one, secrets included.
	if ch.Type != existing.Type {
		return nil, happydns.ValidationError{Msg: "the type of a channel cannot be changed, create a new channel instead"}
	}

	// A config left out is the stored one, sealed values included; only one
	// the client sent is checked.
	if !bytes.Equal(ch.Config, existing.Config) {
		if err := s.registry.CheckIncomingChannel(ch); err != nil {
			return nil, happydns.ValidationError{Msg: err.Error()}
		}
	}

	// Carry forward stored secrets, so that a GET then PUT round-trip does
	// not wipe them.
	merged, err := s.registry.MergeChannelForUpdate(existing, ch)
	if err != nil {
		return nil, happydns.ValidationError{Msg: err.Error()}
	}
	ch.Config = merged

	// Both open what is carried forward sealed.
	_, err = s.registry.AcceptChannelConfig(ctx, ch)
	if uerr := secret.UserError(err, "notification channel"); uerr != nil {
		return nil, uerr
	} else if err != nil {
		return nil, happydns.ValidationError{Msg: err.Error()}
	}

	err = s.registry.SealChannelConfig(ctx, ch)
	if uerr := secret.UserError(err, "notification channel"); uerr != nil {
		return nil, uerr
	} else if errors.Is(err, secret.ErrRedactedSecret) {
		// A placeholder with nothing stored under its name, such as a
		// renamed header.
		return nil, happydns.ValidationError{Msg: "a secret placeholder has no stored value behind it, such as a renamed header: enter the value again"}
	} else if err != nil {
		return nil, fmt.Errorf("unable to seal the channel secrets: %w", err)
	}

	return ch, nil
}
