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

// A Source knows one way of turning a site into an icon. They are tried in the
// order the operator listed them, so that an instance can prefer asking the
// site itself and only fall back on a third party, or the reverse, or never
// talk to a third party at all.
//
// Whatever the source, the bytes reach the browser from happyDomain's own
// origin: no source is ever a redirect. That is what keeps the list of domains
// a user manages from being handed to anyone, and what keeps the endpoint
// working when the browser has no route to the outside.
type Source interface {
	// Name is what the operator writes in -favicon-source.
	Name() string

	// FetchIcon returns the icon bytes and the content type to serve them
	// with. site is the http(s) URL of the site the icon is wanted for.
	FetchIcon(site *url.URL) ([]byte, string, error)
}

// Known source names.
const (
	SourceDirect      = "direct"
	SourceDuckDuckGo  = "duckduckgo"
	SourceGoogle      = "google"
	DefaultSourceList = SourceDirect
)

// templatePlaceholder is what a custom source URL uses to mark where the host
// goes.
const templatePlaceholder = "{domain}"

// buildSources turns the configured list into the chain the service walks.
//
// An unknown name is an error rather than a skipped entry: a typo would
// otherwise silently turn into "no icons at all", which looks exactly like a
// network problem.
func buildSources(names []string, client *http.Client) ([]Source, error) {
	var sources []Source

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		source, err := buildSource(name, client)
		if err != nil {
			return nil, err
		}

		sources = append(sources, source)
	}

	return sources, nil
}

// ValidateSourceName reports whether name designates a source, so that the
// configuration can refuse a typo at startup rather than at the first request.
func ValidateSourceName(name string) error {
	_, err := buildSource(name, nil)
	return err
}

// buildSource is filled in as each source gets implemented.
func buildSource(name string, client *http.Client) (Source, error) {
	if strings.EqualFold(name, SourceDirect) {
		return newDirectSource(client), nil
	}

	return nil, fmt.Errorf("unknown favicon source %q: expected %s, %s, %s, or a URL template containing %s", name, SourceDirect, SourceDuckDuckGo, SourceGoogle, templatePlaceholder)
}
