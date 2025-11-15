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

// GetDomains retrieves all domains belonging to the user.
func (s *Server) GetDomains(ctx context.Context, request GetDomainsRequestObject) (GetDomainsResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return GetDomains401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domains, err := s.dependancies.DomainUsecase().ListUserDomains(user)
	if err != nil {
		return GetDomains404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Unable to retrieve domains: %s", err.Error()),
		}), nil
	}

	var result []happydns.Domain
	for _, d := range domains {
		result = append(result, *d)
	}

	return GetDomains200JSONResponse(result), nil
}

// CreateDomain appends a new domain to those managed.
func (s *Server) CreateDomain(ctx context.Context, request CreateDomainRequestObject) (CreateDomainResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return CreateDomain401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	var domain happydns.Domain
	domain.Domain = request.Body.Domain
	domain.IdProvider = request.Body.IdProvider

	err = s.dependancies.DomainUsecase().CreateDomain(user, &domain)
	if err != nil {
		return CreateDomain500JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Unable to create domain: %s", err.Error()),
		}), nil
	}

	return CreateDomain200JSONResponse(domain), nil
}

// GetDomain retrieves information about a given Domain owned by the user.
func (s *Server) GetDomain(ctx context.Context, request GetDomainRequestObject) (GetDomainResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return GetDomain401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domain, err := s.GetUserDomainById(user, request.DomainId)
	if err != nil {
		return GetDomain404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domainExtended, err := s.dependancies.DomainUsecase().ExtendsDomainWithZoneMeta(domain)
	if err != nil {
		return GetDomain404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Unable to extend domain: %s", err.Error()),
		}), nil
	}

	return GetDomain200JSONResponse(*domainExtended), nil
}

// UpdateDomain updates the information about a given Domain owned by the user.
func (s *Server) UpdateDomain(ctx context.Context, request UpdateDomainRequestObject) (UpdateDomainResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return UpdateDomain401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domain, err := s.GetUserDomainById(user, request.DomainId)
	if err != nil {
		return UpdateDomain404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	err = s.dependancies.DomainUsecase().UpdateDomain(domain.Id, user, func(d *happydns.Domain) {
		d.Group = request.Body.Group
	})
	if err != nil {
		return UpdateDomain500JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Unable to update domain: %s", err.Error()),
		}), nil
	}

	return UpdateDomain200JSONResponse(*domain), nil
}

// DeleteDomain removes a domain from the database.
func (s *Server) DeleteDomain(ctx context.Context, request DeleteDomainRequestObject) (DeleteDomainResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return DeleteDomain401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domain, err := s.GetUserDomainById(user, request.DomainId)
	if err != nil {
		return DeleteDomain400JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	err = s.dependancies.DomainUsecase().DeleteDomain(domain.Id)
	if err != nil {
		return DeleteDomain500JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Unable to delete domain: %s", err.Error()),
		}), nil
	}

	return DeleteDomain204Response{}, nil
}

// GetDomainLogs retrieves information about the actions performed on the domain.
func (s *Server) GetDomainLogs(ctx context.Context, request GetDomainLogsRequestObject) (GetDomainLogsResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return GetDomainLogs401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domain, err := s.GetUserDomainById(user, request.DomainId)
	if err != nil {
		return GetDomainLogs404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	logs, err := s.dependancies.DomainLogUsecase().ListDomainLogs(domain)
	if err != nil {
		return GetDomainLogs404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Unable to retrieve logs: %s", err.Error()),
		}), nil
	}

	var result []happydns.DomainLog
	for _, log := range logs {
		result = append(result, *log)
	}

	return GetDomainLogs200JSONResponse(result), nil
}
