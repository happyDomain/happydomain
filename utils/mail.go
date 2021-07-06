// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
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
	"bytes"
	"flag"
	"io"
	"io/fs"
	"net/mail"
	"text/template"

	"git.happydns.org/happydns/ui"

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
	MailFrom = mail.Address{Name: "happyDNS", Address: "happydns@localhost"}

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

	if data, err := fs.ReadFile(ui.GetEmbedFS(), "dist/img/happydns.png"); err != nil {
		m.EmbedReader("happydns.png", bytes.NewReader(data))
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
