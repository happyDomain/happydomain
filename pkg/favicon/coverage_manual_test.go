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

package favicon_test

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"git.happydns.org/happyDomain/pkg/favicon"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	_ "git.happydns.org/happyDomain/providers"
)

// TestProviderCoverage reports, for each registered provider, which sources can
// serve its icon. It reaches the network, so it only runs when explicitly asked
// for: FAVICON_COVERAGE=1 go test -run TestProviderCoverage -v ./pkg/favicon/
//
// It exists to answer one question: can the embedded PNGs be dropped?
func TestProviderCoverage(t *testing.T) {
	if os.Getenv("FAVICON_COVERAGE") == "" {
		t.Skip("set FAVICON_COVERAGE=1 to run this network-reaching report")
	}

	sources := []string{"direct", "duckduckgo"}

	services := map[string]*favicon.FaviconService{}
	for _, name := range sources {
		service, err := favicon.NewFaviconService(nil, []string{name})
		if err != nil {
			t.Fatalf("building %s source: %s", name, err)
		}
		services[name] = service
	}

	type result struct {
		ptype   string
		website string
		ok      map[string]bool
		size    map[string]int
	}

	var (
		mu      sync.Mutex
		results []result
		wg      sync.WaitGroup
		sem     = make(chan struct{}, 8)
	)

	for ptype, creator := range providerReg.GetProviders() {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			r := result{ptype: ptype, website: creator.Infos.Website, ok: map[string]bool{}, size: map[string]int{}}

			if r.website != "" {
				for _, name := range sources {
					body, _, err := services[name].Fetch(r.website, time.Minute)
					r.ok[name] = err == nil
					r.size[name] = len(body)
				}
			}

			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		})
	}

	wg.Wait()
	sort.Slice(results, func(i, j int) bool { return results[i].ptype < results[j].ptype })

	counts := map[string]int{}
	var noWebsite, none []string

	for _, r := range results {
		if r.website == "" {
			noWebsite = append(noWebsite, r.ptype)
			continue
		}

		line := fmt.Sprintf("%-24s", r.ptype)
		covered := false
		for _, name := range sources {
			if r.ok[name] {
				counts[name]++
				covered = true
				line += fmt.Sprintf(" %s=%dB", name, r.size[name])
			} else {
				line += fmt.Sprintf(" %s=KO", name)
			}
		}
		if !covered {
			none = append(none, r.ptype)
		}
		t.Log(line)
	}

	t.Logf("providers: %d", len(results))
	t.Logf("without Website: %d %v", len(noWebsite), noWebsite)
	for _, name := range sources {
		t.Logf("%s covers: %d", name, counts[name])
	}
	t.Logf("no source covers: %d %v", len(none), none)
}
