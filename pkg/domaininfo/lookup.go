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

package domaininfo

import (
	"context"
	"sync"
)

// Result is the outcome of the lookup of a domain.
type Result struct {
	Info *DomainInfo
	Err  error
}

// LookupMany looks the names up with get, each distinct name once and at most
// concurrency at a time, so that many lookups do not overwhelm the registries.
// It stops starting lookups once ctx is done: the names it did not look up
// are missing from the result.
func LookupMany(ctx context.Context, get Getter, names []string, concurrency int) map[string]Result {
	if concurrency < 1 {
		concurrency = 1
	}

	results := make(map[string]Result, len(names))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	seen := make(map[string]bool, len(names))
dispatch:
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true

		// select picks at random among ready cases: do not start a lookup
		// once ctx is done, even when a slot is free.
		if ctx.Err() != nil {
			break
		}
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break dispatch
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			defer func() { <-sem }()

			info, err := get(ctx, name)

			mu.Lock()
			results[name] = Result{Info: info, Err: err}
			mu.Unlock()
		}(name)
	}
	wg.Wait()

	return results
}
