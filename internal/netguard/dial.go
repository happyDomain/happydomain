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
	"net"
	"net/http"
	"syscall"
	"time"
)

const (
	// dialTimeout bounds a single connection attempt.
	dialTimeout = 10 * time.Second

	// maxRedirects is how many hops an HTTPClient follows before giving up.
	// Each one is re-validated, so this only bounds the work.
	maxRedirects = 5
)

// DialContext resolves the destination, checks every address it resolves to,
// then dials one of those addresses directly.
//
// Pinning the dial to an already-checked IP literal is the point: letting the
// dialer resolve the name a second time would reopen the rebinding window that
// checking it in the first place was meant to close.
func (g *Guard) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	addrs, err := g.ResolveAllowed(ctx, host)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: dialTimeout}

	var lastErr error
	for _, addr := range addrs {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
		if err == nil {
			return conn, nil
		}

		lastErr = err
		if errors.Is(err, syscall.ECONNREFUSED) {
			continue
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no usable address to reach %q", host)
}

// HTTPClient returns a client that only ever reaches allowed destinations,
// including across redirects.
func (g *Guard) HTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:           g.DialContext,
			ResponseHeaderTimeout: timeout,
			IdleConnTimeout:       30 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}

			// The dial is guarded too, so this is belt and braces: it rejects
			// a redirect to a non-http(s) scheme, which never reaches
			// DialContext at all.
			_, err := g.ValidateURL(req.Context(), req.URL.String())
			return err
		},
	}
}
