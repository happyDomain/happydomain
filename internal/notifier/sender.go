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

package notifier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

// Carries only what at least one transport needs, so other transports cannot leak the user record.
type Recipient struct {
	// May be empty for transports that don't need it (webhook, UnifiedPush).
	Email string
}

// Senders receive only render-needed data — no user object, no server config — so adding a transport cannot leak privileged data.
type NotificationPayload struct {
	Recipient  Recipient
	CheckerID  string
	Target     happydns.CheckTarget
	DomainName string
	OldStatus  happydns.Status
	NewStatus  happydns.Status
	States     []happydns.CheckState
	Annotation string
}

type ChannelConfig interface {
	Validate() error
}

// Senders own their config shape so adding a transport is a one-file change.
// Most implementations should embed TypedSender[C] via Adapt rather than implementing this directly.
type ChannelSender interface {
	Type() happydns.NotificationChannelType
	DecodeConfig(raw json.RawMessage) (ChannelConfig, error)
	Send(ctx context.Context, cfg ChannelConfig, payload *NotificationPayload) error
	SendTest(ctx context.Context, cfg ChannelConfig, user *happydns.User) error
	// Strip secrets to presence booleans before echoing config back to clients.
	RedactConfig(raw json.RawMessage) (json.RawMessage, error)
	// Preserve stored secrets when client submits empty fields (client never sees them on read).
	MergeForUpdate(existing, incoming json.RawMessage) (json.RawMessage, error)
}

// Optional capability: senders with secret fields opt in by implementing this on their TypedSender.
type ConfigRedactor[C ChannelConfig] interface {
	RedactConfig(cfg C) C
}

type ConfigMerger[C ChannelConfig] interface {
	MergeForUpdate(existing, incoming C) C
}

// Strongly-typed contract; Adapt wraps it as ChannelSender, providing JSON decode, validation, type-asserted dispatch, and SendTest.
type TypedSender[C ChannelConfig] interface {
	Type() happydns.NotificationChannelType
	Send(ctx context.Context, cfg C, payload *NotificationPayload) error
}

func Adapt[C ChannelConfig](s TypedSender[C]) ChannelSender {
	return &typedAdapter[C]{inner: s}
}

type typedAdapter[C ChannelConfig] struct {
	inner TypedSender[C]
}

func (a *typedAdapter[C]) Type() happydns.NotificationChannelType { return a.inner.Type() }

func (a *typedAdapter[C]) DecodeConfig(raw json.RawMessage) (ChannelConfig, error) {
	var c C
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("decoding %s config: %w", a.inner.Type(), err)
		}
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (a *typedAdapter[C]) Send(ctx context.Context, cfg ChannelConfig, payload *NotificationPayload) error {
	typed, ok := cfg.(C)
	if !ok {
		return fmt.Errorf("%s sender: unexpected config type %T", a.inner.Type(), cfg)
	}
	return a.inner.Send(ctx, typed, payload)
}

func (a *typedAdapter[C]) SendTest(ctx context.Context, cfg ChannelConfig, user *happydns.User) error {
	return a.Send(ctx, cfg, testPayload(Recipient{Email: user.Email}))
}

func (a *typedAdapter[C]) RedactConfig(raw json.RawMessage) (json.RawMessage, error) {
	redactor, ok := a.inner.(ConfigRedactor[C])
	if !ok {
		return raw, nil
	}
	var c C
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("decoding %s config: %w", a.inner.Type(), err)
		}
	}
	c = redactor.RedactConfig(c)
	return json.Marshal(c)
}

func (a *typedAdapter[C]) MergeForUpdate(existing, incoming json.RawMessage) (json.RawMessage, error) {
	merger, ok := a.inner.(ConfigMerger[C])
	if !ok {
		return incoming, nil
	}
	var ec, ic C
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &ec); err != nil {
			return nil, fmt.Errorf("decoding existing %s config: %w", a.inner.Type(), err)
		}
	}
	if len(incoming) > 0 {
		if err := json.Unmarshal(incoming, &ic); err != nil {
			return nil, fmt.Errorf("decoding %s config: %w", a.inner.Type(), err)
		}
	}
	return json.Marshal(merger.MergeForUpdate(ec, ic))
}

// Senders self-register at startup; adding a transport requires no changes here.
type Registry struct {
	senders map[happydns.NotificationChannelType]ChannelSender
}

func NewRegistry() *Registry {
	return &Registry{senders: make(map[happydns.NotificationChannelType]ChannelSender)}
}

// Panics on duplicate — programming error.
func (r *Registry) Register(s ChannelSender) {
	t := s.Type()
	if _, exists := r.senders[t]; exists {
		panic(fmt.Sprintf("notification: sender already registered for type %q", t))
	}
	r.senders[t] = s
}

func (r *Registry) Get(t happydns.NotificationChannelType) (ChannelSender, bool) {
	s, ok := r.senders[t]
	return s, ok
}

func (r *Registry) Types() []happydns.NotificationChannelType {
	out := make([]happydns.NotificationChannelType, 0, len(r.senders))
	for t := range r.senders {
		out = append(out, t)
	}
	return out
}

func (r *Registry) DecodeChannelConfig(ch *happydns.NotificationChannel) (ChannelConfig, error) {
	s, ok := r.Get(ch.Type)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownChannelType, ch.Type)
	}
	return s.DecodeConfig(ch.Config)
}

// Channels of unknown types are returned unchanged so administrators can still observe legacy data.
func (r *Registry) RedactChannel(ch *happydns.NotificationChannel) (*happydns.NotificationChannel, error) {
	if ch == nil {
		return nil, nil
	}
	s, ok := r.Get(ch.Type)
	if !ok {
		copy := *ch
		return &copy, nil
	}
	redacted, err := s.RedactConfig(ch.Config)
	if err != nil {
		return nil, err
	}
	copy := *ch
	copy.Config = redacted
	return &copy, nil
}

func (r *Registry) RedactChannels(chs []*happydns.NotificationChannel) ([]*happydns.NotificationChannel, error) {
	out := make([]*happydns.NotificationChannel, 0, len(chs))
	for _, ch := range chs {
		red, err := r.RedactChannel(ch)
		if err != nil {
			return nil, err
		}
		out = append(out, red)
	}
	return out, nil
}

// Caller should DecodeConfig the returned raw before persisting.
func (r *Registry) MergeChannelForUpdate(existing, incoming *happydns.NotificationChannel) (json.RawMessage, error) {
	s, ok := r.Get(incoming.Type)
	if !ok {
		return incoming.Config, nil
	}
	return s.MergeForUpdate(existing.Config, incoming.Config)
}

var ErrUnknownChannelType = errors.New("unknown channel type")
