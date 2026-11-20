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

package app

import (
	"testing"

	"git.happydns.org/happyDomain/internal/mailer"
	happydns "git.happydns.org/happyDomain/model"
)

// With -no-mail, no mail leaves the instance: they are written to the
// console instead.
func TestNewMailerWithNoMailLogsToTheConsole(t *testing.T) {
	m, err := newMailer(&happydns.Options{NoMail: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.(*mailer.LogMailer); !ok {
		t.Errorf("newMailer() = %T, want *mailer.LogMailer", m)
	}
}

// The configuration refuses both together, but -no-mail must win even when an
// SMTP server reaches the options another way.
func TestNewMailerWithNoMailIgnoresTheSMTPServer(t *testing.T) {
	m, err := newMailer(&happydns.Options{NoMail: true, MailSMTPHost: "smtp.example.com", MailSMTPPort: 587})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.(*mailer.LogMailer); !ok {
		t.Errorf("newMailer() = %T, want *mailer.LogMailer", m)
	}
}

func TestNewMailerDefaultsToSendmail(t *testing.T) {
	m, err := newMailer(&happydns.Options{})
	if err != nil {
		t.Fatal(err)
	}
	mm, ok := m.(*mailer.Mailer)
	if !ok {
		t.Fatalf("newMailer() = %T, want *mailer.Mailer", m)
	}
	if _, ok := mm.SendMethod.(*mailer.SystemSendmail); !ok {
		t.Errorf("SendMethod = %T, want *mailer.SystemSendmail", mm.SendMethod)
	}
}
