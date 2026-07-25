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
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// testPassword stands in for the configured Options.AdminPasswordHash.
const testPassword = "$2a$10$abcdefghijklmnopqrstuv"

// signForTest signs claims with the admin session key derived from secret and
// testPassword, so that tests exercising the claim checks are not rejected on
// the signature first.
func signForTest(t *testing.T, secret []byte, method jwt.SigningMethod, claims jwt.Claims) string {
	t.Helper()

	key, err := signingKey(secret, testPassword)
	if err != nil {
		t.Fatalf("unable to derive signing key: %s", err)
	}

	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("unable to sign token: %s", err)
	}

	return token
}

func TestVerifyAdminPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("s3cr3tpassword"), 12)
	if err != nil {
		t.Fatalf("unable to generate hash: %s", err)
	}

	tests := []struct {
		name       string
		configured string
		provided   string
		want       bool
	}{
		{"bcrypt correct", string(hash), "s3cr3tpassword", true},
		{"bcrypt wrong", string(hash), "wrongpassword", false},
		{"cleartext correct", "cleartextpw", "cleartextpw", true},
		{"cleartext wrong", "cleartextpw", "nope", false},
		{"empty configured rejects any", "", "anything", false},
		{"empty configured rejects empty", "", "", false},
		{"cleartext starting with $2 correct", "$2much$ecret!", "$2much$ecret!", true},
		{"cleartext starting with $2 wrong", "$2much$ecret!", "nope", false},
		{"truncated hash correct", string(hash[:20]), string(hash[:20]), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := VerifyAdminPassword(tc.configured, tc.provided); got != tc.want {
				t.Errorf("VerifyAdminPassword(%q, %q) = %v, want %v", tc.configured, tc.provided, got, tc.want)
			}
		})
	}
}

func TestIsHashed(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("whatever"), bcrypt.MinCost)
	if !IsHashed(string(hash)) {
		t.Errorf("IsHashed(%q) = false, want true", string(hash))
	}
	if IsHashed("cleartext") {
		t.Errorf("IsHashed(\"cleartext\") = true, want false")
	}
	if IsHashed("$2much$ecret!") {
		t.Errorf("IsHashed(\"$2much$ecret!\") = true, want false")
	}
	if IsHashed(string(hash[:20])) {
		t.Errorf("IsHashed(%q) = true, want false", string(hash[:20]))
	}
}

func TestIsMalformedHash(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("whatever"), bcrypt.MinCost)
	if IsMalformedHash(string(hash)) {
		t.Errorf("IsMalformedHash(%q) = true, want false", string(hash))
	}
	if IsMalformedHash("cleartext") {
		t.Errorf("IsMalformedHash(\"cleartext\") = true, want false")
	}
	if !IsMalformedHash(string(hash[:20])) {
		t.Errorf("IsMalformedHash(%q) = false, want true", string(hash[:20]))
	}
	if !IsMalformedHash("$2much$ecret!") {
		t.Errorf("IsMalformedHash(\"$2much$ecret!\") = false, want true")
	}
}

func TestIssueAndVerifyAdminToken(t *testing.T) {
	secret := []byte("this-is-a-32-byte-long-secret!!!")

	token, expiresAt, err := IssueAdminToken(secret, testPassword, time.Hour)
	if err != nil {
		t.Fatalf("IssueAdminToken returned error: %s", err)
	}

	if time.Until(expiresAt) <= 0 {
		t.Errorf("expiresAt is not in the future: %s", expiresAt)
	}

	if err := VerifyAdminToken(secret, testPassword, token); err != nil {
		t.Errorf("VerifyAdminToken rejected a freshly issued token: %s", err)
	}
}

func TestIssueAdminTokenClampsDuration(t *testing.T) {
	secret := []byte("another-32-byte-long-secret-key!")

	// A non-positive TTL falls back to the default lifetime.
	_, exp, err := IssueAdminToken(secret, testPassword, 0)
	if err != nil {
		t.Fatalf("IssueAdminToken(0) returned error: %s", err)
	}
	if d := time.Until(exp); d <= 0 || d > DefaultTokenTTL+time.Minute {
		t.Errorf("default TTL not applied, expiry in %s", d)
	}

	// A TTL above the maximum is clamped down.
	_, exp, err = IssueAdminToken(secret, testPassword, 10*24*time.Hour)
	if err != nil {
		t.Fatalf("IssueAdminToken(large) returned error: %s", err)
	}
	if d := time.Until(exp); d > MaxTokenTTL+time.Minute {
		t.Errorf("TTL not clamped to max, expiry in %s", d)
	}
}

