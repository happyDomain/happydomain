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

// mailHTMLTpl is the HTML template used for email.
const mailHTMLTpl = `
<!DOCTYPE html>
<html lang="{{ .Lang }}">
  <head>
    <meta charset="UTF-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>{{ .Subject }}</title>
  </head>
  <body style="margin: 0; text-indent: 0px; text-transform: none; white-space: normal; word-spacing: 0px; background-color: rgb(255, 255, 255); text-decoration: none;">
    <div style="caret-color: rgb(0, 0, 0); font-family: Helvetica; font-size: 12px; font-style: normal; font-variant-caps: normal; font-weight: normal; letter-spacing: normal; text-align: start; text-indent: 0px; text-transform: none; white-space: normal; word-spacing: 0px; -webkit-text-stroke-width: 0px; background-color: rgb(255, 255, 255); text-decoration: none;">
      <div style="margin: 0px auto; max-width: 600px;">
        <table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-collapse: collapse; width: 600px;">
          <tbody>
            <tr>
              <td style="direction: ltr; font-size: 0px; padding: 0px; text-align: center; vertical-align: top;">
                <div class="mj-column-per-100 outlook-group-fix" style="max-width: 100%; width: 600px; font-size: 13px; text-align: left; direction: ltr; display: inline-block; vertical-align: top;">
                  <table border="0" cellpadding="0" cellspacing="0" role="presentation" width="100%" style="border-collapse: collapse; vertical-align: top;">
                    <tbody>
                      <tr>
                        <td align="left" style="border-collapse: collapse; font-size: 0px; padding: 16px 0px 0px 0px; word-break: break-word;">
                          <table border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-collapse: collapse; border-spacing: 0px;">
                            <tbody>
                              <tr>
                                <td style="border-collapse: collapse; font-size: 30px;">
                                  <img alt="happyDomain" height="24" src="cid:happydomain.png" style="border: 0px; height: 24px; line-height: 0px; outline: none; text-decoration: none; display: block;">
                                </td>
                              </tr>
                            </tbody>
                          </table>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
        <table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-collapse: collapse; background-color: rgb(255, 255, 255); width: 600px; background-position: initial initial; background-repeat: initial initial;">
          <tbody>
            <tr>
              <td style="border-collapse: collapse; direction: ltr; font-size: 0px; padding: 0px; text-align: center; vertical-align: top;">
                <div style="margin: 0px auto; max-width: 600px;">
                  <table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-collapse: collapse; width: 600px;">
                    <tbody>
                      <tr>
                        <td style="border-collapse: collapse; direction: ltr; font-size: 0px; padding: 0px 0px 8px 0px; text-align: center; vertical-align: top;">
                          <div class="mj-column-per-100 outlook-group-fix" style="max-width: 100%; width: 600px; font-size: 13px; text-align: left; direction: ltr; display: inline-block; vertical-align: top;">
                            <table border="0" cellpadding="0" cellspacing="0" role="presentation" width="100%" style="border-collapse: collapse; vertical-align: top;">
                              <tbody>
                                <tr>
                                  <td align="left" style="border-collapse: collapse; font-family: Montserrat, Arial; font-size: 16px; line-height: 1.5; text-align: left; color: rgb(50, 54, 63); padding: 0px 24px 8px; word-break: break-word;">
                                    {{ .Content }}
                                  </td>
                                </tr>
                                <tr>
                                  <td align="left" style="border-collapse: collapse; font-size: 0px; padding: 0px 24px 16px; word-break: break-word;">
                                    <div style="font-family: Montserrat, Arial; font-size: 16px; line-height: 1.5; text-align: left; color: rgb(50, 54, 63);">Regards,<br>Fred - customer support @ happyDomain</div>
                                  </td>
                                </tr>
                              </tbody>
                            </table>
                          </div>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div style="margin: 0px auto; max-width: 600px;">
        <table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-collapse: collapse; width: 600px;">
          <tbody>
            <tr>
              <td style="border-collapse: collapse; direction: ltr; font-size: 0px; padding: 0px; text-align: center; vertical-align: top;">
                <div style="margin: 0px auto; max-width: 600px;">
                  <table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-collapse: collapse; width: 600px;">
                    <tbody>
                      <tr>
                        <td style="border-collapse: collapse; direction: ltr; font-size: 0px; padding: 0px; text-align: center; vertical-align: top;">
                          <div class="mj-column-per-100 outlook-group-fix" style="max-width: 100%; width: 600px; font-size: 13px; text-align: left; direction: ltr; display: inline-block; vertical-align: top;">
                            <table border="0" cellpadding="0" cellspacing="0" role="presentation" width="100%" style="border-collapse: collapse; vertical-align: top;">
                              <tbody>
                                <tr>
                                  <td align="center" style="border-collapse: collapse; font-size: 0px; padding: 0px; word-break: break-word;">
                                    <div style="font-family: Montserrat, Arial; font-size: 16px; font-weight: bold; line-height: 1.5; text-align: center; color: rgb(28, 180, 135);">happyDomain, finally a simple interface for domain names.</div>
                                  </td>
                                </tr>
                              </tbody>
                            </table>
                          </div>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <div style="margin: 0px auto; max-width: 600px;">
                  <table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-collapse: collapse; width: 600px;">
                    <tbody>
                      <tr>
                        <td style="border-collapse: collapse; direction: ltr; font-size: 0px; padding: 0px; text-align: center; vertical-align: top;">
                          <div class="mj-column-per-100 outlook-group-fix" style="max-width: 100%; width: 600px; font-size: 13px; text-align: left; direction: ltr; display: inline-block; vertical-align: top;">
                            <table border="0" cellpadding="0" cellspacing="0" role="presentation" width="100%" style="border-collapse: collapse; vertical-align: top;">
                              <tbody>
                                <tr>
                                  <td align="center" style="border-collapse: collapse; font-size: 0px; padding: 10px 25px; word-break: break-word;">
                                    <div style="font-family: Montserrat, Arial; font-size: 12px; line-height: 1.5; text-align: center; color: rgb(50, 54, 63);">
                                      <a href="https://happydomain.org/en/legal-notice" style="color: rgb(93, 97, 101);">Legal Notice</a>
                                    </div>
                                  </td>
                                </tr>
                              </tbody>
                            </table>
                          </div>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </body>
</html>`

// mailHTMLTpl is the template used for text emails.
const mailTXTTpl = `{{ .Content }}

--
Fred - customer support @ happyDomain
Legal notice: https://www.happydomain.org/en/legal-notice/`
