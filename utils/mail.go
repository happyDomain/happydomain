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
	"bytes"
	"flag"
	"io"
	"net/mail"
	"text/template"

	"git.happydns.org/happyDomain/ui"

	gomail "github.com/go-mail/mail"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// mailAddress defines an interface that handle mail.Address configuration
// throught custom flag.
type mailAddress struct {
	*mail.Address
}

func (i *mailAddress) String() string {
	if i.Address == nil {
		return ""
	}
	return i.Address.String()
}

func (i *mailAddress) Set(value string) error {
	v, err := mail.ParseAddress(value)
	if err != nil {
		return err
	}
	*i.Address = *v
	return nil
}

type sendMethod interface {
	PrepareAndSend(...*gomail.Message) error
}

var (
	// MailFrom holds the content of the From field for all e-mails that
	// will be send.
	MailFrom = mail.Address{Name: "happyDomain", Address: "happydomain@localhost"}

	// SendMethod is a pointer to the current global method used to send
	// e-mails.
	SendMethod sendMethod = &SystemSendmail{}
)

func init() {
	flag.Var(&mailAddress{&MailFrom}, "mail-from", "Define the sender name and address for all e-mail sent")
}

// SendMail takes a content writen in Markdown to send it to the given user. It
// uses Markdown to create a HTML version of the message and leave the Markdown
// format in the text version. To perform sending, it relies on the SendMethod
// global variable.
func SendMail(to *mail.Address, subject, content string) (err error) {
	m := gomail.NewMessage()
	m.SetHeader("From", MailFrom.String())
	m.SetHeader("To", to.String())
	m.SetHeader("Subject", subject)

	toName := to.Name
	if len(toName) == 0 {
		toName = to.Address
	}

	tplData := map[string]string{
		"Lang":        "en",
		"To":          toName,
		"ToAddress":   to.Address,
		"Subject":     subject,
		"From":        MailFrom.Name,
		"FromAddress": MailFrom.Address,
		"Content":     content,
	}

	if t, err := template.New("mailText").Parse(mailTXTTpl); err != nil {
		return err
	} else {
		m.SetBodyWriter("text/plain", func(w io.Writer) error {
			return t.Execute(w, tplData)
		})
	}

	// Convert text from Markdown to HTML
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	var buf bytes.Buffer
	if err = md.Convert([]byte(content), &buf); err != nil {
		return
	}

	if data, err := ui.GetEmbedFS().Open("dist/img/happydomain.png"); err == nil {
		m.EmbedReader("happydomain.png", data)
	}

	if t, err := template.New("mailHTML").Parse(mailHTMLTpl); err != nil {
		return err
	} else {
		m.AddAlternativeWriter("text/html", func(w io.Writer) error {
			tplData["Content"] = buf.String()
			return t.Execute(w, tplData)
		})
	}

	if err = SendMethod.PrepareAndSend(m); err != nil {
		return
	}

	return
}
