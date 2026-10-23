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
	"flag"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/ct"
)

// fakeSearcher answers every search with its issuances and error, counting
// the searches.
type fakeSearcher struct {
	issuances []ct.Issuance
	err       error
	searches  *atomic.Int32
	query     *ct.Query
}

func (f fakeSearcher) Search(ctx context.Context, q ct.Query) ([]ct.Issuance, error) {
	if f.searches != nil {
		f.searches.Add(1)
	}
	if f.query != nil {
		*f.query = q
	}
	return f.issuances, f.err
}

func fakeBackend(id string, searcher fakeSearcher) ct.Backend {
	return ct.Backend{
		ID:   id,
		Name: id,
		New:  func(ct.Config) ct.Searcher { return searcher },
	}
}

func TestCTFactory(t *testing.T) {
	f := factory(t, "ct")

	if _, ok := f.New(settingsFrom(t, `{}`)).(CTSource); !ok {
		t.Error("the certificate transparency search is disabled by default")
	}
	if source := f.New(settingsFrom(t, `{"disable_ct":true}`)); source != nil {
		t.Errorf("disable_ct gives %v; want no source", source)
	}
}

func TestCTSource(t *testing.T) {
	var query ct.Query
	errDown := errors.New("down")

	source := CTSource{
		Backends: []ct.Backend{
			fakeBackend("first", fakeSearcher{
				query: &query,
				issuances: []ct.Issuance{
					{DNSNames: []string{"example.com", "*.example.com", "www.example.com"}},
					{DNSNames: []string{"*.dev.example.com", "other.org", "bad_name!.example.com"}},
				},
			}),
			fakeBackend("second", fakeSearcher{
				err: errDown,
				issuances: []ct.Issuance{
					{DNSNames: []string{"WWW.example.com", "mail.example.com"}},
				},
			}),
		},
		cache: &ctNamesCache{},
	}

	names, err := source.Discover(context.Background(), Target{Domain: "Example.com"})
	if !errors.Is(err, errDown) {
		t.Errorf("err = %v; want the error of the failing service", err)
	}

	want := []string{"dev.example.com.", "example.com.", "www.example.com.", "mail.example.com."}
	if got := fqdns(names); !slices.Equal(got, want) {
		t.Errorf("names = %v; want %v", got, want)
	}
	for _, name := range names {
		if !slices.Equal(name.Sources, []string{"ct"}) || name.Types != nil {
			t.Errorf("%s = %+v; want it from ct, any type", name.FQDN, name)
		}
	}

	if query != (ct.Query{Domain: "example.com.", IncludeSubdomains: true, ExcludeExpired: true}) {
		t.Errorf("query = %+v; want the valid certificates of the domain and its subdomains", query)
	}
}

// blockingSearcher answers once its context is done.
type blockingSearcher struct{}

