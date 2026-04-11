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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"git.happydns.org/happyDomain/model"
)

// UnifiedPushSender sends notifications via the UnifiedPush protocol.
type UnifiedPushSender struct {
	client *http.Client
}

// NewUnifiedPushSender creates a new UnifiedPushSender.
func NewUnifiedPushSender() *UnifiedPushSender {
	return &UnifiedPushSender{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *UnifiedPushSender) Send(channel *happydns.NotificationChannel, payload *NotificationPayload) error {
	if channel.Config.UnifiedPushEndpoint == "" {
		return fmt.Errorf("no UnifiedPush endpoint configured for channel %s", channel.Id)
	}

	msg := WebhookPayload{
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
		msg.DashboardURL = payload.BaseURL
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling UnifiedPush payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, channel.Config.UnifiedPushEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating UnifiedPush request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending UnifiedPush notification: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("UnifiedPush endpoint returned status %d", resp.StatusCode)
	}

	return nil
}
