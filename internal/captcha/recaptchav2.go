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

package captcha

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var (
	recaptchav2SiteKey   string
	recaptchav2SecretKey string
)

func init() {
	flag.StringVar(&recaptchav2SiteKey, "recaptcha-site-key", "", "Public site key for Google reCAPTCHA v2")
	flag.StringVar(&recaptchav2SecretKey, "recaptcha-secret-key", "", "Secret key for Google reCAPTCHA v2 server-side token verification")
}

type reCAPTCHAv2Verifier struct{}

func (r *reCAPTCHAv2Verifier) Provider() string { return "recaptchav2" }
func (r *reCAPTCHAv2Verifier) SiteKey() string  { return recaptchav2SiteKey }

func (r *reCAPTCHAv2Verifier) Verify(token, remoteIP string) error {
	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
		"secret":   {recaptchav2SecretKey},
		"response": {token},
		"remoteip": {remoteIP},
	})
	if err != nil {
		return fmt.Errorf("captcha verification request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success    bool     `json:"success"`
		ErrorCodes []string `json:"error-codes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("captcha response decode failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("captcha verification failed: %s", strings.Join(result.ErrorCodes, ", "))
	}

	return nil
}
