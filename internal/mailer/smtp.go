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

package mailer

import (
	"crypto/tls"

	gomail "github.com/wneessen/go-mail"
)

// SMTPSendmail uses a SMTP server to send message
type SMTPMailer struct {
	Client *gomail.Client
	Host   string
}

func NewSMTPMailer(host string, port uint, username, password string) (*SMTPMailer, error) {
	opts := []gomail.Option{
		gomail.WithPort(int(port)),
	}

	if port == 465 {
		opts = append(opts, gomail.WithSSL())
	}

	if username != "" {
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
			gomail.WithUsername(username),
			gomail.WithPassword(password),
		)
	}

	client, err := gomail.NewClient(host, opts...)
	if err != nil {
		return nil, err
	}

	return &SMTPMailer{Client: client, Host: host}, nil
}

func (t *SMTPMailer) WithTLSNoVerify() error {
	return t.Client.SetTLSConfig(&tls.Config{
		ServerName:         t.Host,
		InsecureSkipVerify: true,
	})
}

// PrepareAndSend sends an e-mail to the given recipients using configured SMTP host.
func (t *SMTPMailer) PrepareAndSend(m ...*gomail.Msg) (err error) {
	err = t.Client.DialAndSend(m...)

	return
}
