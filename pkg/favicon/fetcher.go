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
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	// FetchTimeout bounds every request made on someone else's behalf, both the
	// page fetch performed during discovery and the icon download. It is
	// exported so a caller building the *http.Client passed to
	// NewFaviconService can size its own timeout consistently.
	FetchTimeout = 5 * time.Second

	// maxIconSize is how much of a single icon we are willing to buffer and
	// cache. A favicon past that size is an anomaly, and accepting one only
	// costs memory that the whole cache then has to account for.
	maxIconSize = 256 << 10 // 256kB

	// negativeTTL is how long a failure is remembered. Without it, every
	// request for a domain that has no favicon starts a fresh outbound fetch,
	// which turns this service into an amplifier aimed at a third party.
	negativeTTL = 15 * time.Minute
)

// allowedContentTypes maps the media types we accept from a remote host to the
// value we hand back. Anything else is refused rather than reflected: the bytes
// are served from our own origin, so a remote "text/html" would be a script
// running as us.
var allowedContentTypes = map[string]string{
	"image/apng":               "image/apng",
	"image/avif":               "image/avif",
	"image/bmp":                "image/bmp",
	"image/gif":                "image/gif",
	"image/jpeg":               "image/jpeg",
	"image/png":                "image/png",
	"image/svg+xml":            "image/svg+xml",
	"image/vnd.microsoft.icon": "image/x-icon",
	"image/webp":               "image/webp",
	"image/x-icon":             "image/x-icon",
}

type FaviconService struct {
	cache   cache
	sources []Source
	group   singleflight.Group
}

// NewFaviconService builds the service around an HTTP client and the list of
// sources the operator asked for, in the order they will be tried.
//
// The client is what makes fetching a URL a hostile party controls acceptable:
// callers are expected to pass one built around an outbound guard that pins
// each dial to an address it has already checked and re-checks every
// redirect, so neither a rebinding answer nor a redirect nor an icon link
// pointing at an internal service reaches anything but a public address. It
// applies to the third-party sources too, so an operator pointing a template
// at an icon service on their own network has to allow that address
// explicitly in the guard. A nil client falls back to http.DefaultClient,
// which applies no such policy and is only fit for tests.
//
// An empty list yields a nil service, which is how the endpoint gets left
// undeclared rather than answering nothing.
func NewFaviconService(client *http.Client, sourceNames []string) (*FaviconService, error) {
	if client == nil {
		client = http.DefaultClient
	}

	sources, err := buildSources(sourceNames, client)
	if err != nil {
		return nil, err
	}

	if len(sources) == 0 {
		return nil, nil
	}

	return &FaviconService{
		cache:   newMemCache(),
		sources: sources,
	}, nil
}

// Sources lists the configured source names, for the startup log.
func (fs *FaviconService) Sources() []string {
	if fs == nil {
		return nil
	}

	names := make([]string, 0, len(fs.sources))
	for _, source := range fs.sources {
		names = append(names, source.Name())
	}

	return names
}

// Fetch retrieves the favicon for the given URL. It returns the icon bytes,
// content type, and any error. Successful results are cached with the given
// TTL, failures for a shorter one.
func (fs *FaviconService) Fetch(rawURL string, ttl time.Duration) ([]byte, string, error) {
	if fs == nil {
		return nil, "", fmt.Errorf("favicon fetching is disabled")
	}

	// The cache key is namespaced by ttl: two callers fetching the same URL
	// under different ttls (a domain lookup and a provider icon lookup can
	// coincide) must not share an entry, or whichever populates it first
	// fixes its ttl for the other.
	cacheKey := fmt.Sprintf("%d:%s", ttl, rawURL)

	if entry, ok := fs.cache.lookup(cacheKey); ok {
		return entry.bytes, entry.contentType, entry.err
	}

	// singleflight collapses concurrent misses for the same key into one
	// outbound fetch: without it, a burst of requests for the same cold URL
	// would each fetch independently, working against the point of the
	// cache and the anti-amplification role of negativeTTL above.
	type fetchResult struct {
		bytes       []byte
		contentType string
	}

	result, err, _ := fs.group.Do(cacheKey, func() (any, error) {
		iconBytes, contentType, err := fs.fetch(rawURL)
		if err != nil {
			fs.cache.store(cacheKey, &cacheEntry{err: err, expiresAt: time.Now().Add(negativeTTL)})
			return nil, err
		}

		fs.cache.store(cacheKey, &cacheEntry{
			bytes:       iconBytes,
			contentType: contentType,
			expiresAt:   time.Now().Add(ttl),
		})

		return fetchResult{bytes: iconBytes, contentType: contentType}, nil
	})
	if err != nil {
		return nil, "", err
	}

	r := result.(fetchResult)
	return r.bytes, r.contentType, nil
}

