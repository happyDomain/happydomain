// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

type ProviderSuggestionController struct {
	suggestionService happydns.ProviderSuggestionUsecase
}

func NewProviderSuggestionController(suggestionService happydns.ProviderSuggestionUsecase) *ProviderSuggestionController {
	return &ProviderSuggestionController{
		suggestionService: suggestionService,
	}
}

// SuggestProviders looks for the providers hosting a domain.
//
//	@Summary	Suggest the providers hosting a domain.
//	@Schemes
//	@Description	Look up the name servers of the given domain, and return the types of the providers they belong to. The lists are empty when the domain or its provider cannot be found.
//	@Tags			providers
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			domain	query		string	true	"Domain name"
//	@Success		200		{object}	happydns.ProviderSuggestion
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid domain name"
//	@Failure		401		{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		429		{object}	happydns.ErrorResponse	"Too many requests"
//	@Router			/providers/_suggestions [get]
func (psc *ProviderSuggestionController) SuggestProviders(c *gin.Context) {
	suggestion, err := psc.suggestionService.SuggestProviders(c.Request.Context(), c.Query("domain"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, suggestion)
}
