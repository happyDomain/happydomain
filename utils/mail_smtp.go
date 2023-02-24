// Copyright or © or Copr. happyDNS (2021)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

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
