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

package domaininfo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"git.happydns.org/happyDomain/model"
)

// GetDomainInfo tries RDAP first, then falls back to WHOIS. It strips
// any trailing dot from the domain and short-circuits on
// ErrDomainDoesNotExist.
func GetDomainInfo(ctx context.Context, fqdn happydns.Origin) (*happydns.DomainInfo, error) {
	domain := happydns.Origin(strings.TrimSuffix(string(fqdn), "."))

	info, err := GetDomainRDAPInfo(ctx, domain)
	if err == nil {
		return info, nil
	}
	if errors.Is(err, happydns.ErrDomainDoesNotExist) {
		return nil, err
	}

	info, err = GetDomainWhoisInfo(ctx, domain)
	if err == nil {
		return info, nil
	}
	if errors.Is(err, happydns.ErrDomainDoesNotExist) {
		return nil, err
	}

	return nil, fmt.Errorf("unable to retrieve RDAP/WHOIS info: %w", err)
}

// sanitizeURL returns a pointer to the URL string only if it uses http or
// https. Any other scheme (javascript:, data:, etc.) or malformed URL yields
// nil so it is never exposed to the frontend.
func sanitizeURL(raw string) *string {
	u, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil
	}
	return &raw
}
