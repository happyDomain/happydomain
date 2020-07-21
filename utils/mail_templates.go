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
                                  <img alt="happyDNS" height="24" src="cid:happydns.png" style="border: 0px; height: 24px; line-height: 0px; outline: none; text-decoration: none; display: block;">
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
                                    <div style="font-family: Montserrat, Arial; font-size: 16px; line-height: 1.5; text-align: left; color: rgb(50, 54, 63);">Regards,<br>Fred - customer support @ happyDNS</div>
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
                                    <div style="font-family: Montserrat, Arial; font-size: 16px; font-weight: bold; line-height: 1.5; text-align: center; color: rgb(28, 180, 135);">happyDNS, finally a simple interface for domain names.</div>
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
                                      <a href="https://happydns.org/en/legal-notice" style="color: rgb(93, 97, 101);">Legal Notice</a>
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
Fred - customer support @ happyDNS
Legal notice: https://www.happydns.org/en/legal-notice/`
