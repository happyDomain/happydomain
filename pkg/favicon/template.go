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

package favicon

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// templateSource asks a third-party icon service, by substituting the host
// into a URL template. It never touches the site itself.
//
// Compared to the direct source it costs one request instead of two, needs no
// HTML parsing, and cannot be pointed at an arbitrary host: the destination is
// fixed by the operator and only the path or query part varies. What it buys in
// simplicity it pays in coverage (the service only knows what it crawled) and
// in confidentiality towards that service, which learns which domains this
// instance looks up.
type templateSource struct {
	name     string
	template string
	client   *http.Client
}

func newTemplateSource(name, template string, client *http.Client) *templateSource {
	return &templateSource{
		name:     name,
		template: template,
		client:   client,
	}
}

func (ts *templateSource) Name() string {
	return ts.name
}

func (ts *templateSource) FetchIcon(site *url.URL) ([]byte, string, error) {
	host := site.Hostname()
	if host == "" {
		return nil, "", fmt.Errorf("no host to look up")
	}

	// The host is the only thing interpolated, and the template usually places
	// it in a path or a query. Rather than guessing which escaping applies,
	// refuse anything that is not already a plain hostname: nothing else can
	// reshape the request the operator described.
	if strings.TrimFunc(host, isHostByte) != "" {
		return nil, "", fmt.Errorf("%s: refusing to look up %q", ts.name, host)
	}

	iconURL := strings.ReplaceAll(ts.template, templatePlaceholder, host)

	iconBytes, contentType, err := downloadIcon(ts.client, iconURL)
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", ts.name, err)
	}

	return iconBytes, contentType, nil
}

// isHostByte reports whether r may appear in a hostname, punycoded IDN
// included.
func isHostByte(r rune) bool {
	return r == '.' || r == '-' || r == '_' ||
		(r >= '0' && r <= '9') ||
		(r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z')
}