func (blockingSearcher) Search(ctx context.Context, q ct.Query) ([]ct.Issuance, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestCTSourceBoundByContext(t *testing.T) {
	source := CTSource{
		Backends: []ct.Backend{{ID: "slow", Name: "slow", New: func(ct.Config) ct.Searcher { return blockingSearcher{} }}},
		cache:    &ctNamesCache{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	if _, err := source.Discover(ctx, Target{Domain: "example.com."}); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v; want the deadline of the exploration", err)
	}
	if elapsed := time.Since(start); elapsed > ctTimeout/2 {
		t.Errorf("Discover took %s; want it to end with the exploration", elapsed)
	}
}

func TestCTSourceAllFailing(t *testing.T) {
	errA, errB := errors.New("a down"), errors.New("b down")
	source := CTSource{
		Backends: []ct.Backend{
			fakeBackend("a", fakeSearcher{err: errA}),
			fakeBackend("b", fakeSearcher{err: errB}),
		},
		cache: &ctNamesCache{},
	}

	names, err := source.Discover(context.Background(), Target{Domain: "example.com."})
	if len(names) != 0 || !errors.Is(err, errA) || !errors.Is(err, errB) {
		t.Errorf("Discover = %v, %v; want no name and both errors", names, err)
	}
}

func TestCTSourceConcurrent(t *testing.T) {
	source := CTSource{
		Backends: []ct.Backend{fakeBackend("ok", fakeSearcher{issuances: []ct.Issuance{{DNSNames: []string{"www.example.com"}}}})},
		cache:    &ctNamesCache{},
	}

	// Meant for the race detector: explorations of several domains share
	// the cache.
	done := make(chan []Name)
	for _, domain := range []string{"example.com.", "example.com.", "example.org."} {
		go func() {
			names, _ := source.Discover(context.Background(), Target{Domain: domain})
			done <- names
		}()
	}
	for range 3 {
		<-done
	}
}

func TestCTNamesCacheExpiry(t *testing.T) {
	cache := &ctNamesCache{entries: map[string]ctNamesCacheEntry{
		"old": {names: []string{"www.example.com."}, expires: time.Now().Add(-time.Second)},
	}}

	if names, ok := cache.get("old"); ok {
		t.Errorf("an expired entry was served: %v", names)
	}

	cache.set("new", []string{"mail.example.com."}, ctCacheTTL)
	if _, ok := cache.entries["old"]; ok {
		t.Error("the expired entry was kept on a later set")
	}
	if names, ok := cache.get("new"); !ok || !slices.Equal(names, []string{"mail.example.com."}) {
		t.Errorf("fresh entry = %v, %v; want it served", names, ok)
	}
}

// retainedFor tells how long the cache keeps the names of key.
func retainedFor(t *testing.T, cache *ctNamesCache, key string) time.Duration {
	t.Helper()
	entry, ok := cache.entries[key]
	if !ok {
		t.Fatalf("%q is not in the cache", key)
	}
	return time.Until(entry.expires)
}

func assertRetained(t *testing.T, cache *ctNamesCache, key string, want time.Duration) {
	t.Helper()
	if got := retainedFor(t, cache, key); got > want || got < want-time.Minute {
		t.Errorf("%q kept for %s; want %s", key, got, want)
	}
}

func TestCTSourceCache(t *testing.T) {
	var ok, failing atomic.Int32
	errDown := errors.New("down")
	cache := &ctNamesCache{}
	source := CTSource{
		Backends: []ct.Backend{
			fakeBackend("ok", fakeSearcher{searches: &ok, issuances: []ct.Issuance{{DNSNames: []string{"www.example.com"}}}}),
			fakeBackend("failing", fakeSearcher{searches: &failing, err: errDown, issuances: []ct.Issuance{{DNSNames: []string{"mail.example.com"}}}}),
		},
		cache: cache,
	}

	for i := range 2 {
		names, err := source.Discover(context.Background(), Target{Domain: "example.com."})
		if got := fqdns(names); !slices.Equal(got, []string{"www.example.com.", "mail.example.com."}) {
			t.Errorf("names = %v; want www and mail, from the cache the second time", got)
		}
		if wantErr := i == 0; errors.Is(err, errDown) != wantErr {
			t.Errorf("search %d: err = %v; want the failure only when the service was asked", i, err)
		}
	}

	if n := ok.Load(); n != 1 {
		t.Errorf("the service answering was asked %d times; want once, then cached", n)
	}
	if n := failing.Load(); n != 1 {
		t.Errorf("the failing service was asked %d times; want once, then its partial names cached", n)
	}
	assertRetained(t, cache, "ok example.com.", ctCacheTTL)
	assertRetained(t, cache, "failing example.com.", ctFailureTTL)

	if _, err := source.Discover(context.Background(), Target{Domain: "example.org."}); ok.Load() != 2 || failing.Load() != 2 || !errors.Is(err, errDown) {
		t.Errorf("another domain is not searched apart: %d and %d searches, %v", ok.Load(), failing.Load(), err)
	}
}

func TestCTSourceTimeoutRetained(t *testing.T) {
	cache := &ctNamesCache{}
	source := CTSource{
		Backends: []ct.Backend{{ID: "slow", Name: "slow", New: func(ct.Config) ct.Searcher { return blockingSearcher{} }}},
		cache:    cache,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	source.Discover(ctx, Target{Domain: "example.com."})

	// A service too slow for the domain would be as slow the next time.
	assertRetained(t, cache, "slow example.com.", ctFailureTTL)
}

func TestCTSourceCanceledNotRetained(t *testing.T) {
	var searches atomic.Int32
	cache := &ctNamesCache{}
	source := CTSource{
		Backends: []ct.Backend{fakeBackend("cut", fakeSearcher{searches: &searches, err: context.Canceled})},
		cache:    cache,
	}

	for range 2 {
		source.Discover(context.Background(), Target{Domain: "example.com."})
	}
	if n := searches.Load(); n != 2 {
		t.Errorf("a search given up by the caller was asked %d times; want asked again", n)
	}
}

func TestCTSourceRateLimited(t *testing.T) {
	for name, tc := range map[string]struct {
		retryAfter time.Duration
		want       time.Duration
	}{
		"told":      {2 * time.Minute, 2 * time.Minute},
		"untold":    {0, ctRateLimitTTL},
		"excessive": {48 * time.Hour, ctCacheTTL},
	} {
		t.Run(name, func(t *testing.T) {
			var searches atomic.Int32
			cache := &ctNamesCache{}
			source := CTSource{
				Backends: []ct.Backend{fakeBackend("limited", fakeSearcher{
					searches:  &searches,
					err:       &ct.HTTPError{StatusCode: http.StatusTooManyRequests, RetryAfter: tc.retryAfter},
					issuances: []ct.Issuance{{DNSNames: []string{"www.example.com", "www.example.org"}}},
				})},
				cache: cache,
			}

			names, err := source.Discover(context.Background(), Target{Domain: "example.com."})
			if got := fqdns(names); !slices.Equal(got, []string{"www.example.com."}) || !errors.Is(err, ct.ErrRateLimited) {
				t.Errorf("Discover = %v, %v; want the names got before the limit, and the limit", got, err)
			}
			assertRetained(t, cache, "limited example.com.", tc.want)

			// The limit holds for every domain: the service is not
			// asked again before it ends.
			names, err = source.Discover(context.Background(), Target{Domain: "example.org."})
			if len(names) != 0 || !errors.Is(err, ct.ErrRateLimited) {
				t.Errorf("Discover of another domain = %v, %v; want no name and the limit", names, err)
			}
			if n := searches.Load(); n != 1 {
				t.Errorf("the service was asked %d times while limited; want once", n)
			}

			cache.paused["limited"] = time.Now().Add(-time.Second)
			source.Discover(context.Background(), Target{Domain: "example.org."})
			if n := searches.Load(); n != 2 {
				t.Errorf("the service was asked %d times; want asked again once the limit ended", n)
			}
		})
	}
}

func TestCTSourceTruncated(t *testing.T) {
	var searches atomic.Int32
	cache := &ctNamesCache{}
	source := CTSource{
		Backends: []ct.Backend{fakeBackend("capped", fakeSearcher{
			searches:  &searches,
			err:       ct.ErrTruncated,
			issuances: []ct.Issuance{{DNSNames: []string{"www.example.com"}}},
		})},
		cache: cache,
	}

	names, err := source.Discover(context.Background(), Target{Domain: "example.com."})
	if got := fqdns(names); !slices.Equal(got, []string{"www.example.com."}) || !errors.Is(err, ct.ErrTruncated) {
		t.Errorf("Discover = %v, %v; want the names got, and the truncation told", got, err)
	}

	// Asking again would stop at the same page.
	assertRetained(t, cache, "capped example.com.", ctCacheTTL)
	source.Discover(context.Background(), Target{Domain: "example.com."})
	if n := searches.Load(); n != 1 {
		t.Errorf("the truncated search was asked %d times; want once, then cached", n)
	}
}

func TestCTSourceAPIKey(t *testing.T) {
	// The backends are built concurrently by Discover.
	var mu sync.Mutex
	keys := map[string]string{}
	backend := func(id string) ct.Backend {
		return ct.Backend{
			ID:   id,
			Name: id,
			New: func(cfg ct.Config) ct.Searcher {
				mu.Lock()
				defer mu.Unlock()
				keys[id] = cfg.APIKey
				return fakeSearcher{}
			},
		}
	}

	key := "instance-key"
	ctAPIKeys["keyed"] = &key
	defer delete(ctAPIKeys, "keyed")

	source := CTSource{
		Backends: []ct.Backend{backend("keyed"), backend("unkeyed")},
		cache:    &ctNamesCache{},
	}
	if _, err := source.Discover(context.Background(), Target{Domain: "example.com."}); err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if keys["keyed"] != "instance-key" || keys["unkeyed"] != "" {
		t.Errorf("API keys = %v; want the key of the instance to its service only", keys)
	}
}

func TestCTAPIKeyFlags(t *testing.T) {
	for _, backend := range ct.Backends {
		if f := flag.Lookup(backend.ID + "-api-key"); (f != nil) != backend.TakesAPIKey {
			t.Errorf("flag for the API key of %s: %v; want one only when it takes a key", backend.ID, f)
		}
	}
}

func TestCTSourceOwnAPIKey(t *testing.T) {
	var got string
	backend := ct.Backend{
		ID:   "keyed",
		Name: "keyed",
		New: func(cfg ct.Config) ct.Searcher {
			got = cfg.APIKey
			return fakeSearcher{}
		},
	}

	key := "instance-key"
	ctAPIKeys["keyed"] = &key
	defer delete(ctAPIKeys, "keyed")

	source := CTSource{Backends: []ct.Backend{backend}, Config: ct.Config{APIKey: "own-key"}, cache: &ctNamesCache{}}
	if _, err := source.Discover(context.Background(), Target{Domain: "example.com."}); err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if got != "own-key" {
		t.Errorf("API key = %q; want the one of the source over the one of the instance", got)
	}
}
