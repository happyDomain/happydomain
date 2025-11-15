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

//go:build nooidc

package api

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

type OIDCProvider struct{}

func NewOIDCProvider(cfg *happydns.Options, authService happydns.AuthenticationUsecase) *OIDCProvider {
	log.Fatal("OIDCProviderURL is defined whereas happydomain is compiled without OIDC support.")
	return nil
}

func (p *OIDCProvider) RedirectOIDC(c *gin.Context) {
	c.Status(http.StatusInternalServerError)
}

func (p *OIDCProvider) CompleteOIDC(c *gin.Context) {
	c.Status(http.StatusInternalServerError)
}

// HasOidc checks if OpenID Connect authentication is available and returns the provider name.
func (s *Server) HasOidc(ctx context.Context, request HasOidcRequestObject) (HasOidcResponseObject, error) {
	// OIDC is not compiled in
	return HasOidc404JSONResponse(happydns.ErrorResponse{
		Message: "OIDC is not configured",
	}), nil
}

// RedirectOidc initiates the OpenID Connect authentication flow.
func (s *Server) RedirectOidc(ctx context.Context, request RedirectOidcRequestObject) (RedirectOidcResponseObject, error) {
	return RedirectOidc500JSONResponse(happydns.ErrorResponse{
		Message: "OIDC is not supported in this build",
	}), nil
}

// CompleteOidc completes the OpenID Connect authentication flow.
func (s *Server) CompleteOidc(ctx context.Context, request CompleteOidcRequestObject) (CompleteOidcResponseObject, error) {
	return CompleteOidc500JSONResponse(happydns.ErrorResponse{
		Message: "OIDC is not supported in this build",
	}), nil
}
