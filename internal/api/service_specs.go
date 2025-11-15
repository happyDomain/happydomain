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

// GetServiceSpecs returns the static list of usable services in this happyDomain release.
func (s *Server) GetServiceSpecs(ctx context.Context, request GetServiceSpecsRequestObject) (GetServiceSpecsResponseObject, error) {
	services := s.dependancies.ServiceSpecsUsecase().ListServices()
	return GetServiceSpecs200JSONResponse(services), nil
}

// GetServiceSpec returns a description of the expected fields.
func (s *Server) GetServiceSpec(ctx context.Context, request GetServiceSpecRequestObject) (GetServiceSpecResponseObject, error) {
	// TODO: Get service type from request.ServiceType string
	// Currently the controller uses reflection to get the type, we'll need to adapt this
	// svctype := c.MustGet("servicetype").(reflect.Type)

	// specs, err := s.dependancies.ServiceSpecsUsecase().GetServiceSpecs(svctype)
	// if err != nil {
	// 	return GetServiceSpec404JSONResponse{ErrorResponse: happydns.ErrorResponse{
	// 		Message: fmt.Sprintf("Service type does not exist: %s", err.Error()),
	// 	}}, nil
	// }

	// return GetServiceSpec200JSONResponse(specs), nil

	return GetServiceSpec404JSONResponse(happydns.ErrorResponse{
		Message: "Not implemented yet - needs reflection support",
	}), nil
}

// GetServiceIcon returns the icon as image/png.
func (s *Server) GetServiceIcon(ctx context.Context, request GetServiceIconRequestObject) (GetServiceIconResponseObject, error) {
	cnt, err := s.dependancies.ServiceSpecsUsecase().GetServiceIcon(request.ServiceType)
	if err != nil {
		return GetServiceIcon404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Service icon not found: %s", err.Error()),
		}), nil
	}

	return GetServiceIcon200ImagepngResponse{
		Body:          io.NopCloser(bytes.NewReader(cnt)),
		ContentLength: int64(len(cnt)),
	}, nil
}
