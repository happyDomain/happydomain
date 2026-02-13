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

package captcha

import (
	"git.happydns.org/happyDomain/model"
)

// NewVerifier returns a CaptchaVerifier for the given provider.
// Each provider reads its own site key and secret key from flags registered in its init().
// Returns a no-op verifier when provider is empty.
func NewVerifier(provider string) happydns.CaptchaVerifier {
	switch provider {
	case "altcha":
		return NewAltchaVerifier()
	case "hcaptcha":
		return &hCaptchaVerifier{}
	case "recaptchav2":
		return &reCAPTCHAv2Verifier{}
	case "turnstile":
		return &turnstileVerifier{}
	default:
		return &noCaptcha{}
	}
}
