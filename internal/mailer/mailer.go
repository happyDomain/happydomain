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
	"bytes"
	"fmt"
	"io"
	"maps"
	"net/mail"
	"regexp"
	"strings"
	"text/template"

	"git.happydns.org/happyDomain/web"

	gomail "github.com/wneessen/go-mail"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type sendMethod interface {
	PrepareAndSend(...*gomail.Msg) error
}

// templateWriteFunc adapts a text/template's Execute method to the
// func(io.Writer) (int64, error) signature expected by go-mail's body
// writers.
func templateWriteFunc(tpl *template.Template, data any) func(io.Writer) (int64, error) {
	return func(w io.Writer) (int64, error) {
		var buf bytes.Buffer
		if err := tpl.Execute(&buf, data); err != nil {
			return 0, err
		}
		n, err := w.Write(buf.Bytes())
		return int64(n), err
	}
}

type Mailer struct {
	MailFrom   *mail.Address
	SendMethod sendMethod
}

// SendMail takes a content writen in Markdown to send it to the given user. It
// uses Markdown to create a HTML version of the message and leave the Markdown
// format in the text version. To perform sending, it relies on the SendMethod
// global variable.
func (r *Mailer) SendMail(to *mail.Address, subject, content string) (err error) {
	m := gomail.NewMsg()
	m.FromMailAddress(r.MailFrom)
	m.ToMailAddress(to)
	m.Subject(subject)

	toName := to.Name
	if len(toName) == 0 {
		toName = to.Address
	}

	tplData := map[string]string{
		"Lang":        "en",
		"To":          toName,
		"ToAddress":   to.Address,
		"Subject":     subject,
		"From":        r.MailFrom.Name,
		"FromAddress": r.MailFrom.Address,
		"Content":     content,
	}

	txtTpl, err := template.New("mailText").Parse(mailTXTTpl)
	if err != nil {
		return err
	}
	txtData := tplData
	m.SetBodyWriter(gomail.TypeTextPlain, templateWriteFunc(txtTpl, txtData))

	htmlContent, err := renderMailContent(content)
	if err != nil {
		return
	}

	if data, imgErr := web.GetEmbedFS().Open("build/img/happyDomain.png"); imgErr == nil {
		if err = m.EmbedReader("happydomain.png", data); err != nil {
			return
		}
	}

	htmlTpl, err := template.New("mailHTML").Parse(mailHTMLTpl)
	if err != nil {
		return err
	}
	htmlData := maps.Clone(tplData)
	htmlData["Content"] = htmlContent
	m.AddAlternativeWriter(gomail.TypeTextHTML, templateWriteFunc(htmlTpl, htmlData))

	if err = r.SendMethod.PrepareAndSend(m); err != nil {
		return
	}

	return
}

var (
	mailMarkdown = goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithXHTML(),
		),
	)

	// mailLoneLinkRe matches a paragraph made of a single link, as rendered by
	// goldmark before any style is added.
	mailLoneLinkRe = regexp.MustCompile(`(?m)^<p><a href="([^"]*)">([^<]*)</a></p>$`)

	// mailTagRe matches the opening of the HTML elements to style. Raw HTML
	// is not rendered from Markdown, so every "<" in the output opens a tag.
	mailTagRe = regexp.MustCompile(`<(h[1-4]|p|ul|ol|li|a|code|pre|blockquote|hr)\b( style=)?`)
)

// renderMailContent converts Markdown content to HTML suited for mail
// clients: each element carries its style inline, and a paragraph holding
// only a link becomes a call to action button.
func renderMailContent(content string) (string, error) {
	var buf bytes.Buffer
	if err := mailMarkdown.Convert([]byte(content), &buf); err != nil {
		return "", err
	}

	out := mailLoneLinkRe.ReplaceAllStringFunc(buf.String(), func(m string) string {
		sub := mailLoneLinkRe.FindStringSubmatch(m)
		return fmt.Sprintf(mailButtonTpl, sub[1], sub[2])
	})

	return mailTagRe.ReplaceAllStringFunc(out, func(m string) string {
		if strings.HasSuffix(m, " style=") {
			return m
		}
		return m + ` style="` + mailContentStyles[m[1:]] + `"`
	}), nil
}
