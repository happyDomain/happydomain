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
	"fmt"
	"net/http"
	"time"

	"git.happydns.org/happyDomain/model"
)

const ChannelTypeUnifiedPush happydns.NotificationChannelType = "unifiedpush"

type UnifiedPushConfig struct {
	Endpoint string `json:"endpoint"`
}

func (c UnifiedPushConfig) Validate() error {
	if c.Endpoint == "" {
		return errors.New("UnifiedPush endpoint is required")
	}
	if _, err := validateOutboundURL(c.Endpoint); err != nil {
		return fmt.Errorf("UnifiedPush endpoint: %w", err)
	}
	return nil
}

// dashboardURL is captured here — server identity, not per-notification data.
type UnifiedPushSender struct {
	client       *http.Client
	dashboardURL string
}

func NewUnifiedPushSender(dashboardURL string) *UnifiedPushSender {
	return &UnifiedPushSender{
		client:       newSafeHTTPClient(10 * time.Second),
		dashboardURL: dashboardURL,
	}
}

func (s *UnifiedPushSender) Type() happydns.NotificationChannelType { return ChannelTypeUnifiedPush }

func (s *UnifiedPushSender) Send(ctx context.Context, c UnifiedPushConfig, payload *NotificationPayload) error {
	return postJSON(ctx, s.client, c.Endpoint, buildHTTPPayload(payload, s.dashboardURL), nil)
}
