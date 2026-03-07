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

//go:build !nooidc

package controller

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

const (
	SESSION_KEY_OIDC_STATE = "oidc-state"
	SESSION_KEY_OIDC_PKCE  = "oidc-pkce"
	SESSION_KEY_OIDC_NONCE = "oidc-nonce"
)

type OIDCProvider struct {
	config       *happydns.Options
	authService  happydns.AuthenticationUsecase
	oauth2config *oauth2.Config
	oidcVerifier *oidc.IDTokenVerifier
}

func GetOIDCProvider(o *happydns.Options, ctx context.Context) (*oidc.Provider, error) {
	return oidc.NewProvider(ctx, strings.TrimSuffix(o.OIDCClients[0].ProviderURL.String(), "/.well-known/openid-configuration"))
}

func GetOAuth2Config(o *happydns.Options, provider *oidc.Provider) *oauth2.Config {
	oauth2Config := oauth2.Config{
		ClientID:     o.OIDCClients[0].ClientID,
		ClientSecret: o.OIDCClients[0].ClientSecret,
		RedirectURL:  o.GetAuthURL().String(),
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return &oauth2Config
}

func NewOIDCProvider(cfg *happydns.Options, authService happydns.AuthenticationUsecase) *OIDCProvider {
	// Initialize OIDC
	provider, err := GetOIDCProvider(cfg, context.Background())
	if err != nil {
		log.Fatal("Unable to instantiate OIDC Provider:", err)
	}

	oauth2Config := GetOAuth2Config(cfg, provider)

	oidcVerifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.OIDCClients[0].ClientID,
	})

	return &OIDCProvider{
		config:       cfg,
		authService:  authService,
		oauth2config: oauth2Config,
		oidcVerifier: oidcVerifier,
	}
}

func (p *OIDCProvider) RedirectOIDC(c *gin.Context) {
	session := sessions.Default(c)

	state := make([]byte, 32)
	_, err := rand.Read(state)
	if err != nil {
		log.Println("Unable to redirect_OIDC, rand.Read fails:", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to respond to your request. Please retry later."})
		return
	}

	nonce := make([]byte, 32)
	if _, err = rand.Read(nonce); err != nil {
		log.Println("Unable to redirect_OIDC, rand.Read fails:", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to respond to your request. Please retry later."})
		return
	}
	nonceStr := hex.EncodeToString(nonce)

	pkceVerifier := oauth2.GenerateVerifier()

	session.Set(SESSION_KEY_OIDC_STATE, hex.EncodeToString(state))
	session.Set(SESSION_KEY_OIDC_PKCE, pkceVerifier)
	session.Set(SESSION_KEY_OIDC_NONCE, nonceStr)
	err = session.Save()
	if err != nil {
		log.Println("Unable to redirect_OIDC, session.Save fails:", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to respond to your request. Please retry later."})
		return
	}

	c.Redirect(http.StatusFound, p.oauth2config.AuthCodeURL(hex.EncodeToString(state), oauth2.S256ChallengeOption(pkceVerifier), oauth2.SetAuthURLParam("nonce", nonceStr)))
}

func (p *OIDCProvider) CompleteOIDC(c *gin.Context) {
	session := sessions.Default(c)

	state := c.Query("state")

	if state != session.Get(SESSION_KEY_OIDC_STATE) {
		log.Printf("Invalid CSRF token on /auth/callback: got %q, expected %q", state, session.Get(SESSION_KEY_OIDC_STATE))
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: "Invalid CSRF token"})
		return
	}

	pkceVerifier, _ := session.Get(SESSION_KEY_OIDC_PKCE).(string)
	expectedNonce, _ := session.Get(SESSION_KEY_OIDC_NONCE).(string)

	session.Delete(SESSION_KEY_OIDC_STATE)
	session.Delete(SESSION_KEY_OIDC_PKCE)
	session.Delete(SESSION_KEY_OIDC_NONCE)
	err := session.Save()
	if err != nil {
		log.Println("Unable to CompleteOIDC, session.Save fails:", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to respond to your request. Please retry later."})
		return
	}

	oauth2Token, err := p.oauth2config.Exchange(c, c.Query("code"), oauth2.VerifierOption(pkceVerifier))
	if err != nil {
		log.Printf("CompleteOIDC: failed to exchange token: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to respond to your request. Please retry later."})
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Printf("CompleteOIDC: no id_token field in oauth2 token")
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to respond to your request. Please retry later."})
		return
	}

	idToken, err := p.oidcVerifier.Verify(c, rawIDToken)
	if err != nil {
		log.Printf("CompleteOIDC: failed to verify ID Token: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to respond to your request. Please retry later."})
		return
	}

	if idToken.Nonce != expectedNonce {
		log.Printf("CompleteOIDC: nonce mismatch: got %q, expected %q", idToken.Nonce, expectedNonce)
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: "Invalid nonce in ID token"})
		return
	}

	var claims map[string]any
	if err = idToken.Claims(&claims); err != nil {
		log.Printf("CompleteOIDC: unable to retrieve user profile: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to respond to your request. Please retry later."})
		return
	}

	var profile happydns.UserAuth

	if email, ok := claims["email"].(string); ok {
		profile.Email = email
	}
	if _, ok := claims["email_verified"].(bool); ok {
		now := time.Now()
		profile.EmailVerification = &now
	}

	if len(profile.Id) == 0 {
		if len(profile.Email) == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: "No email nor user identifier found in OIDC profile."})
			return
		}

		hash := sha256.Sum256([]byte(profile.Email))
		profile.Id = hash[:]
	}

	_, err = p.authService.CompleteAuthentication(&profile)
	if err != nil {
		log.Printf("CompleteOIDC: unable to complete authentication: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to respond to your request. Please retry later."})
		return
	}

	middleware.SessionLoginOK(c, &profile)

	c.Redirect(http.StatusFound, p.config.GetBaseURL()+"/")
}
