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

package discovery

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/ct"
)

func init() {
	RegisterSource("ct", 200, func(s *Settings) NameSource {
		if s.DisableCT {
			return nil
		}
		return CTSource{}
	})
}

// CTSettings are the settings of the certificate transparency technique.
type CTSettings struct {
	// The description names the services of ct.Backends: a tag cannot
	// list them, so keep it in sync when adding one.
	DisableCT bool `json:"disable_ct,omitempty" happydomain:"label=Do not search the certificate transparency logs,description=By default the names of the certificates issued for the domain are looked for in the certificate transparency logs: the name of the domain is then sent to Cert Spotter and crt.sh."`
}

const (
	// ctTimeout bounds the search of the certificate transparency
	// services, to leave the rest of the exploration time to the DNS.
	ctTimeout = 30 * time.Second

	// ctCacheTTL is how long the names a service returned are reused: the
	// zone is refreshed far more often than certificates are issued, and
	// the services limit how often they may be asked.
	ctCacheTTL = 6 * time.Hour

	// ctFailureTTL is how long the names of a failed search are reused: a
	// service too slow for a domain, or down, would most likely fail the
	// same way if asked again at once.
	ctFailureTTL = 30 * time.Minute

	// ctRateLimitTTL is how long a service limiting the rate of its answers
	// is left alone, when it does not tell.
	ctRateLimitTTL = time.Hour
)

// CTSource yields the names of the certificates issued for the domain, as
// recorded in the certificate transparency logs, still valid: a certificate
// is usually requested for a name in use. The name may have disappeared
// from the zone since, so it is explored like any guess.
type CTSource struct {
	// Backends are the services asked, all at once. Nil asks every
	// service of ct.Backends.
	Backends []ct.Backend

	// Config is handed to every service.
	Config ct.Config

	// cache keeps the names found, nil uses the cache shared by every
	// CTSource.
	cache *ctNamesCache
}

func (CTSource) ID() string {
	return "ct"
}

func (s CTSource) Discover(ctx context.Context, t Target) (names []Name, err error) {
	ctx, cancel := context.WithTimeout(ctx, ctTimeout)
	defer cancel()

	backends := s.Backends
	if backends == nil {
		backends = ct.Backends
	}

	domain := dns.CanonicalName(t.Domain)

	type result struct {
		names []string
		err   error
	}
	results := make([]result, len(backends))

	var wg sync.WaitGroup
	for i, backend := range backends {
		wg.Go(func() {
			results[i].names, results[i].err = s.search(ctx, backend, domain)
		})
	}
	wg.Wait()

	// A service failing does not spoil the names of the others.
	var errs []error
	seen := map[string]bool{}
	for i, r := range results {
		if r.err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", backends[i].Name, r.err))
		}

		for _, name := range r.names {
			fqdn := dns.CanonicalName(name)
			if seen[fqdn] || !isHostLikeName(fqdn) {
				continue
			}
			seen[fqdn] = true

			names = append(names, Name{
				FQDN:    fqdn,
				Sources: []string{s.ID()},
			})
		}
	}

	return names, errors.Join(errs...)
}

// search returns the names backend knows in domain, from the cache when it
// was asked recently. The names of a failed search are kept too, for less
// time, see retention.
func (s CTSource) search(ctx context.Context, backend ct.Backend, domain string) ([]string, error) {
	cache := s.cache
	if cache == nil {
		cache = sharedCTCache
	}

	key := backend.ID + " " + domain
	if names, ok := cache.get(key); ok {
		return names, nil
	}
	if until := cache.pausedUntil(backend.ID); time.Now().Before(until) {
		return nil, fmt.Errorf("%w, until %s", ct.ErrRateLimited, until.Format(time.TimeOnly))
	}

	issuances, err := backend.New(s.Config).Search(ctx, ct.Query{
		Domain:            domain,
		IncludeSubdomains: true,
		ExcludeExpired:    true,
	})
	names := ct.Names(issuances, domain)

	// A search the caller gave up tells nothing about the service.
	if errors.Is(err, context.Canceled) {
		return names, err
	}

	ttl := retention(err)
	if errors.Is(err, ct.ErrRateLimited) {
		// The limit applies to the instance, whatever the domain.
		cache.pause(backend.ID, time.Now().Add(ttl))
	}
	cache.set(key, names, ttl)

	return names, err
}

// retention tells how long the names of a search ending with err are reused.
func retention(err error) time.Duration {
	var herr *ct.HTTPError
	switch {
	case err == nil, errors.Is(err, ct.ErrTruncated):
		// Asking again would stop at the same place.
		return ctCacheTTL
	case errors.Is(err, ct.ErrRateLimited):
		if errors.As(err, &herr) && herr.RetryAfter > 0 {
			return min(herr.RetryAfter, ctCacheTTL)
		}
		return ctRateLimitTTL
	default:
		return ctFailureTTL
	}
}

// sharedCTCache is the cache of the CTSources not given their own.
var sharedCTCache = &ctNamesCache{}

// ctNamesCache keeps the names returned by the certificate transparency
// services, and until when each service limiting its rate is left alone.
// Expired entries are dropped as new ones come.
type ctNamesCache struct {
	mu      sync.Mutex
	entries map[string]ctNamesCacheEntry

	// paused holds, by backend ID, until when the service is not asked.
	paused map[string]time.Time
}

type ctNamesCacheEntry struct {
	names   []string
	expires time.Time
}

func (c *ctNamesCache) get(key string) ([]string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expires) {
		return nil, false
	}
	return entry.names, true
}

func (c *ctNamesCache) set(key string, names []string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, entry := range c.entries {
		if now.After(entry.expires) {
			delete(c.entries, k)
		}
	}

	if c.entries == nil {
		c.entries = map[string]ctNamesCacheEntry{}
	}
	c.entries[key] = ctNamesCacheEntry{names: names, expires: now.Add(ttl)}
}

func (c *ctNamesCache) pausedUntil(backendID string) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.paused[backendID]
}

func (c *ctNamesCache) pause(backendID string, until time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.paused == nil {
		c.paused = map[string]time.Time{}
	}
	c.paused[backendID] = until
}
