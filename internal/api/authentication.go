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

package api

import (
	"context"
	"fmt"
	"log"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	middleware "github.com/oapi-codegen/gin-middleware"

	hdmiddleware "git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

func goAuthenticate(authService happydns.AuthenticationUsecase, c *gin.Context) (*happydns.User, error) {
	session := sessions.Default(c)

	var userid happydns.Identifier
	if iu, ok := session.Get("iduser").(happydns.Identifier); ok && len(iu) > 0 {
		userid = iu
	} else {
		return nil, fmt.Errorf("invalid session")
	}

	user, err := authService.CompleteAuthentication(&happydns.UserProfile{
		UserId: userid,
	})
	if err != nil {
		log.Printf("%s: Unable to validate session authentication: %s", c.ClientIP(), err.Error())
		return nil, fmt.Errorf("invalid session")
	}

	c.Set("AuthMethod", "session")
	c.Set("LoggedUser", user)

	return user, nil
}

// Authenticate uses the specified validator to ensure the cookie is valid.
func Authenticate(authService happydns.AuthenticationUsecase, ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	// Our security scheme is named cookieAuth, ensure this is the case
	if input.SecuritySchemeName != "ApiKeyAuth" {
		return fmt.Errorf("security scheme %s != 'ApiKeyAuth'", input.SecuritySchemeName)
	}

	c := ctx.Value(middleware.GinContextKey).(*gin.Context)

	_, err := goAuthenticate(authService, c)
	if err != nil {
		return err
	}

	return nil
}

func NewAuthenticator(authService happydns.AuthenticationUsecase) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		return Authenticate(authService, ctx, input)
	}
}

func CreateAuthMiddleware(authService happydns.AuthenticationUsecase) (gin.HandlerFunc, error) {
	spec, err := GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("loading spec: %w", err)
	}

	// Clear out the servers array in the swagger spec, that skips validating
	// that server names match. We don't know how this thing will be run.
	spec.Servers = nil

	validator := middleware.OapiRequestValidatorWithOptions(spec,
		&middleware.Options{
			Options: openapi3filter.Options{
				AuthenticationFunc: NewAuthenticator(authService),
			},
		})

	return validator, nil
}

// GetLoggedUser retrieves information about the currently authenticated user.
func (s *Server) GetLoggedUser(ctx context.Context, request GetLoggedUserRequestObject) (GetLoggedUserResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return GetLoggedUser401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	return GetLoggedUser200JSONResponse(*user), nil
}

// Login authenticates a user with email and password credentials.
func (s *Server) Login(ctx context.Context, request LoginRequestObject) (LoginResponseObject, error) {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return Login401JSONResponse(happydns.ErrorResponse{
			Message: "Unable to extract context",
		}), nil
	}

	if request.Body == nil {
		log.Printf("%s sends invalid LoginForm JSON: missing body", ginCtx.ClientIP())
		return Login400JSONResponse(happydns.ErrorResponse{
			Message: "Something is wrong in received data: missing body",
		}), nil
	}

	loginRequest := happydns.LoginRequest{
		Email:    request.Body.Email,
		Password: request.Body.Password,
	}

	user, err := s.dependancies.AuthenticationUsecase().AuthenticateUserWithPassword(loginRequest)
	if err != nil {
		log.Printf("%s: %s", ginCtx.ClientIP(), err.Error())
		return Login401JSONResponse(happydns.ErrorResponse{
			Message: "Invalid username or password.",
		}), nil
	}

	hdmiddleware.SessionLoginOK(ginCtx, user)

	return Login200JSONResponse(*user), nil
}

// Logout ends the current user session.
func (s *Server) Logout(ctx context.Context, request LogoutRequestObject) (LogoutResponseObject, error) {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return Logout500JSONResponse(happydns.ErrorResponse{
			Message: "Unable to extract context",
		}), nil
	}

	session := sessions.Default(ginCtx)

	session.Clear()
	err := session.Save()
	if err != nil {
		return Logout500JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	return Logout204Response{}, nil
}
