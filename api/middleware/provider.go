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

package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

func ProviderMetaHandler(providerService happydns.ProviderUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract provider ID
		pid, err := happydns.NewIdentifierFromString(string(c.Param("pid")))
		if err != nil {
			ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid provider id: %w", err))
			return
		}

		// Get a valid user
		user := MyUser(c)
		if user == nil {
			ErrorResponse(c, http.StatusUnauthorized, fmt.Errorf("user not defined"))
			return
		}

		// Retrieve provider meta
		providermeta, err := providerService.GetUserProviderMeta(user, pid)
		if err != nil {
			ErrorResponse(c, http.StatusNotFound, fmt.Errorf("provider not found"))
			return
		}

		// Continue
		c.Set("providermeta", providermeta)

		c.Next()
	}
}

func ProviderHandler(providerService happydns.ProviderUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract provider ID
		pid, err := happydns.NewIdentifierFromString(string(c.Param("pid")))
		if err != nil {
			ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid provider id: %w", err))
			return
		}

		// Get a valid user
		user := MyUser(c)
		if user == nil {
			ErrorResponse(c, http.StatusUnauthorized, fmt.Errorf("user not defined"))
			return
		}

		// Retrieve provider
		provider, err := providerService.GetUserProvider(user, pid)
		if err != nil {
			ErrorResponse(c, http.StatusNotFound, fmt.Errorf("provider not found"))
			return
		}

		// Continue
		c.Set("provider", provider)
		c.Set("providermeta", provider.ProviderMeta)

		c.Next()
	}
}
