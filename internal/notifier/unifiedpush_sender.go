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
	"errors"
	"net/http"
	"time"

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

const ChannelTypeUnifiedPush happydns.NotificationChannelType = "unifiedpush"

type UnifiedPushConfig struct {
	// Endpoint is a capability URL, enough to push to the user's device:
	// sealed, and never sent back to the client.
	Endpoint happydns.Secret `json:"endpoint"`
}

func (c UnifiedPushConfig) Validate() error {
	if c.Endpoint.IsEmpty() {
		return errors.New("UnifiedPush endpoint is required")
	}
	return nil
}

// dashboardURL is captured here — server identity, not per-notification data.
type UnifiedPushSender struct {
	client       *http.Client
	dashboardURL string
}

func NewUnifiedPushSender(dashboardURL string, guard *netguard.Guard) *UnifiedPushSender {
	return &UnifiedPushSender{
		client:       guard.HTTPClient(10 * time.Second),
		dashboardURL: dashboardURL,
	}
}

func (s *UnifiedPushSender) Type() happydns.NotificationChannelType { return ChannelTypeUnifiedPush }

func (s *UnifiedPushSender) Destinations(c UnifiedPushConfig) []Destination {
	return []Destination{{Label: "The UnifiedPush endpoint", URL: c.Endpoint.Reveal()}}
}

func (s *UnifiedPushSender) RedactConfig(cfg UnifiedPushConfig) UnifiedPushConfig {
	cfg.Endpoint.Redact()
	return cfg
}

// The placeholder the client echoes back keeps the stored endpoint.
func (s *UnifiedPushSender) MergeForUpdate(existing, incoming UnifiedPushConfig) UnifiedPushConfig {
	if incoming.Endpoint.IsRedacted() {
		incoming.Endpoint = existing.Endpoint
	}
	return incoming
}

func (s *UnifiedPushSender) Send(ctx context.Context, c UnifiedPushConfig, payload *NotificationPayload) error {
	endpoint := c.Endpoint.Reveal()
	if endpoint == "" {
		return errors.New("UnifiedPush endpoint not opened")
	}
	return postJSON(ctx, s.client, endpoint, buildHTTPPayload(payload, s.dashboardURL), nil)
}
