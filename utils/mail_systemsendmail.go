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
	"io"
	"os"
	"os/exec"

	gomail "github.com/go-mail/mail"
)

// sendmail contains the path to the sendmail command
const sendmail = "/usr/sbin/sendmail"

// SystemSendmail uses the sendmail command to send message
type SystemSendmail struct{}

func (t *SystemSendmail) Send(from string, to []string, msg io.WriterTo) error {
	cmd := exec.Command(sendmail, "-t")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	pw, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	if _, err = msg.WriteTo(pw); err != nil {
		return err
	}

	if err = pw.Close(); err != nil {
		return err
	}

	if err = cmd.Wait(); err != nil {
		return err
	}

	return nil
}

// Send sends an e-mail to the given recipients using the sendmail command.
func (t *SystemSendmail) PrepareAndSend(m ...*gomail.Message) (err error) {
	err = gomail.Send(t, m...)
	return
}
