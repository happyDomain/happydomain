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
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	altcha "github.com/altcha-org/altcha-lib-go"
)

var (
	altchaComplexity int64
	altchaHMACKey    string
)

func init() {
	flag.Int64Var(&altchaComplexity, "altcha-complexity", 100_000, "Serves as a measure to balance security against automated abuse/spam and user experience")
	flag.StringVar(&altchaHMACKey, "altcha-hmac-key", "", "Secret HMAC key for Altcha challenge signing and verification")
}

type AltchaVerifier struct {
	options altcha.ChallengeOptions
}

func NewAltchaVerifier() *AltchaVerifier {
	if altchaHMACKey == "" {
		b := make([]byte, 24)
		_, err := rand.Read(b)
		if err != nil {
			log.Fatalf("error generating Altcha HMAC key: %v", err)
		}
		altchaHMACKey = base64.URLEncoding.EncodeToString(b)[:32]
	}

	return &AltchaVerifier{
		options: altcha.ChallengeOptions{
			HMACKey:   altchaHMACKey,
			MaxNumber: altchaComplexity,
		},
	}
}

func (a *AltchaVerifier) Provider() string { return "altcha" }
func (a *AltchaVerifier) SiteKey() string  { return "" }

func (a *AltchaVerifier) Verify(token, _ string) error {
	ok, err := altcha.VerifySolution(token, altchaHMACKey, true)
	if err != nil {
		return fmt.Errorf("altcha verification failed: %w", err)
	}
	if !ok {
		return fmt.Errorf("altcha verification failed: invalid solution")
	}
	return nil
}

// NewAltchaChallenge generates a new Altcha challenge to be served to the frontend.
func (a *AltchaVerifier) NewChallenge() (json.RawMessage, error) {
	challenge, err := altcha.CreateChallenge(a.options)
	if err != nil {
		return nil, fmt.Errorf("failed to create altcha challenge: %w", err)
	}

	data, err := json.Marshal(challenge)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal altcha challenge: %w", err)
	}

	return data, nil
}
