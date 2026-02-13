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

package config // import "git.happydns.org/happyDomain/config"

import (
	"flag"
	"fmt"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

// declareFlags registers flags for the structure Options.
func declareFlags(o *happydns.Options) {
	flag.StringVar(&o.DevProxy, "dev", o.DevProxy, "Proxify traffic to this host for static assets")
	flag.StringVar(&o.AdminBind, "admin-bind", o.AdminBind, "Bind port/socket for administration interface")
	flag.StringVar(&o.Bind, "bind", ":8081", "Bind port/socket")
	flag.BoolVar(&o.DisableProviders, "disable-providers-edit", o.DisableProviders, "Disallow all actions on provider (add/edit/delete)")
	flag.BoolVar(&o.DisableRegistration, "disable-registration", o.DisableRegistration, "Forbids new account creation through public form/API (still allow registration from external services)")
	flag.BoolVar(&o.DisableEmbeddedLogin, "disable-embedded-login", o.DisableEmbeddedLogin, "Disables the internal user/password login in favor of external-auth or OIDC")
	flag.Var(&URL{&o.ExternalURL}, "externalurl", "Begining of the URL, before the base, that should be used eg. in mails")
	flag.StringVar(&o.BasePath, "baseurl", o.BasePath, "URL prepended to each URL")
	flag.StringVar(&o.DefaultNameServer, "default-ns", o.DefaultNameServer, "Adress to the default name server")
	flag.StringVar(&o.StorageEngine, "storage-engine", o.StorageEngine, fmt.Sprintf("Select the storage engine between %v", storage.GetStorageEngines()))
	flag.BoolVar(&o.NoAuth, "no-auth", false, "Disable user access control, use default account")
	flag.Var(&JWTSecretKey{&o.JWTSecretKey}, "jwt-secret-key", "Secret key used to verify JWT authentication tokens (a random secret is used if undefined)")
	flag.Var(&URL{&o.ExternalAuth}, "external-auth", "Base URL to use for login and registration (use embedded forms if left empty)")
	flag.BoolVar(&o.OptOutInsights, "opt-out-insights", false, "Disable the anonymous usage statistics report. If you care about this project and don't participate in discussions, don't opt-out.")

	flag.Var(&URL{&o.ListmonkURL}, "newsletter-server-url", "Base URL of the listmonk newsletter server")
	flag.IntVar(&o.ListmonkId, "newsletter-id", 1, "Listmonk identifier of the list receiving the new user")

	flag.BoolVar(&o.NoMail, "no-mail", o.NoMail, "Disable all automatic mails, skip email verification at registration")
	flag.Var(&mailAddress{&o.MailFrom}, "mail-from", "Define the sender name and address for all e-mail sent")
	flag.StringVar(&o.MailSMTPHost, "mail-smtp-host", o.MailSMTPHost, "Use the given SMTP server as default way to send emails")
	flag.UintVar(&o.MailSMTPPort, "mail-smtp-port", o.MailSMTPPort, "Define the port to use to send e-mail through SMTP method")
	flag.StringVar(&o.MailSMTPUsername, "mail-smtp-username", o.MailSMTPUsername, "If the SMTP server requires authentication, fill with the username to authenticate with")
	flag.StringVar(&o.MailSMTPPassword, "mail-smtp-password", o.MailSMTPPassword, "Password associated with the given username for SMTP authentication")
	flag.BoolVar(&o.MailSMTPTLSSNoVerify, "mail-smtp-tls-no-verify", o.MailSMTPTLSSNoVerify, "Do not verify certificate validity on SMTP connection")

	flag.StringVar(&o.CaptchaProvider, "captcha-provider", o.CaptchaProvider, "Captcha provider to use for bot protection (altcha, hcaptcha, recaptchav2, turnstile, or empty to disable)")
	flag.IntVar(&o.CaptchaLoginThreshold, "captcha-login-threshold", 3, "Number of failed login attempts before captcha is required (0 = always require when provider configured)")

	// Others flags are declared in some other files likes sources, storages, ... when they need specials configurations
}

// parseCLI parse the flags and treats extra args as configuration filename.
func parseCLI(o *happydns.Options) error {
	flag.Parse()

	for _, conf := range flag.Args() {
		err := parseFile(o, conf)
		if err != nil {
			return err
		}
	}

	return nil
}
