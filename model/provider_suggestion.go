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

package happydns

import (
	"context"
)

// EnableProviderSuggestion tells whether the name servers of a domain being
// added are looked up, to suggest the provider hosting it.
const EnableProviderSuggestion = true

// ProviderSuggestion tells which providers seem to host a domain.
type ProviderSuggestion struct {
	// Nameservers are the name servers of the domain, as publicly answered.
	Nameservers []string `json:"nameservers" binding:"required"`

	// Providers are the types of the providers these name servers belong to.
	// Several providers may share their name servers, and a domain may be
	// served by several providers.
	Providers []string `json:"providers" binding:"required"`
}

type ProviderSuggestionUsecase interface {
	// SuggestProviders looks for the providers hosting the domain. Being
	// unable to find any is not an error: the suggestion is then empty.
	SuggestProviders(ctx context.Context, domain string) (*ProviderSuggestion, error)
}
