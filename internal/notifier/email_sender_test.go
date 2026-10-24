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
	"net/mail"
	"strings"
	"testing"
)

type recordingMailer struct {
	to      *mail.Address
	subject string
	content string
}

func (m *recordingMailer) SendMail(to *mail.Address, subject, content string) error {
	m.to, m.subject, m.content = to, subject, content
	return nil
}

func TestEmailSenderEvent(t *testing.T) {
	m := &recordingMailer{}
	s := NewEmailSender(m, "https://happydomain.example")

	err := s.Send(context.Background(), EmailConfig{}, &NotificationPayload{
		Recipient: Recipient{Email: "alice@example.com"},
		Event: &NotificationEvent{
			Kind:  "domain_watch",
			Title: "example.com is available\r\nBcc: mallory@example.com",
			Body:  "Register it now.",
			Path:  "/watchlist/",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if m.to.Address != "alice@example.com" {
		t.Errorf("to = %v", m.to)
	}
	if m.subject != "[happyDomain] example.com is availableBcc: mallory@example.com" {
		t.Errorf("subject = %q", m.subject)
	}
	if !strings.Contains(m.content, "Register it now.") || !strings.Contains(m.content, "(https://happydomain.example/watchlist/)") {
		t.Errorf("content = %q", m.content)
	}
	if strings.Contains(m.content, "Status Change") {
		t.Errorf("event email reports a status change: %q", m.content)
	}
}

func TestEmailSenderEventWithoutBaseURL(t *testing.T) {
	m := &recordingMailer{}
	s := NewEmailSender(m, "")

	err := s.Send(context.Background(), EmailConfig{}, &NotificationPayload{
		Recipient: Recipient{Email: "alice@example.com"},
		Event: &NotificationEvent{
			Kind:  "domain_watch",
			Title: "example.com is available",
			Body:  "Register it now.",
			Path:  "/watchlist/",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(m.content, "/watchlist/") {
		t.Errorf("content links to a page with no base URL to reach it: %q", m.content)
	}
}
