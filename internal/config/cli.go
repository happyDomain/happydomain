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
	"runtime"
	"strings"
	"time"

	"git.happydns.org/happyDomain/internal/checker"
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
	flag.IntVar(&o.CheckerMaxConcurrency, "checker-max-concurrency", runtime.NumCPU(), "Maximum number of checker jobs that can run simultaneously")
	flag.IntVar(&o.CheckerRetentionDays, "checker-retention-days", 365, "System-wide default retention horizon for check execution history (overridable per user)")
	flag.DurationVar(&o.CheckerJanitorInterval, "checker-janitor-interval", 6*time.Hour, "How often the checker retention janitor runs")
	flag.IntVar(&o.CheckerInactivityPauseDays, "checker-inactivity-pause-days", 90, "Pause checks for users that haven't logged in for this many days (0 disables, overridable per user)")
	flag.IntVar(&o.CheckerMaxChecksPerDay, "checker-max-checks-per-day", 0, "System-wide default cap on scheduled checker executions per user per day; counter resets at 00:00 UTC and is in-memory only (0 = unlimited, overridable per user; see docs/checker-quotas.md)")
	flag.BoolVar(&o.CheckerCountManualTriggers, "checker-count-manual-triggers", true, "When true (default), manual checker triggers count against UserQuota.MaxChecksPerDay and are refused with HTTP 429 once exhausted; when false, manual triggers bypass the quota entirely (see docs/checker-quotas.md)")
	flag.BoolVar(&o.DisableCheckerScheduler, "disable-checker-scheduler", o.DisableCheckerScheduler, "Prevent the checker scheduler from starting automatically at boot (it can still be enabled at runtime through the admin API)")

	flag.Var(&URL{&o.ListmonkURL}, "newsletter-server-url", "Base URL of the listmonk newsletter server")
	flag.IntVar(&o.ListmonkID, "newsletter-id", 1, "Listmonk identifier of the list receiving the new user")

	flag.BoolVar(&o.NoMail, "no-mail", o.NoMail, "Disable all automatic mails, skip email verification at registration")
	flag.Var(&mailAddress{&o.MailFrom}, "mail-from", "Define the sender name and address for all e-mail sent")
	flag.StringVar(&o.MailSMTPHost, "mail-smtp-host", o.MailSMTPHost, "Use the given SMTP server as default way to send emails")
	flag.UintVar(&o.MailSMTPPort, "mail-smtp-port", o.MailSMTPPort, "Define the port to use to send e-mail through SMTP method")
	flag.StringVar(&o.MailSMTPUsername, "mail-smtp-username", o.MailSMTPUsername, "If the SMTP server requires authentication, fill with the username to authenticate with")
	flag.StringVar(&o.MailSMTPPassword, "mail-smtp-password", o.MailSMTPPassword, "Password associated with the given username for SMTP authentication")
	flag.BoolVar(&o.MailSMTPTLSSNoVerify, "mail-smtp-tls-no-verify", o.MailSMTPTLSSNoVerify, "Do not verify certificate validity on SMTP connection")

	flag.StringVar(&o.MailAutoconfigHost, "mail-autoconfig-host", o.MailAutoconfigHost, "Public FQDN serving Mozilla Autoconfig and Microsoft Autodiscover (defaults to externalurl host)")

	flag.StringVar(&o.CaptchaProvider, "captcha-provider", o.CaptchaProvider, "Captcha provider to use for bot protection (altcha, hcaptcha, recaptchav2, turnstile, or empty to disable)")
	flag.IntVar(&o.CaptchaLoginThreshold, "captcha-login-threshold", 3, "Number of failed login attempts before captcha is required (0 = always require when provider configured)")

	flag.Var(&stringSlice{&o.PluginsDirectories}, "plugins-directory", "Path to a directory containing checker plugins (.so files); may be repeated")

	// Register one -checker-<id>-<opt-id> flag per registered checker AdminOpt.
	// Checkers register themselves in init() of the blank-imported `checkers`
	// package, so by the time declareFlags runs the registry is fully
	// populated. Values set here win over the same options stored in the DB
	// (handled by the checker engine when merging options).
	if o.CheckerAdminOptions == nil {
		o.CheckerAdminOptions = map[string]happydns.CheckerOptions{}
	}
	for id, def := range checker.GetCheckers() {
		if len(def.Options.AdminOpts) == 0 {
			continue
		}
		if o.CheckerAdminOptions[id] == nil {
			o.CheckerAdminOptions[id] = happydns.CheckerOptions{}
		}
		opts := o.CheckerAdminOptions[id]
		for _, opt := range def.Options.AdminOpts {
			flag.Var(
				&checkerOptionFlag{Opts: opts, Key: opt.Id, Type: opt.Type},
				fmt.Sprintf("checker-%s-%s", id, strings.ToLower(opt.Id)),
				adminOptFlagUsage(id, opt),
			)
		}
	}

	// Others flags are declared in some other files likes sources, storages, ... when they need specials configurations
}

// adminOptFlagUsage returns the help string shown for a checker AdminOpt
// CLI flag. It prefers the option's own description, falling back to its
// label, and always names the checker so the flag's purpose is unambiguous.
func adminOptFlagUsage(checkerID string, opt happydns.CheckerOptionDocumentation) string {
	switch {
	case opt.Description != "":
		return fmt.Sprintf("[checker %s] %s", checkerID, opt.Description)
	case opt.Label != "":
		return fmt.Sprintf("[checker %s] %s", checkerID, opt.Label)
	default:
		return fmt.Sprintf("Admin-scope option %q for checker %q", opt.Id, checkerID)
	}
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