// FetchDomain retrieves the favicon for a domain name.
func (fs *FaviconService) FetchDomain(domain string) ([]byte, string, error) {
	// Only a bare host may be pasted into the URL below: anything else would
	// let the caller steer the request somewhere other than the host we are
	// about to name in the cache key.
	if domain == "" || strings.ContainsAny(domain, "/:@?#\\ ") {
		return nil, "", fmt.Errorf("invalid domain: %q", domain)
	}

	return fs.Fetch("https://"+strings.ToLower(domain), 24*time.Hour)
}

// fetch walks the source chain and returns the first icon obtained.
//
// A source that fails is not fatal: the point of the chain is that a domain the
// crawlers ignore can still be served by asking the site, and a site that is
// unreachable from here can still be served by a crawler.
func (fs *FaviconService) fetch(rawURL string) ([]byte, string, error) {
	site, err := validateURLShape(rawURL)
	if err != nil {
		return nil, "", err
	}

	var errs []error

	for _, source := range fs.sources {
		iconBytes, contentType, err := source.FetchIcon(site)
		if err == nil {
			return iconBytes, contentType, nil
		}

		errs = append(errs, fmt.Errorf("%s: %w", source.Name(), err))
	}

	return nil, "", fmt.Errorf("no favicon for %s: %w", site.Host, errors.Join(errs...))
}

// downloadIcon fetches one icon URL through client and decides what content
// type it may be served as.
func downloadIcon(client *http.Client, iconURL string) ([]byte, string, error) {
	if _, err := validateURLShape(iconURL); err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest(http.MethodGet, iconURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	// Some icon services answer 404 with a placeholder image rather than an
	// empty body, and a browser would happily display it: the status is the
	// only thing telling us the domain is unknown, so it is what decides
	// whether the next source gets a turn.
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, iconURL)
	}

	// Read one byte past the limit to tell a big icon from a truncated one.
	// Truncating silently is worse than failing: the bytes still look like an
	// image to every check we make, so a half-downloaded icon would be cached
	// for a day and displayed broken, instead of letting the next source try.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxIconSize+1))
	if err != nil {
		return nil, "", err
	}

	if len(body) > maxIconSize {
		return nil, "", fmt.Errorf("%s: icon larger than %d bytes", iconURL, maxIconSize)
	}

	// An empty body passes every content check that follows, since the type
	// comes from the header, and would be cached and served as an image that
	// no browser can display.
	if len(body) == 0 {
		return nil, "", fmt.Errorf("%s: empty icon", iconURL)
	}

	contentType, err := imageContentType(resp.Header.Get("Content-Type"), body)
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", iconURL, err)
	}

	return body, contentType, nil
}

// imageContentType validates what the remote host told us the icon is, and
// falls back to sniffing when it told us nothing.
func imageContentType(header string, body []byte) (string, error) {
	if strings.TrimSpace(header) == "" {
		header = http.DetectContentType(body)
	}

	mediaType, _, err := mime.ParseMediaType(header)
	if err != nil {
		return "", fmt.Errorf("unusable Content-Type %q: %w", header, err)
	}

	contentType, ok := allowedContentTypes[strings.ToLower(mediaType)]
	if !ok {
		return "", fmt.Errorf("refusing to serve favicon of type %q", mediaType)
	}

	return contentType, nil
}
