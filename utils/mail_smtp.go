// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

package utils

import (
	"crypto/tls"
	"flag"
	"strconv"
	"time"

	gomail "github.com/go-mail/mail"
)

var (
	smtpSendmail             *SMTPSendmail = nil
	smtpSendmailTLSSNoVerify bool          = false
)

// SMTPSendmail uses a SMTP server to send message
type SMTPSendmail struct {
	Dialer *gomail.Dialer
}

// Send sends an e-mail to the given recipients using configured SMTP host.
func (t *SMTPSendmail) PrepareAndSend(m ...*gomail.Message) (err error) {
	if smtpSendmailTLSSNoVerify {
		SendMethod.(*SMTPSendmail).Dialer.TLSConfig = &tls.Config{
			ServerName:         SendMethod.(*SMTPSendmail).Dialer.Host,
			InsecureSkipVerify: true,
		}
	}

	err = t.Dialer.DialAndSend(m...)

	return
}

func changeSendMethodToSMTP() {
	if _, ok := SendMethod.(*SMTPSendmail); !ok {
		if smtpSendmail == nil {
			smtpSendmail = &SMTPSendmail{
				Dialer: &gomail.Dialer{
					Timeout:      10 * time.Second,
					RetryFailure: true,
				},
			}
		}
		SendMethod = smtpSendmail
	}
}

type smtpSendmailHostname struct{}

func (s *smtpSendmailHostname) Set(value string) (err error) {
	changeSendMethodToSMTP()
	SendMethod.(*SMTPSendmail).Dialer.Host = value
	return
}

func (s *smtpSendmailHostname) String() string {
	return "smtp.happydomain.org"
}

type smtpSendmailPort struct{}

func (s *smtpSendmailPort) Set(value string) (err error) {
	changeSendMethodToSMTP()
	SendMethod.(*SMTPSendmail).Dialer.Port, err = strconv.Atoi(value)
	if err != nil {
		return
	}

	SendMethod.(*SMTPSendmail).Dialer.SSL = SendMethod.(*SMTPSendmail).Dialer.Port == 465

	return
}

func (s *smtpSendmailPort) String() string {
	return "465"
}

type smtpSendmailUsername struct{}

func (s *smtpSendmailUsername) Set(value string) (err error) {
	changeSendMethodToSMTP()
	SendMethod.(*SMTPSendmail).Dialer.Username = value
	return
}

func (s *smtpSendmailUsername) String() string {
	return ""
}

type smtpSendmailPassword struct{}

func (s *smtpSendmailPassword) Set(value string) (err error) {
	changeSendMethodToSMTP()
	SendMethod.(*SMTPSendmail).Dialer.Password = value
	return
}

func (s *smtpSendmailPassword) String() string {
	return ""
}

func init() {
	flag.BoolVar(&smtpSendmailTLSSNoVerify, "mail-smtp-tls-no-verify", false, "Do not verify certificate validity on SMTP connection")
	flag.Var(&smtpSendmailHostname{}, "mail-smtp-host", "Use the given SMTP server as default way to send emails")
	flag.Var(&smtpSendmailPort{}, "mail-smtp-port", "Define the port to use to send e-mail through SMTP method")
	flag.Var(&smtpSendmailUsername{}, "mail-smtp-username", "If the SMTP server requires authentication, fill with the username to authenticate with")
	flag.Var(&smtpSendmailPassword{}, "mail-smtp-password", "Password associated with the given username for SMTP authentication")
}
