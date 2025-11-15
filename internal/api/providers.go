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

	"git.happydns.org/happyDomain/model"
)

// GetProviders retrieves all providers belonging to the user.
func (s *Server) GetProviders(ctx context.Context, request GetProvidersRequestObject) (GetProvidersResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return GetProviders401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	providers, err := s.dependancies.ProviderUsecase(false).ListUserProviders(user)
	if err != nil {
		return GetProviders404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Unable to retrieve providers: %s", err.Error()),
		}), nil
	}

	var result []happydns.ProviderMeta
	for _, p := range providers {
		result = append(result, *p)
	}

	return GetProviders200JSONResponse(result), nil
}

// CreateProvider appends a new provider for the user.
func (s *Server) CreateProvider(ctx context.Context, request CreateProviderRequestObject) (CreateProviderResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return CreateProvider401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	var providerMessage happydns.ProviderMessage
	// TODO: Map request body to providerMessage - schema needs checking

	provider, err := s.dependancies.ProviderUsecase(false).CreateProvider(user, &providerMessage)
	if err != nil {
		return CreateProvider500JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Unable to create provider: %s", err.Error()),
		}), nil
	}

	return CreateProvider200JSONResponse(*provider), nil
}

// GetProvider retrieves information about a given Provider owned by the user.
func (s *Server) GetProvider(ctx context.Context, request GetProviderRequestObject) (GetProviderResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return GetProvider401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	provider, err := s.GetUserProviderById(user, request.ProviderId)
	if err != nil {
		return GetProvider404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	return GetProvider200JSONResponse(*provider), nil
}

// UpdateProvider updates the information about a given Provider owned by the user.
func (s *Server) UpdateProvider(ctx context.Context, request UpdateProviderRequestObject) (UpdateProviderResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return UpdateProvider401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	provider, err := s.GetUserProviderById(user, request.ProviderId)
	if err != nil {
		return UpdateProvider404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	var providerMessage happydns.ProviderMessage
	// TODO: Map request.Body to providerMessage properly

	err = s.dependancies.ProviderUsecase(false).UpdateProviderFromMessage(provider.UnderscoreId, user, &providerMessage)
	if err != nil {
		return UpdateProvider500JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Unable to update provider: %s", err.Error()),
		}), nil
	}

	return UpdateProvider200JSONResponse(*provider), nil
}

// DeleteProvider removes a provider from the database.
func (s *Server) DeleteProvider(ctx context.Context, request DeleteProviderRequestObject) (DeleteProviderResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return DeleteProvider401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	provider, err := s.GetUserProviderById(user, request.ProviderId)
	if err != nil {
		return DeleteProvider400JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	err = s.dependancies.ProviderUsecase(false).DeleteProvider(user, provider.UnderscoreId)
	if err != nil {
		return DeleteProvider500JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Unable to delete provider: %s", err.Error()),
		}), nil
	}

	return DeleteProvider204Response{}, nil
}

// ListProviderDomains lists domains available from the given Provider.
func (s *Server) ListProviderDomains(ctx context.Context, request ListProviderDomainsRequestObject) (ListProviderDomainsResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return ListProviderDomains401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	provider, err := s.GetUserProviderById(user, request.ProviderId)
	if err != nil {
		return ListProviderDomains404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domains, err := s.dependancies.ProviderUsecase(false).ListHostedDomains(provider)
	if err != nil {
		return ListProviderDomains400JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Unable to list domains: %s", err.Error()),
		}), nil
	}

	_ = domains // Suppress unused warning for now

	return ListProviderDomains200JSONResponse(*provider), nil
}

// GetProviderDomain retrieves or creates a domain on the given Provider.
func (s *Server) GetProviderDomain(ctx context.Context, request GetProviderDomainRequestObject) (GetProviderDomainResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return GetProviderDomain401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	provider, err := s.GetUserProviderById(user, request.ProviderId)
	if err != nil {
		return GetProviderDomain404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	err = s.dependancies.ProviderUsecase(false).CreateDomainOnProvider(provider, request.Fqdn)
	if err != nil {
		return GetProviderDomain400JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Unable to create domain on provider: %s", err.Error()),
		}), nil
	}

	return GetProviderDomain200JSONResponse(*provider), nil
}
