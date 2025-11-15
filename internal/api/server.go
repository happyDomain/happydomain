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

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

type Server struct {
	cfg          *happydns.Options
	dependancies happydns.UsecaseDependancies
}

func NewServer(cfg *happydns.Options, dependancies happydns.UsecaseDependancies) StrictServerInterface {
	return &Server{
		cfg:          cfg,
		dependancies: dependancies,
	}
}

// GetUserFromContext extracts the logged-in user from the gin context.
// Returns the gin.Context, the user, and an error if extraction fails.
func (s *Server) GetUserFromContext(ctx context.Context) (*gin.Context, *happydns.User, error) {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return nil, nil, fmt.Errorf("unable to extract context")
	}

	user, exists := ginCtx.Get("LoggedUser")
	if !exists {
		return ginCtx, nil, fmt.Errorf("user not found in context")
	}

	return ginCtx, user.(*happydns.User), nil
}

// ParseDomainId parses and validates a domain ID string from a request.
// Returns the parsed identifier or an error.
func (s *Server) ParseDomainId(domainIdStr string) (happydns.Identifier, error) {
	domainId, err := happydns.NewIdentifierFromString(domainIdStr)
	if err != nil {
		return nil, fmt.Errorf("Invalid domain ID: %s", err.Error())
	}
	return domainId, nil
}

// ParseZoneId parses and validates a zone ID string from a request.
// Returns the parsed identifier or an error.
func (s *Server) ParseZoneId(zoneIdStr string) (happydns.Identifier, error) {
	zoneId, err := happydns.NewIdentifierFromString(zoneIdStr)
	if err != nil {
		return nil, fmt.Errorf("Invalid zone ID: %s", err.Error())
	}
	return zoneId, nil
}

// ParseProviderId parses and validates a provider ID string from a request.
// Returns the parsed identifier or an error.
func (s *Server) ParseProviderId(providerIdStr string) (happydns.Identifier, error) {
	providerId, err := happydns.NewIdentifierFromString(providerIdStr)
	if err != nil {
		return nil, fmt.Errorf("Invalid provider ID: %s", err.Error())
	}
	return providerId, nil
}

// GetUserDomainById retrieves a domain for the given user by parsing and fetching in one step.
func (s *Server) GetUserDomainById(user *happydns.User, domainIdStr string) (*happydns.Domain, error) {
	domainId, err := s.ParseDomainId(domainIdStr)
	if err != nil {
		return nil, err
	}

	domain, err := s.dependancies.DomainUsecase().GetUserDomain(user, domainId)
	if err != nil {
		return nil, fmt.Errorf("Domain not found: %s", err.Error())
	}
	return domain, nil
}

// GetZoneById retrieves a zone by parsing and fetching in one step.
func (s *Server) GetZoneById(zoneIdStr string) (*happydns.Zone, error) {
	zoneId, err := s.ParseZoneId(zoneIdStr)
	if err != nil {
		return nil, err
	}

	zone, err := s.dependancies.ZoneUsecase().GetZone(zoneId)
	if err != nil {
		return nil, fmt.Errorf("Zone not found: %s", err.Error())
	}
	return zone, nil
}

// GetUserDomainAndZone retrieves both domain and zone in one step.
func (s *Server) GetUserDomainAndZone(user *happydns.User, domainIdStr, zoneIdStr string) (*happydns.Domain, *happydns.Zone, error) {
	domain, err := s.GetUserDomainById(user, domainIdStr)
	if err != nil {
		return nil, nil, err
	}

	zone, err := s.GetZoneById(zoneIdStr)
	if err != nil {
		return nil, nil, err
	}

	return domain, zone, nil
}

// GetUserProviderById retrieves a provider for the given user by parsing and fetching in one step.
func (s *Server) GetUserProviderById(user *happydns.User, providerIdStr string) (*happydns.Provider, error) {
	providerId, err := s.ParseProviderId(providerIdStr)
	if err != nil {
		return nil, err
	}

	provider, err := s.dependancies.ProviderUsecase(false).GetUserProvider(user, providerId)
	if err != nil {
		return nil, fmt.Errorf("Provider not found: %s", err.Error())
	}
	return provider, nil
}
