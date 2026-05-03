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
	"net/mail"
	"strings"

	"git.happydns.org/happyDomain/model"
)

const ChannelTypeEmail happydns.NotificationChannelType = "email"

type EmailConfig struct {
	// Empty means fall back to the user's account email.
	Address string `json:"address,omitempty"`
}

func (c EmailConfig) Validate() error {
	if c.Address == "" {
		return nil
	}
	if _, err := mail.ParseAddress(c.Address); err != nil {
		return fmt.Errorf("invalid email address: %w", err)
	}
	return nil
}

// baseURL is captured here — server identity, not per-notification data.
type EmailSender struct {
	mailer  happydns.Mailer
	baseURL string
}

func NewEmailSender(mailer happydns.Mailer, baseURL string) *EmailSender {
	return &EmailSender{mailer: mailer, baseURL: baseURL}
}

func (s *EmailSender) Type() happydns.NotificationChannelType { return ChannelTypeEmail }

func (s *EmailSender) Send(_ context.Context, c EmailConfig, payload *NotificationPayload) error {
	addr := c.Address
	if addr == "" {
		addr = payload.Recipient.Email
	}
	if addr == "" {
		return errors.New("no email address available")
	}

	to := &mail.Address{Address: addr}

	// Strip CR/LF to prevent RFC 5322 header injection.
	safeDomain := stripCRLF(payload.DomainName)
	subject := fmt.Sprintf("[happyDomain] %s: %s", safeDomain, payload.NewStatus)

	// Wrap third-party-sourced fields as Markdown code spans to neutralize injected link syntax in DKIM-signed mail; Annotation is user-authored, no boundary.
	var body strings.Builder
	fmt.Fprintf(&body, "## Status Change: %s -> %s\n\n", payload.OldStatus, payload.NewStatus)
	fmt.Fprintf(&body, "**Domain:** %s\n\n", mdLiteral(payload.DomainName))
	fmt.Fprintf(&body, "**Checker:** %s\n\n", mdLiteral(payload.CheckerID))

	if len(payload.States) > 0 {
		body.WriteString("### Rule Results\n\n")
		for _, state := range payload.States {
			fmt.Fprintf(&body, "- %s (%s): %s\n", mdLiteral(state.Code), state.Status, mdLiteral(state.Message))
		}
		body.WriteString("\n")
	}

	if payload.Annotation != "" {
		fmt.Fprintf(&body, "**Note:** %s\n\n", payload.Annotation)
	}

	if s.baseURL != "" {
		fmt.Fprintf(&body, "[View in happyDomain](%s)\n", s.baseURL)
	}

	return s.mailer.SendMail(to, subject, body.String())
}

func stripCRLF(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

// Wraps s as a code span; backticks become apostrophes to avoid fence accounting.
func mdLiteral(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "'") + "`"
}
