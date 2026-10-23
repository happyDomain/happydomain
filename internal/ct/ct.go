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

// Package ct searches the certificates issued for a domain, as recorded in
// the Certificate Transparency logs.
//
// The logs themselves can only be read in order, not searched by name, so
// the package asks services indexing them instead. Each service is a
// Backend, implemented in a file of its own and listed in Backends.
//
// The package depends on nothing but the standard library, so that it can
// move out of happyDomain, such as next to a checker watching the issuances.
package ct

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

// DefaultUserAgent is sent when Config.UserAgent is empty.
const DefaultUserAgent = "happyDomain (+https://www.happydomain.org/)"

// DefaultTimeout bounds each request made with the default client.
const DefaultTimeout = 60 * time.Second

// maxBodySize caps the size of the answers read from the services: the
// certificates of a large domain can weigh tens of megabytes.
var maxBodySize int64 = 32 << 20

var (
	// ErrRateLimited is returned when the service refuses to answer more
	// requests for now.
	ErrRateLimited = errors.New("rate limited by the certificate transparency service")

	// ErrUnavailable is returned when the service cannot be reached or
	// fails to answer.
	ErrUnavailable = errors.New("certificate transparency service unavailable")
)

// Issuance is a certificate issued for the domain searched.
type Issuance struct {
	// ID identifies the certificate at the service that returned it.
	ID string

	// DNSNames are the names the certificate is valid for, wildcards
	// included, as listed in it.
	DNSNames []string

	// Issuer names the certification authority that issued it.
	Issuer string

	// Serial is the serial number of the certificate, in hexadecimal,
	// when the service tells it.
	Serial string

	NotBefore time.Time
	NotAfter  time.Time

	// CertSHA256 is the hexadecimal SHA-256 of the certificate, when the
	// service tells it.
	CertSHA256 string
}

// Query tells which certificates to search.
type Query struct {
	// Domain is the name searched, with or without its trailing dot.
	Domain string

	// IncludeSubdomains also searches the certificates of the subdomains
	// of Domain.
	IncludeSubdomains bool

	// ExcludeExpired leaves out the certificates no longer valid. A
	// service that only knows the valid ones ignores it.
	ExcludeExpired bool
}

// Searcher searches the certificates issued for a domain.
type Searcher interface {
	Search(ctx context.Context, q Query) ([]Issuance, error)
}

// Config is handed to every backend, each one using what applies to it.
type Config struct {
	// Client makes the requests. Nil uses a client with DefaultTimeout.
	Client *http.Client

	// BaseURL replaces the address of the service, such as in tests.
	BaseURL string

	// APIKey authenticates to the services requiring or rewarding it.
	APIKey string

	// UserAgent is sent with every request. Empty sends DefaultUserAgent.
	UserAgent string
}

// Backend is a service searching the certificate transparency logs.
type Backend struct {
	// ID identifies the backend, such as in settings.
	ID string

	// Name is the name of the service, for humans.
	Name string

	// TakesAPIKey tells the service accepts an API key, in Config.APIKey.
	TakesAPIKey bool

	// New builds the searcher of the service.
	New func(cfg Config) Searcher
}

// Names returns the names the issuances hold that are domain or one of its
// subdomains, lowercased, without trailing dot, sorted and deduplicated. A
// wildcard yields the name it is the wildcard of.
func Names(issuances []Issuance, domain string) []string {
	domain = normalizeName(domain)

	seen := map[string]bool{}
	for _, iss := range issuances {
		for _, name := range iss.DNSNames {
			name = strings.TrimPrefix(normalizeName(name), "*.")
			if name == domain || strings.HasSuffix(name, "."+domain) {
				seen[name] = true
			}
		}
	}

	return slices.Sorted(func(yield func(string) bool) {
		for name := range seen {
			if !yield(name) {
				return
			}
		}
	})
}

func normalizeName(name string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
}

// HTTPError is returned when a service answers with an unexpected status.
// It matches ErrRateLimited or ErrUnavailable when relevant.
type HTTPError struct {
	StatusCode int

	// RetryAfter is how long the service asks to wait, if it tells.
	RetryAfter time.Duration
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("certificate transparency service answered %d %s", e.StatusCode, http.StatusText(e.StatusCode))
}

func (e *HTTPError) Is(target error) bool {
	switch target {
	case ErrRateLimited:
		return e.StatusCode == http.StatusTooManyRequests
	case ErrUnavailable:
		return e.StatusCode >= 500
	}
	return false
}

// baseURL returns cfg.BaseURL, or def when unset.
func (cfg Config) baseURL(def string) string {
	if cfg.BaseURL != "" {
		return strings.TrimSuffix(cfg.BaseURL, "/")
	}
	return def
}

// get fetches url and returns the body of a successful answer.
func (cfg Config) get(ctx context.Context, url string, header http.Header) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		req.Header[k] = v
	}
	req.Header.Set("Accept", "application/json")
	if cfg.UserAgent != "" {
		req.Header.Set("User-Agent", cfg.UserAgent)
	} else {
		req.Header.Set("User-Agent", DefaultUserAgent)
	}

	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: DefaultTimeout}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		herr := &HTTPError{StatusCode: resp.StatusCode}
		if secs, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && secs > 0 {
			herr.RetryAfter = time.Duration(secs) * time.Second
		}
		return nil, herr
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if int64(len(body)) > maxBodySize {
		return nil, fmt.Errorf("certificate transparency service answered more than %d bytes", maxBodySize)
	}

	return body, nil
}
