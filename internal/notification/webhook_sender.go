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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"git.happydns.org/happyDomain/model"
)

// WebhookPayload is the JSON body sent to webhook endpoints.
type WebhookPayload struct {
	Event        string              `json:"event"`
	Checker      string              `json:"checker"`
	Domain       string              `json:"domain"`
	Target       happydns.CheckTarget `json:"target"`
	OldStatus    happydns.Status     `json:"oldStatus"`
	NewStatus    happydns.Status     `json:"newStatus"`
	States       []happydns.CheckState `json:"states,omitempty"`
	Timestamp    time.Time           `json:"timestamp"`
	DashboardURL string              `json:"dashboardUrl,omitempty"`
}

// WebhookSender sends notifications via HTTP POST to a configured URL.
type WebhookSender struct {
	client *http.Client
}

// NewWebhookSender creates a new WebhookSender.
func NewWebhookSender() *WebhookSender {
	return &WebhookSender{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *WebhookSender) Send(channel *happydns.NotificationChannel, payload *NotificationPayload) error {
	if channel.Config.WebhookURL == "" {
		return fmt.Errorf("no webhook URL configured for channel %s", channel.Id)
	}

	whPayload := WebhookPayload{
		Event:     "status_change",
		Checker:   payload.CheckerID,
		Domain:    payload.DomainName,
		Target:    payload.Target,
		OldStatus: payload.OldStatus,
		NewStatus: payload.NewStatus,
		States:    payload.States,
		Timestamp: time.Now(),
	}
	if payload.BaseURL != "" {
		whPayload.DashboardURL = payload.BaseURL
	}

	body, err := json.Marshal(whPayload)
	if err != nil {
		return fmt.Errorf("marshaling webhook payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, channel.Config.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "happyDomain-Notification/1.0")

	for k, v := range channel.Config.WebhookHeaders {
		req.Header.Set(k, v)
	}

	if channel.Config.WebhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(channel.Config.WebhookSecret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Happydomain-Signature", "sha256="+sig)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending webhook: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
