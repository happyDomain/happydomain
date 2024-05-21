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

package app

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"git.happydns.org/happyDomain/api"
	"git.happydns.org/happyDomain/config"
)

const (
	SESSION_KEY_OIDC_STATE = "oidc-state"
)

func InitializeOIDC(cfg *config.Options, router *gin.RouterGroup) {
	// Initialize OIDC
	provider, err := cfg.GetOIDCProvider(context.Background())
	if err != nil {
		log.Fatal("Unable to instantiate OIDC Provider:", err)
	}

	oauth2Config := cfg.GetOAuth2Config(provider)

	oidcVerifier := provider.Verifier(&oidc.Config{
		ClientID: config.OIDCClientID,
	})

	declareOidcRoutes(cfg, router, oauth2Config, oidcVerifier)
}

func declareOidcRoutes(cfg *config.Options, router *gin.RouterGroup, oauth2config *oauth2.Config, oidcVerifier *oidc.IDTokenVerifier) {
	providerurl, _ := url.Parse(config.OIDCProviderURL)
	router.GET("has_oidc", func(c *gin.Context) {
		parts := strings.Split(strings.TrimSuffix(providerurl.Host, "."), ".")
		if len(parts) > 2 {
			c.JSON(http.StatusOK, gin.H{"provider": strings.Join(parts[len(parts)-2:len(parts)], ".")})
		} else {
			c.JSON(http.StatusOK, gin.H{"provider": strings.Join(parts, ".")})
		}
	})
	router.GET("oidc", func(c *gin.Context) {
		redirect_OIDC(cfg, oauth2config, c)
	})
	router.GET("callback", func(c *gin.Context) {
		complete_OIDC(cfg, oauth2config, oidcVerifier, c)
	})
}

func redirect_OIDC(cfg *config.Options, oauth2Config *oauth2.Config, c *gin.Context) {
	session := sessions.Default(c)

	state := make([]byte, 32)
	_, err := rand.Read(state)
	if err != nil {
		log.Println("Unable to redirect_OIDC, rand.Read fails:", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to respond to your request. Please retry later."})
		return
	}

	session.Set(SESSION_KEY_OIDC_STATE, hex.EncodeToString(state))
	err = session.Save()

	if err != nil {
		log.Println("Unable to redirect_OIDC, session.Save fails:", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to respond to your request. Please retry later."})
		return
	}

	c.Redirect(http.StatusFound, oauth2Config.AuthCodeURL(hex.EncodeToString(state)))
}

func complete_OIDC(cfg *config.Options, oauth2Config *oauth2.Config, oidcVerifier *oidc.IDTokenVerifier, c *gin.Context) {
	session := sessions.Default(c)

	state := c.Query("state")

	if state != session.Get(SESSION_KEY_OIDC_STATE) {
		log.Printf("Invalid CSRF token on /auth/callback: got %q, expected %q", state, session.Get(SESSION_KEY_OIDC_STATE))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Invalid CSRF token"})
		return
	}

	oauth2Token, err := oauth2Config.Exchange(c, c.Query("code"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": fmt.Sprintf("Failed to exchange token: %s", err.Error())})
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "No id_token field in oauth2 token."})
		return
	}

	idToken, err := oidcVerifier.Verify(c, rawIDToken)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": fmt.Sprintf("Failed to verify ID Token: %s", err.Error())})
		return
	}

	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": fmt.Sprintf("Unable to retrieve user profile: %s", err.Error())})
		return
	}

	var profile api.UserProfile

	if email, ok := claims["email"].(string); ok {
		profile.Email = email
	}
	if email_verified, ok := claims["email_verified"].(bool); ok {
		profile.EmailVerified = email_verified
	}

	if len(profile.UserId) == 0 {
		if len(profile.Email) == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "No email nor user identifier found in OIDC profile."})
			return
		}

		hash := sha1.Sum([]byte(profile.Email))
		profile.UserId = hash[:]
	}

	_, err = api.CompleteAuth(cfg, c, profile)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": fmt.Sprintf("Unable to complete authentication: %s", err.Error())})
		return
	}

	c.Redirect(http.StatusFound, cfg.BaseURL+"/")
}
