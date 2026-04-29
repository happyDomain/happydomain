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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"git.happydns.org/happyDomain/model"
)

// Reserved by the HTTP client or used to spoof outbound identity (smuggling/host-routing risk).
var disallowedWebhookHeaders = map[string]struct{}{
	"host":              {},
	"content-length":    {},
	"content-encoding":  {},
	"transfer-encoding": {},
	"connection":        {},
	"upgrade":           {},
	"te":                {},
	"trailer":           {},
}

func validateHeader(k, v string) error {
	if k == "" {
		return errors.New("empty header name")
	}
	if strings.ContainsAny(k, "\r\n") || strings.ContainsAny(v, "\r\n") {
		return fmt.Errorf("header %q contains CR/LF", k)
	}
	if _, blocked := disallowedWebhookHeaders[strings.ToLower(k)]; blocked {
		return fmt.Errorf("header %q is not allowed", k)
	}
	return nil
}

const ChannelTypeWebhook happydns.NotificationChannelType = "webhook"

type WebhookConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	// HMAC-SHA256 signing key.
	Secret string `json:"secret,omitempty"`
	// Set only by RedactConfig — never stored or accepted on input.
	HasSecret bool `json:"hasSecret,omitempty"`
}

func (c WebhookConfig) Validate() error {
	if c.URL == "" {
		return errors.New("webhook URL is required")
	}
	if _, err := validateOutboundURL(c.URL); err != nil {
		return fmt.Errorf("webhook URL: %w", err)
	}
	for k, v := range c.Headers {
		if err := validateHeader(k, v); err != nil {
			return fmt.Errorf("webhook header: %w", err)
		}
	}
	return nil
}

// dashboardURL is captured here — server identity, not per-notification data.
type WebhookSender struct {
	client       *http.Client
	dashboardURL string
}

func NewWebhookSender(dashboardURL string) *WebhookSender {
	return &WebhookSender{
		client:       newSafeHTTPClient(10 * time.Second),
		dashboardURL: dashboardURL,
	}
}

func (s *WebhookSender) Type() happydns.NotificationChannelType { return ChannelTypeWebhook }

func (s *WebhookSender) RedactConfig(cfg WebhookConfig) WebhookConfig {
	cfg.HasSecret = cfg.Secret != ""
	cfg.Secret = ""
	return cfg
}

// Preserve stored secret on empty submit; client never receives it back, so absence means "no change".
func (s *WebhookSender) MergeForUpdate(existing, incoming WebhookConfig) WebhookConfig {
	if incoming.Secret == "" {
		incoming.Secret = existing.Secret
	}
	incoming.HasSecret = false
	return incoming
}

func (s *WebhookSender) Send(ctx context.Context, c WebhookConfig, payload *NotificationPayload) error {
	return postJSON(ctx, s.client, c.URL, buildHTTPPayload(payload, s.dashboardURL), func(req *http.Request, body []byte) {
		req.Header.Set("User-Agent", "happyDomain-Notification/1.0")
		for k, v := range c.Headers {
			// Defense in depth: catches stored channels that pre-date Validate().
			if err := validateHeader(k, v); err != nil {
				continue
			}
			req.Header.Set(k, v)
		}
		if c.Secret != "" {
			mac := hmac.New(sha256.New, []byte(c.Secret))
			mac.Write(body)
			req.Header.Set("X-Happydomain-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		}
	})
}
