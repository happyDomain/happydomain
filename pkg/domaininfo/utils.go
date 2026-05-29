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

	"golang.org/x/sync/singleflight"

	"git.happydns.org/happyDomain/model"
)

// lookupGroup deduplicates concurrent GetDomainInfo calls for the same
// domain, so overlapping checks (e.g. a manual recheck racing the scheduled
// one) issue a single upstream RDAP/WHOIS lookup instead of one each.
var lookupGroup singleflight.Group

// GetDomainInfo tries RDAP first, then falls back to WHOIS. It strips
// any trailing dot from the domain and short-circuits on
// ErrDomainDoesNotExist. Concurrent calls for the same domain are
// coalesced into a single upstream lookup: the shared lookup runs
// detached from any single caller's context (so one caller cancelling
// doesn't abort the others), while each caller still stops waiting as
// soon as its own context is done.
func GetDomainInfo(ctx context.Context, fqdn happydns.Origin) (*happydns.DomainInfo, error) {
	domain := happydns.Origin(strings.TrimSuffix(string(fqdn), "."))

	resCh := lookupGroup.DoChan(string(domain), func() (any, error) {
		return getDomainInfo(context.WithoutCancel(ctx), domain)
	})

	select {
	case res := <-resCh:
		if res.Err != nil {
			return nil, res.Err
		}
		return res.Val.(*happydns.DomainInfo), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func getDomainInfo(ctx context.Context, domain happydns.Origin) (*happydns.DomainInfo, error) {
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
