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

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
)

// Carries only what at least one transport needs, so other transports cannot leak the user record.
type Recipient struct {
	// May be empty for transports that don't need it (webhook, UnifiedPush).
	Email string
}

// Senders receive only render-needed data — no user object, no server config — so adding a transport cannot leak privileged data.
type NotificationPayload struct {
	Recipient     Recipient
	CheckerID     string
	Target        happydns.CheckTarget
	DomainName    string
	ServiceDomain string // FQDN of the specific service (subdomain.domain), empty when the check is domain-scoped
	OldStatus     happydns.Status
	NewStatus     happydns.Status
	States        []happydns.CheckState
	Annotation    string
}

type ChannelConfig interface {
	Validate() error
}

// Senders own their config shape so adding a transport is a one-file change.
// Most implementations should embed TypedSender[C] via Adapt rather than implementing this directly.
type ChannelSender interface {
	Type() happydns.NotificationChannelType
	DecodeConfig(raw json.RawMessage) (ChannelConfig, error)
	// Reserved for the administration path: it may perform network lookups, so
	// it must never run while sending a notification.
	CheckConfig(ctx context.Context, cfg ChannelConfig) error
	Send(ctx context.Context, cfg ChannelConfig, payload *NotificationPayload) error
	SendTest(ctx context.Context, cfg ChannelConfig, user *happydns.User) error
	// Strip secrets to presence booleans before echoing config back to clients.
	RedactConfig(raw json.RawMessage) (json.RawMessage, error)
	// Preserve stored secrets when client submits empty fields (client never sees them on read).
	MergeForUpdate(existing, incoming json.RawMessage) (json.RawMessage, error)
	// Refuse a config sent by a client that holds a sealed value.
	CheckIncomingConfig(raw json.RawMessage) error
	// Seal the secrets of a config before it is stored.
	SealConfig(ctx context.Context, secrets *secret.Manager, sc secret.SecretContext, raw json.RawMessage) (json.RawMessage, error)
	// Decode a stored config with its secrets opened, for sending only.
	OpenConfig(ctx context.Context, secrets *secret.Manager, sc secret.SecretContext, raw json.RawMessage) (ChannelConfig, error)
	// Add how the secrets of a stored config are stored to counts.
	InspectConfig(ctx context.Context, secrets *secret.Manager, sc secret.SecretContext, raw json.RawMessage, counts *secret.Counts) error
	// Store the secrets of a stored config the way the current policy stores
	// new ones; the config is nil when nothing changed.
	ResealConfig(ctx context.Context, secrets *secret.Manager, sc secret.SecretContext, raw json.RawMessage) (json.RawMessage, error)
}

// Optional capability: senders with secret fields opt in by implementing this on their TypedSender.
type ConfigRedactor[C ChannelConfig] interface {
	RedactConfig(cfg C) C
}

type ConfigMerger[C ChannelConfig] interface {
	MergeForUpdate(existing, incoming C) C
}

// Destination is a URL a sender will dial, paired with the label naming the
// field it came from so a refusal can point the user at what they filled in.
type Destination struct {
	// Subject of the refusal message, e.g. "The webhook URL".
	Label string
	URL   string
}

// Strongly-typed contract; Adapt wraps it as ChannelSender, providing JSON
// decode, validation, address-policy checks, type-asserted dispatch and
// SendTest.
type TypedSender[C ChannelConfig] interface {
	Type() happydns.NotificationChannelType

	// Destinations lists every URL taken from the config that this sender will
	// dial. The adapter holds each one against the URL shape rules and, on the
	// administration path, against the instance address policy, neither of
	// which ChannelConfig.Validate can do: it is a value method with nothing
	// but the config in scope.
	//
	// Returning nil belongs to transports that dial nothing the user chose,
	// such as email. It is not an opt-out: a sender that omits a URL it later
	// dials sends happyDomain to an address its administrator never allowed.
	Destinations(cfg C) []Destination

	Send(ctx context.Context, cfg C, payload *NotificationPayload) error
}

// guard carries the instance address policy; it must not be nil unless the
// sender returns no destination at all.
func Adapt[C ChannelConfig](s TypedSender[C], guard *netguard.Guard) ChannelSender {
	return &typedAdapter[C]{inner: s, guard: guard}
}

type typedAdapter[C ChannelConfig] struct {
	inner TypedSender[C]
	guard *netguard.Guard
}

func (a *typedAdapter[C]) Type() happydns.NotificationChannelType { return a.inner.Type() }

func (a *typedAdapter[C]) DecodeConfig(raw json.RawMessage) (ChannelConfig, error) {
	c, err := a.decode(raw)
	if err != nil {
		return nil, err
	}
	if err := a.validate(c); err != nil {
		return nil, err
	}
	return c, nil
}

