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
	"bytes"
	"context"
	"fmt"
	"io"

	"git.happydns.org/happyDomain/model"
)

// GetProviderSpecs returns the static list of usable providers in this happyDomain release.
func (s *Server) GetProviderSpecs(ctx context.Context, request GetProviderSpecsRequestObject) (GetProviderSpecsResponseObject, error) {
	providers := s.dependancies.ProviderSpecsUsecase().ListProviders()
	return GetProviderSpecs200JSONResponse(providers), nil
}

// GetProviderSpec returns a description of the expected settings and the provider capabilities.
func (s *Server) GetProviderSpec(ctx context.Context, request GetProviderSpecRequestObject) (GetProviderSpecResponseObject, error) {
	specs, err := s.dependancies.ProviderSpecsUsecase().GetProviderSpecs(request.ProviderType)
	if err != nil {
		return GetProviderSpec404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Provider type does not exist: %s", err.Error()),
		}), nil
	}

	return GetProviderSpec200JSONResponse(*specs), nil
}

// GetProviderIcon returns the icon as image/png.
func (s *Server) GetProviderIcon(ctx context.Context, request GetProviderIconRequestObject) (GetProviderIconResponseObject, error) {
	cnt, err := s.dependancies.ProviderSpecsUsecase().GetProviderIcon(request.ProviderType)
	if err != nil {
		return GetProviderIcon404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Provider icon not found: %s", err.Error()),
		}), nil
	}

	return GetProviderIcon200ImagepngResponse{
		Body:          io.NopCloser(bytes.NewReader(cnt)),
		ContentLength: int64(len(cnt)),
	}, nil
}

// CreateProviderFromSettings creates or updates a Provider with human fillable forms.
func (s *Server) CreateProviderFromSettings(ctx context.Context, request CreateProviderFromSettingsRequestObject) (CreateProviderFromSettingsResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return CreateProviderFromSettings401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	var settingsState happydns.ProviderSettingsState
	settingsState = *request.Body

	provider, form, err := s.dependancies.ProviderSettingsUsecase().NextProviderSettingsState(&settingsState, request.ProviderType, user)
	if err != nil {
		return CreateProviderFromSettings400JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to process settings: %s", err.Error()),
		}), nil
	}

	if provider != nil {
		return CreateProviderFromSettings200JSONResponse(*provider), nil
	} else {
		return CreateProviderFromSettings202JSONResponse(*form), nil
	}
}
