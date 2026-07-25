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

package adminauth

import (
	"crypto/hkdf"
	"crypto/sha512"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// adminAudience marks a token as scoped to the admin interface, so an admin
	// session token cannot be confused with a user-session JWT signed with the
	// same JWTSecretKey.
	adminAudience = "admin"

	// adminSigningMethod is the JWT signing method used for admin tokens.
	adminSigningMethod = "HS512"

	// adminKeyInfo domain-separates the admin session key derivation.
	adminKeyInfo = "happydomain admin session token v1"

	// adminKeyLength is the size, in bytes, of the derived HS512 signing key.
	adminKeyLength = 64

	// DefaultTokenTTL is the session lifetime used when the operator does not
	// request a specific duration.
	DefaultTokenTTL = time.Hour

	// MaxTokenTTL caps the session lifetime an operator may request.
	MaxTokenTTL = 24 * time.Hour
)

// ErrNoAdminPassword is returned when an admin session key is requested while
// no admin password is configured: without a password there is nothing to bind
// a session to, so no admin token can be issued or verified.
var ErrNoAdminPassword = errors.New("no admin password configured")

// signingKey derives the admin session signing key from secret (the server
// JWTSecretKey) and the configured admin password.
//
// The JWTSecretKey alone must not sign admin tokens: it is shared with the
// external JWT issuer configured for user sessions (Auth0 and friends), which
// could otherwise mint an admin token without knowing the admin password.
// Mixing the password into the key also makes rotating the password invalidate
// every session issued under the previous one.
func signingKey(secret []byte, password string) ([]byte, error) {
	if password == "" {
		return nil, ErrNoAdminPassword
	}

	return hkdf.Key(sha512.New, secret, []byte(password), adminKeyInfo, adminKeyLength)
}

// IssueAdminToken signs a new admin session token, valid for ttl, with a key
// derived from secret (the server JWTSecretKey) and the configured admin
// password (the bcrypt hash or cleartext held in Options.AdminPasswordHash).
// ttl is clamped to (0, MaxTokenTTL]; a non-positive ttl falls back to
// DefaultTokenTTL. It returns the signed token and its absolute expiry time.
func IssueAdminToken(secret []byte, password string, ttl time.Duration) (string, time.Time, error) {
	key, err := signingKey(secret, password)
	if err != nil {
		return "", time.Time{}, err
	}

	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}
	if ttl > MaxTokenTTL {
		ttl = MaxTokenTTL
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	claims := jwt.RegisteredClaims{
		Audience:  jwt.ClaimStrings{adminAudience},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString(key)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

// VerifyAdminToken checks that token is a valid, unexpired admin session token
// signed with the key derived from secret and the currently configured admin
// password. It enforces the HS512 signing method and the admin audience, and
// returns a non-nil error otherwise. Tokens issued under a previous password
// no longer verify.
func VerifyAdminToken(secret []byte, password, token string) error {
	key, err := signingKey(secret, password)
	if err != nil {
		return err
	}

	_, err = jwt.ParseWithClaims(token, &jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) {
			return key, nil
		},
		jwt.WithValidMethods([]string{adminSigningMethod}),
		jwt.WithAudience(adminAudience),
	)
	return err
}
