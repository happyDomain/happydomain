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
	"fmt"
	"net/mail"
	"strings"

	"git.happydns.org/happyDomain/model"
)

// EmailSender sends notifications via email using the existing Mailer.
type EmailSender struct {
	mailer happydns.Mailer
}

// NewEmailSender creates a new EmailSender.
func NewEmailSender(mailer happydns.Mailer) *EmailSender {
	return &EmailSender{mailer: mailer}
}

func (s *EmailSender) Send(channel *happydns.NotificationChannel, payload *NotificationPayload) error {
	addr := channel.Config.EmailAddress
	if addr == "" && payload.User != nil {
		addr = payload.User.Email
	}
	if addr == "" {
		return fmt.Errorf("no email address configured for channel %s", channel.Id)
	}

	to := &mail.Address{Address: addr}
	if payload.User != nil {
		to.Name = payload.User.Email
	}

	subject := fmt.Sprintf("[happyDomain] %s: %s", payload.DomainName, payload.NewStatus)

	var body strings.Builder
	fmt.Fprintf(&body, "## Status Change: %s -> %s\n\n", payload.OldStatus, payload.NewStatus)
	fmt.Fprintf(&body, "**Domain:** %s\n\n", payload.DomainName)
	fmt.Fprintf(&body, "**Checker:** %s\n\n", payload.CheckerID)

	if len(payload.States) > 0 {
		body.WriteString("### Rule Results\n\n")
		for _, state := range payload.States {
			fmt.Fprintf(&body, "- **%s** (%s): %s\n", state.Code, state.Status, state.Message)
		}
		body.WriteString("\n")
	}

	if payload.Annotation != "" {
		fmt.Fprintf(&body, "**Note:** %s\n\n", payload.Annotation)
	}

	if payload.BaseURL != "" {
		fmt.Fprintf(&body, "[View in happyDomain](%s)\n", payload.BaseURL)
	}

	return s.mailer.SendMail(to, subject, body.String())
}
