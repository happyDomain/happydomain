// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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
	"encoding/json"
)

// CaptchaVerifier is implemented by all captcha providers.
type CaptchaVerifier interface {
	// Provider returns the provider identifier ("hcaptcha", "recaptchav2", "turnstile", or "").
	Provider() string
	// SiteKey returns the public site key to be embedded in the frontend.
	SiteKey() string
	// Verify checks the token returned by the captcha widget.
	Verify(token, remoteIP string) error
}

type CaptchaLocalChallenge interface {
	NewChallenge() (json.RawMessage, error)
}

type FailureTracker interface {
	RecordFailure(ip, email string)
	RecordSuccess(ip, email string)
	RequiresCaptcha(ip, email string) bool
}
