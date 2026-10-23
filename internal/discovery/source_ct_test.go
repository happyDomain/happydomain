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
	"slices"
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

	// Explorations of several domains share the cache.
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

	cache.set("new", []string{"mail.example.com."})
	if _, ok := cache.entries["old"]; ok {
		t.Error("the expired entry was kept on a later set")
	}
	if names, ok := cache.get("new"); !ok || !slices.Equal(names, []string{"mail.example.com."}) {
		t.Errorf("fresh entry = %v, %v; want it served", names, ok)
	}
}

func TestCTSourceCache(t *testing.T) {
	var ok, failing atomic.Int32
	source := CTSource{
		Backends: []ct.Backend{
			fakeBackend("ok", fakeSearcher{searches: &ok, issuances: []ct.Issuance{{DNSNames: []string{"www.example.com"}}}}),
			fakeBackend("failing", fakeSearcher{searches: &failing, err: ct.ErrRateLimited}),
		},
		cache: &ctNamesCache{},
	}

	for range 2 {
		names, _ := source.Discover(context.Background(), Target{Domain: "example.com."})
		if got := fqdns(names); !slices.Equal(got, []string{"www.example.com."}) {
			t.Errorf("names = %v; want www, from the cache the second time", got)
		}
	}

	if n := ok.Load(); n != 1 {
		t.Errorf("the service answering was asked %d times; want once, then cached", n)
	}
	if n := failing.Load(); n != 2 {
		t.Errorf("the failing service was asked %d times; want asked again", n)
	}

	if _, err := source.Discover(context.Background(), Target{Domain: "example.org."}); ok.Load() != 2 || !errors.Is(err, ct.ErrRateLimited) {
		t.Errorf("another domain is not searched apart: %d searches, %v", ok.Load(), err)
	}
}
