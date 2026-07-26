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

package netguard

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ValidateURLShape checks that rawURL is an http or https URL with a host. It
// applies no policy, so it is safe to call from a context that has no Guard,
// such as a config object validating its own fields.
func ValidateURLShape(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q (only http and https are allowed)", u.Scheme)
	}
	if u.Hostname() == "" {
		return nil, errors.New("URL has no host")
	}
	return u, nil
}

// ValidateURL checks the shape of rawURL and that its host resolves entirely
// within the allowed destinations.
func (g *Guard) ValidateURL(ctx context.Context, rawURL string) (*url.URL, error) {
	u, err := ValidateURLShape(rawURL)
	if err != nil {
		return nil, err
	}

	if _, err := g.ResolveAllowed(ctx, u.Hostname()); err != nil {
		return nil, err
	}

	return u, nil
}

// HostFromEndpoint extracts the host out of a destination a user typed into a
// provider form.
//
// Those forms accept three shapes, sometimes within the same field: a full URL
// ("https://pdns.example.com:8081/api"), a bare authority ("192.168.1.1:8080"),
// and a bare host or IP literal. providers/openwrt.go even tells the user that
// "http:// is assumed if no scheme is given". Being lenient here is deliberate:
// a value we fail to parse is a value we would fail to check.
func HostFromEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", errors.New("empty endpoint")
	}

	if strings.Contains(endpoint, "://") {
		u, err := url.Parse(endpoint)
		if err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}
		if u.Hostname() == "" {
			return "", errors.New("URL has no host")
		}
		return u.Hostname(), nil
	}

	// Strip anything a scheme-less URL may still carry after the authority.
	if idx := strings.IndexAny(endpoint, "/?#"); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	// And any credentials before it.
	if idx := strings.LastIndex(endpoint, "@"); idx >= 0 {
		endpoint = endpoint[idx+1:]
	}

	host, _ := splitHostPort(endpoint, 0)
	if host == "" {
		return "", errors.New("no host in endpoint")
	}

	return host, nil
}
