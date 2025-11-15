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

package api

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
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

	session.Set(SESSION_KEY_OIDC_STATE, hex.EncodeToString(state))
	err = session.Save()
	if err != nil {
		log.Println("Unable to redirect_OIDC, session.Save fails:", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to respond to your request. Please retry later."})
		return
	}

	c.Redirect(http.StatusFound, p.oauth2config.AuthCodeURL(hex.EncodeToString(state)))
}

func (p *OIDCProvider) CompleteOIDC(c *gin.Context) {
	session := sessions.Default(c)

	state := c.Query("state")

	if state != session.Get(SESSION_KEY_OIDC_STATE) {
		log.Printf("Invalid CSRF token on /auth/callback: got %q, expected %q", state, session.Get(SESSION_KEY_OIDC_STATE))
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: "Invalid CSRF token"})
		return
	}

	session.Delete(SESSION_KEY_OIDC_STATE)
	err := session.Save()
	if err != nil {
		log.Println("Unable to CompleteOIDC, session.Save fails:", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "Sorry, we are currently unable to respond to your request. Please retry later."})
		return
	}

	oauth2Token, err := p.oauth2config.Exchange(c, c.Query("code"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: fmt.Sprintf("Failed to exchange token: %s", err.Error())})
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: "No id_token field in oauth2 token."})
		return
	}

	idToken, err := p.oidcVerifier.Verify(c, rawIDToken)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: fmt.Sprintf("Failed to verify ID Token: %s", err.Error())})
		return
	}

	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: fmt.Sprintf("Unable to retrieve user profile: %s", err.Error())})
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

		hash := sha1.Sum([]byte(profile.Email))
		profile.Id = hash[:]
	}

	_, err = p.authService.CompleteAuthentication(&profile)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: fmt.Sprintf("Unable to complete authentication: %s", err.Error())})
		return
	}

	middleware.SessionLoginOK(c, &profile)

	c.Redirect(http.StatusFound, p.config.GetBaseURL()+"/")
}

// HasOidc checks if OpenID Connect authentication is available and returns the provider name.
func (s *Server) HasOidc(ctx context.Context, request HasOidcRequestObject) (HasOidcResponseObject, error) {
	if len(s.cfg.OIDCClients) > 0 {
		providerURL := s.cfg.OIDCClients[0].ProviderURL
		parts := strings.Split(strings.TrimSuffix(providerURL.Host, "."), ".")
		var provider string
		if len(parts) > 2 {
			provider = strings.Join(parts[len(parts)-2:], ".")
		} else {
			provider = strings.Join(parts, ".")
		}
		return HasOidc200JSONResponse{
			Provider: &provider,
		}, nil
	}

	return HasOidc404JSONResponse(happydns.ErrorResponse{
		Message: "OIDC is not configured",
	}), nil
}

// RedirectOidc initiates the OpenID Connect authentication flow.
func (s *Server) RedirectOidc(ctx context.Context, request RedirectOidcRequestObject) (RedirectOidcResponseObject, error) {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return RedirectOidc400JSONResponse(happydns.ErrorResponse{
			Message: "Unable to extract context",
		}), nil
	}

	if len(s.cfg.OIDCClients) == 0 {
		return RedirectOidc400JSONResponse(happydns.ErrorResponse{
			Message: "OIDC is not configured",
		}), nil
	}

	oidcProvider := NewOIDCProvider(s.cfg, s.dependancies.AuthenticationUsecase())
	oidcProvider.RedirectOIDC(ginCtx)

	// The redirect is handled by RedirectOIDC, return nil response
	return nil, nil
}

// CompleteOidc completes the OpenID Connect authentication flow.
func (s *Server) CompleteOidc(ctx context.Context, request CompleteOidcRequestObject) (CompleteOidcResponseObject, error) {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return CompleteOidc400JSONResponse(happydns.ErrorResponse{
			Message: "Unable to extract context",
		}), nil
	}

	if len(s.cfg.OIDCClients) == 0 {
		return CompleteOidc400JSONResponse(happydns.ErrorResponse{
			Message: "OIDC is not configured",
		}), nil
	}

	oidcProvider := NewOIDCProvider(s.cfg, s.dependancies.AuthenticationUsecase())
	oidcProvider.CompleteOIDC(ginCtx)

	// The redirect is handled by CompleteOIDC, return nil response
	return nil, nil
}