// decode decodes raw, without checking it.
func (a *typedAdapter[C]) decode(raw json.RawMessage) (C, error) {
	var c C
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &c); err != nil {
			return c, fmt.Errorf("decoding %s config: %w", a.inner.Type(), err)
		}
	}
	return c, nil
}

// validate checks c and the shape of its destinations. A destination held in
// a secret must be opened first.
func (a *typedAdapter[C]) validate(c C) error {
	if err := c.Validate(); err != nil {
		return err
	}
	for _, d := range a.inner.Destinations(c) {
		if _, err := netguard.ValidateURLShape(d.URL); err != nil {
			return fmt.Errorf("%s: %w", d.Label, err)
		}
	}
	return nil
}

// CheckConfig resolves the destinations, so it only runs when a user submits a
// channel. The send path relies on the dialer guard instead, otherwise a
// resolver hiccup would drop notifications for an already accepted channel.
func (a *typedAdapter[C]) CheckConfig(ctx context.Context, cfg ChannelConfig) error {
	typed, ok := cfg.(C)
	if !ok {
		return fmt.Errorf("%s sender: unexpected config type %T", a.inner.Type(), cfg)
	}
	dests := a.inner.Destinations(typed)
	if len(dests) > 0 && a.guard == nil {
		// Fail closed: a sender that dials cannot be registered without a policy.
		return fmt.Errorf("%s sender: registered without an outbound guard", a.inner.Type())
	}
	for _, d := range dests {
		if _, err := a.guard.ValidateURL(ctx, d.URL); err != nil {
			return errors.New(a.guard.Refusal(d.Label))
		}
	}
	return nil
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
	c, err := a.decode(raw)
	if err != nil {
		return nil, err
	}
	c = redactor.RedactConfig(c)
	return json.Marshal(c)
}

func (a *typedAdapter[C]) MergeForUpdate(existing, incoming json.RawMessage) (json.RawMessage, error) {
	merger, ok := a.inner.(ConfigMerger[C])
	if !ok {
		return incoming, nil
	}
	var ec C
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &ec); err != nil {
			return nil, fmt.Errorf("decoding existing %s config: %w", a.inner.Type(), err)
		}
	}
	ic, err := a.decode(incoming)
	if err != nil {
		return nil, err
	}
	merged := merger.MergeForUpdate(ec, ic)
	// Encoded as a request body: a new secret is still in clear, and is
	// sealed once the channel is accepted.
	return secret.MarshalIncoming(&merged)
}

func (a *typedAdapter[C]) CheckIncomingConfig(raw json.RawMessage) error {
	c, err := a.decode(raw)
	if err != nil {
		return err
	}
	return secret.CheckIncoming(&c)
}

func (a *typedAdapter[C]) SealConfig(ctx context.Context, secrets *secret.Manager, sc secret.SecretContext, raw json.RawMessage) (json.RawMessage, error) {
	c, err := a.decode(raw)
	if err != nil {
		return nil, err
	}
	if err := secrets.SealObject(ctx, sc, &c); err != nil {
		return nil, err
	}
	return json.Marshal(&c)
}

func (a *typedAdapter[C]) InspectConfig(ctx context.Context, secrets *secret.Manager, sc secret.SecretContext, raw json.RawMessage, counts *secret.Counts) error {
	c, err := a.decode(raw)
	if err != nil {
		return err
	}
	return secrets.Inspect(ctx, sc, &c, counts)
}

func (a *typedAdapter[C]) ResealConfig(ctx context.Context, secrets *secret.Manager, sc secret.SecretContext, raw json.RawMessage) (json.RawMessage, error) {
	c, err := a.decode(raw)
	if err != nil {
		return nil, err
	}
	changed, err := secrets.ResealObject(ctx, sc, &c)
	if err != nil || !changed {
		return nil, err
	}
	return json.Marshal(&c)
}

func (a *typedAdapter[C]) OpenConfig(ctx context.Context, secrets *secret.Manager, sc secret.SecretContext, raw json.RawMessage) (ChannelConfig, error) {
	c, err := a.decode(raw)
	if err != nil {
		return nil, err
	}
	// Opened before being checked: a destination may be a secret.
	if err := secrets.OpenObject(ctx, sc, &c); err != nil {
		return nil, err
	}
	if err := a.validate(c); err != nil {
		return nil, err
	}
	return c, nil
}

// Senders self-register at startup; adding a transport requires no changes here.
type Registry struct {
	senders map[happydns.NotificationChannelType]ChannelSender
	secrets *secret.Manager
}

// secrets seals the channel secrets before they are stored, and opens them
// right before sending.
func NewRegistry(secrets *secret.Manager) *Registry {
	return &Registry{
		senders: make(map[happydns.NotificationChannelType]ChannelSender),
		secrets: secrets,
	}
}

