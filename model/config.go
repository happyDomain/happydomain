// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package happydns

import (
	"fmt"
	"net/mail"
	"net/url"
	"path"
)

// Options stores the configuration of the software.
type Options struct {
	// AdminBind is the address:port or unix socket used to serve the admin
	// API.
	AdminBind string

	// Bind is the address:port used to bind the main interface with API.
	Bind string

	// BasePath is the relative path where begins the root of the app.
	BasePath string

	// DevProxy is the URL that override static assets.
	DevProxy string

	// DefaultNameServer is the NS server suggested by default.
	DefaultNameServer string

	// DisableProviders should disallow all actions on provider (add/edit/delete) through public API.
	DisableProviders bool

	// DisableRegistration forbids all new registration using the public form/API.
	DisableRegistration bool

	// DisableEmbeddedLogin disables the internal user/password login in favor of ExternalAuth or OIDC.
	DisableEmbeddedLogin bool

	// ExternalAuth is the URL of the login form to use instead of the embedded one.
	ExternalAuth url.URL

	// ExternalURL keeps the URL used in communications (such as email,
	// ...), when it needs to use complete URL, not only relative parts.
	ExternalURL url.URL

	// JWTSecretKey stores the private key to sign and verify JWT tokens.
	JWTSecretKey []byte

	// JWTSigningMethod is the signing method to check token signature.
	JWTSigningMethod string

	// NoAuth controls if there is user access control or not.
	NoAuth bool

	// OptOutInsights disable the anonymous usage statistics report.
	OptOutInsights bool

	// StorageEngine points to the storage engine used.
	StorageEngine string

	ListmonkURL url.URL
	ListmonkId  int

	// MailFrom holds the content of the From field for all e-mails that
	// will be send.
	MailFrom mail.Address

	NoMail               bool
	MailSMTPHost         string
	MailSMTPPort         uint
	MailSMTPUsername     string
	MailSMTPPassword     string
	MailSMTPTLSSNoVerify bool

	OIDCClients []OIDCSettings
}

// GetBaseURL returns the full url to the absolute ExternalURL, including BaseURL.
func (o *Options) GetBaseURL() string {
	return fmt.Sprintf("%s%s", o.ExternalURL.String(), o.BasePath)
}

func (o *Options) GetAuthURL() *url.URL {
	redirecturl := o.ExternalURL
	redirecturl.Path = path.Join(redirecturl.Path, o.BasePath, "auth", "callback")
	return &redirecturl
}