func TestVerifyAdminTokenWrongSecret(t *testing.T) {
	token, _, err := IssueAdminToken([]byte("secret-key-number-one-padding-32"), testPassword, time.Hour)
	if err != nil {
		t.Fatalf("IssueAdminToken returned error: %s", err)
	}

	if err := VerifyAdminToken([]byte("secret-key-number-two-padding-32"), testPassword, token); err == nil {
		t.Error("VerifyAdminToken accepted a token signed with a different secret")
	}
}

// A token minted with the raw JWTSecretKey, as the external JWT issuer
// configured for user sessions could do, must not open the admin API: admin
// tokens are signed with a key derived from the admin password too.
func TestVerifyAdminTokenRawSecret(t *testing.T) {
	secret := []byte("raw-secret-forgery-test-key-3210")

	claims := jwt.RegisteredClaims{
		Audience:  jwt.ClaimStrings{adminAudience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString(secret)
	if err != nil {
		t.Fatalf("unable to sign token: %s", err)
	}

	if err := VerifyAdminToken(secret, testPassword, token); err == nil {
		t.Error("VerifyAdminToken accepted a token signed with the raw JWTSecretKey")
	}
}

// Rotating the admin password invalidates the sessions issued under the
// previous one.
func TestVerifyAdminTokenPasswordRotation(t *testing.T) {
	secret := []byte("rotation-test-secret-key-32-byte")

	token, _, err := IssueAdminToken(secret, testPassword, time.Hour)
	if err != nil {
		t.Fatalf("IssueAdminToken returned error: %s", err)
	}

	if err := VerifyAdminToken(secret, "$2a$10$rotatedrotatedrotatedro", token); err == nil {
		t.Error("VerifyAdminToken accepted a token issued under a previous password")
	}
}

func TestAdminTokenWithoutPassword(t *testing.T) {
	secret := []byte("no-password-test-secret-key-3210")

	if _, _, err := IssueAdminToken(secret, "", time.Hour); !errors.Is(err, ErrNoAdminPassword) {
		t.Errorf("IssueAdminToken without a password returned %v, want ErrNoAdminPassword", err)
	}

	token, _, err := IssueAdminToken(secret, testPassword, time.Hour)
	if err != nil {
		t.Fatalf("IssueAdminToken returned error: %s", err)
	}

	if err := VerifyAdminToken(secret, "", token); !errors.Is(err, ErrNoAdminPassword) {
		t.Errorf("VerifyAdminToken without a password returned %v, want ErrNoAdminPassword", err)
	}
}

func TestVerifyAdminTokenExpired(t *testing.T) {
	secret := []byte("expired-token-test-secret-key-32")

	claims := jwt.RegisteredClaims{
		Audience:  jwt.ClaimStrings{adminAudience},
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	}
	token := signForTest(t, secret, jwt.SigningMethodHS512, claims)

	if err := VerifyAdminToken(secret, testPassword, token); err == nil {
		t.Error("VerifyAdminToken accepted an expired token")
	}
}

func TestVerifyAdminTokenWrongAudience(t *testing.T) {
	secret := []byte("wrong-audience-test-secret-key-3")

	claims := jwt.RegisteredClaims{
		Audience:  jwt.ClaimStrings{"user"},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token := signForTest(t, secret, jwt.SigningMethodHS512, claims)

	if err := VerifyAdminToken(secret, testPassword, token); err == nil {
		t.Error("VerifyAdminToken accepted a token with the wrong audience")
	}
}

func TestVerifyAdminTokenWrongMethod(t *testing.T) {
	secret := []byte("wrong-method-test-secret-key-321")

	// A token signed with HS256 must be rejected: we only accept HS512.
	claims := jwt.RegisteredClaims{
		Audience:  jwt.ClaimStrings{adminAudience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token := signForTest(t, secret, jwt.SigningMethodHS256, claims)

	if err := VerifyAdminToken(secret, testPassword, token); err == nil {
		t.Error("VerifyAdminToken accepted a token signed with a disallowed method")
	}
}