// SecretObjectType names notification channels in the context their secrets
// are bound to. Channels are used in the background, when a scheduled check
// completes: their secrets must stay in safes that open without the user.
const SecretObjectType = "notification-channel"

// ChannelSecretContext returns the context the secrets of ch are bound to.
func ChannelSecretContext(ch *happydns.NotificationChannel) secret.SecretContext {
	return secret.SecretContext{
		Owner:      ch.UserId,
		ObjectType: SecretObjectType,
		ObjectId:   ch.Id.String(),
	}
}

// CheckIncomingChannel refuses a channel sent by a client whose config holds
// a sealed value.
func (r *Registry) CheckIncomingChannel(ch *happydns.NotificationChannel) error {
	s, ok := r.Get(ch.Type)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownChannelType, ch.Type)
	}
	return s.CheckIncomingConfig(ch.Config)
}

// SealChannelConfig seals the secrets of the config of ch, in place, before
// it is stored. ch needs its identifier and owner.
func (r *Registry) SealChannelConfig(ctx context.Context, ch *happydns.NotificationChannel) error {
	s, ok := r.Get(ch.Type)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownChannelType, ch.Type)
	}
	sealed, err := s.SealConfig(ctx, r.secrets, ChannelSecretContext(ch), ch.Config)
	if err != nil {
		return err
	}
	ch.Config = sealed
	return nil
}

// InspectChannelConfig adds how the secrets of ch are stored to counts.
func (r *Registry) InspectChannelConfig(ctx context.Context, ch *happydns.NotificationChannel, counts *secret.Counts) error {
	s, ok := r.Get(ch.Type)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownChannelType, ch.Type)
	}
	return s.InspectConfig(ctx, r.secrets, ChannelSecretContext(ch), ch.Config, counts)
}

// ResealChannelConfig stores the secrets of ch the way the current policy
// stores new ones, in place, and reports whether ch changed.
func (r *Registry) ResealChannelConfig(ctx context.Context, ch *happydns.NotificationChannel) (bool, error) {
	s, ok := r.Get(ch.Type)
	if !ok {
		return false, fmt.Errorf("%w: %q", ErrUnknownChannelType, ch.Type)
	}
	resealed, err := s.ResealConfig(ctx, r.secrets, ChannelSecretContext(ch), ch.Config)
	if err != nil || resealed == nil {
		return false, err
	}
	ch.Config = resealed
	return true, nil
}

// OpenChannelConfig decodes the config of ch with its secrets opened. Only
// the send path uses it.
func (r *Registry) OpenChannelConfig(ctx context.Context, ch *happydns.NotificationChannel) (ChannelConfig, error) {
	s, ok := r.Get(ch.Type)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownChannelType, ch.Type)
	}
	return s.OpenConfig(ctx, r.secrets, ChannelSecretContext(ch), ch.Config)
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

// AcceptChannelConfig validates a channel a user is submitting: it decodes the
// config, opens what it carries forward sealed, then checks its destination
// against runtime policy. Only use it on the administration path; the send
// path must stick to OpenChannelConfig, as the destination check can hit the
// network. ch needs its identifier and owner.
//
// The config returned holds the secrets in clear: never store it.
func (r *Registry) AcceptChannelConfig(ctx context.Context, ch *happydns.NotificationChannel) (ChannelConfig, error) {
	s, ok := r.Get(ch.Type)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownChannelType, ch.Type)
	}
	// A destination carried forward sealed is checked as it is in clear.
	cfg, err := s.OpenConfig(ctx, r.secrets, ChannelSecretContext(ch), ch.Config)
	if err != nil {
		return nil, err
	}
	if err := s.CheckConfig(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
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
//
// It refuses an update changing the type of the channel with
// ErrChannelTypeChanged: each type reads its own config, and the stored one,
// secrets included, would be merged into a type that does not read it.
func (r *Registry) MergeChannelForUpdate(existing, incoming *happydns.NotificationChannel) (json.RawMessage, error) {
	if incoming.Type != existing.Type {
		return nil, fmt.Errorf("%w: from %q to %q", ErrChannelTypeChanged, existing.Type, incoming.Type)
	}

	s, ok := r.Get(incoming.Type)
	if !ok {
		return incoming.Config, nil
	}
	return s.MergeForUpdate(existing.Config, incoming.Config)
}

var ErrUnknownChannelType = errors.New("unknown channel type")

// ErrChannelTypeChanged is returned by MergeChannelForUpdate when the update
// changes the type of the channel.
var ErrChannelTypeChanged = errors.New("the type of a notification channel cannot change")
