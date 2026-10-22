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

// Colors and font stacks follow the web UI theme (web/src/app.scss). Mail
// clients don't load web fonts reliably, so each stack falls back to system
// fonts.
const (
	mailFontTitle = `Lexend, Roboto, 'Helvetica Neue', Helvetica, Arial, sans-serif`
	mailFontText  = `Roboto, 'Helvetica Neue', Helvetica, Arial, sans-serif`
	mailFontMono  = `'JetBrains Mono', SFMono-Regular, Menlo, Consolas, monospace`

	mailColorText   = "#282828" // brand Black, never pure black
	mailColorMuted  = "#5e5e56" // sand-700
	mailColorLink   = "#016f4f" // green-700, green as text on light
	mailColorBrand  = "#00aa7d" // green-500, fills carry #282828 text
	mailColorBorder = "#dcdcce" // sand-200
	mailColorCode   = "#f4f5f5" // marble
	mailColorSpark  = "#a454ff" // purple-500
)

// mailContentStyles holds the inline style applied to each HTML element
// produced by the Markdown renderer, as many mail clients drop <style> blocks.
var mailContentStyles = map[string]string{
	"h1":         "margin: 0 0 16px; font-family: " + mailFontTitle + "; font-size: 26px; line-height: 1.3; font-weight: 600; color: " + mailColorText + ";",
	"h2":         "margin: 0 0 16px; font-family: " + mailFontTitle + "; font-size: 22px; line-height: 1.3; font-weight: 600; color: " + mailColorText + ";",
	"h3":         "margin: 24px 0 8px; font-family: " + mailFontTitle + "; font-size: 18px; line-height: 1.3; font-weight: 600; color: " + mailColorText + ";",
	"h4":         "margin: 24px 0 8px; font-family: " + mailFontTitle + "; font-size: 16px; line-height: 1.3; font-weight: 600; color: " + mailColorText + ";",
	"p":          "margin: 0 0 16px;",
	"ul":         "margin: 0 0 16px; padding-left: 24px;",
	"ol":         "margin: 0 0 16px; padding-left: 24px;",
	"li":         "margin: 0 0 6px;",
	"a":          "color: " + mailColorLink + "; text-decoration: underline;",
	"code":       "font-family: " + mailFontMono + "; font-size: 14px; color: " + mailColorText + "; background-color: " + mailColorCode + "; border-radius: 4px; padding: 1px 5px;",
	"pre":        "margin: 0 0 16px; padding: 12px 16px; background-color: " + mailColorCode + "; border-radius: 8px; overflow-x: auto; white-space: pre-wrap;",
	"blockquote": "margin: 0 0 16px; padding: 4px 16px; border-left: 3px solid " + mailColorSpark + "; color: " + mailColorMuted + ";",
	"hr":         "margin: 24px 0; border: 0; border-top: 1px solid " + mailColorBorder + ";",
}

// mailButtonTpl renders a paragraph holding a single link as a call to action
// button, followed by the bare address for clients that mangle the button.
// Both %[1]s are the link target and %[2]s is the label, already escaped by
// the Markdown renderer.
const mailButtonTpl = `<table role="presentation" border="0" cellpadding="0" cellspacing="0" style="margin: 8px 0 24px; border-collapse: separate;">
  <tr>
    <td align="center" bgcolor="` + mailColorBrand + `" style="border-radius: 8px; background-color: ` + mailColorBrand + `;">
      <a style="display: inline-block; padding: 14px 28px; font-family: ` + mailFontTitle + `; font-size: 16px; font-weight: 600; line-height: 20px; color: ` + mailColorText + `; text-decoration: none; border-radius: 8px;" href="%[1]s" target="_blank">%[2]s</a>
    </td>
  </tr>
</table>
<p style="margin: 0 0 16px; font-size: 13px; line-height: 1.5; color: ` + mailColorMuted + `;">If the button doesn't work, copy this address into your browser:<br /><a style="color: ` + mailColorLink + `; word-break: break-all;" href="%[1]s">%[1]s</a></p>`

// mailHTMLTpl is the HTML template used for email.
const mailHTMLTpl = `<!DOCTYPE html>
<html lang="{{ .Lang }}">
  <head>
    <meta charset="UTF-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="color-scheme" content="light">
    <meta name="supported-color-schemes" content="light">
    <title>{{ .Subject }}</title>
    <!--[if mso]><style>body, table, td, a, p, div { font-family: Arial, sans-serif !important; }</style><![endif]-->
    <style>
      @media only screen and (max-width: 620px) {
        .hd-outer { padding: 24px 16px !important; }
        .hd-card { padding: 28px 20px !important; }
      }
    </style>
  </head>
  <body style="margin: 0; padding: 0; background-color: #f7f7e8; -webkit-text-size-adjust: 100%; text-size-adjust: 100%;">
    <table role="presentation" border="0" cellpadding="0" cellspacing="0" width="100%" bgcolor="#f7f7e8" style="border-collapse: collapse; background-color: #f7f7e8;">
      <tr>
        <td class="hd-outer" align="center" style="padding: 40px 16px;">
          <table role="presentation" border="0" cellpadding="0" cellspacing="0" width="100%" style="border-collapse: collapse; max-width: 600px;">
            <tr>
              <td style="padding: 0 4px 24px;">
                <img alt="HappyDomain" src="cid:happydomain.png" width="231" height="28" style="display: block; width: 231px; height: 28px; border: 0; outline: none; text-decoration: none;">
              </td>
            </tr>
            <tr>
              <td bgcolor="#fffff7" style="background-color: #fffff7; border: 1px solid #dcdcce; border-top: 4px solid #00aa7d; border-radius: 12px;">
                <table role="presentation" border="0" cellpadding="0" cellspacing="0" width="100%" style="border-collapse: collapse;">
                  <tr>
                    <td class="hd-card" style="padding: 40px; font-family: Roboto, 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 16px; line-height: 1.6; color: #282828; word-break: break-word;">
                      {{ .Content }}
                      <p style="margin: 24px 0 0; padding-top: 24px; border-top: 1px solid #dcdcce; color: #5e5e56;">Regards,<br><strong style="color: #282828;">Fred</strong>, customer support at HappyDomain</p>
                    </td>
                  </tr>
                </table>
              </td>
            </tr>
            <tr>
              <td align="center" style="padding: 24px 16px 0; font-family: Roboto, 'Helvetica Neue', Helvetica, Arial, sans-serif; font-size: 13px; line-height: 1.6; color: #5e5e56;">
                <strong style="font-family: Lexend, Roboto, 'Helvetica Neue', Helvetica, Arial, sans-serif; font-weight: 600; color: #016f4f;">HappyDomain</strong>, finally a simple interface for domain names.<br>
                <a href="https://www.happydomain.org/" style="color: #5e5e56; text-decoration: underline;">happydomain.org</a>
                &nbsp;&middot;&nbsp;
                <a href="https://www.happydomain.org/en/legal-notice/" style="color: #5e5e56; text-decoration: underline;">Legal notice</a>
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
  </body>
</html>`

// mailTXTTpl is the template used for text emails.
const mailTXTTpl = "{{ .Content }}\n\n-- \nFred, customer support at HappyDomain\nLegal notice: https://www.happydomain.org/en/legal-notice/"
