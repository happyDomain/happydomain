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
	"cmp"
	"fmt"
	"slices"
	"sync"
)

// SourceFactory describes one enumeration technique to whoever exposes the
// discovery, such as the public DNS discovery provider, so that adding a
// technique only takes a new file registering it.
type SourceFactory struct {
	// ID identifies the technique. It is the ID of the sources it builds.
	ID string

	// Priority orders the techniques, the lowest first. The names of the
	// first ones are explored first, so the query budget leaves those of
	// the last ones unasked when it runs out. Use:
	//   - below 100, for sources handing the whole zone, such as a zone
	//     transfer;
	//   - 100 to 499, for sources yielding few names, most of them real;
	//   - 500 and above, for long lists of guesses.
	// Techniques of equal priority are ordered by ID.
	Priority int

	// New builds the source out of the settings of the discovery. It
	// returns nil when the settings disable the technique.
	New func(settings *Settings) NameSource
}

var (
	sourceFactoriesMu sync.RWMutex
	sourceFactories   = map[string]SourceFactory{}
)

// RegisterSource makes a technique available. It is meant to be called from
// the init function of the file implementing the technique.
func RegisterSource(id string, priority int, build func(settings *Settings) NameSource) {
	sourceFactoriesMu.Lock()
	defer sourceFactoriesMu.Unlock()

	if _, exists := sourceFactories[id]; exists {
		panic(fmt.Sprintf("discovery: source %q registered twice", id))
	}

	sourceFactories[id] = SourceFactory{
		ID:       id,
		Priority: priority,
		New:      build,
	}
}

// SourceFactories returns the registered techniques, in priority order.
func SourceFactories() []SourceFactory {
	sourceFactoriesMu.RLock()
	defer sourceFactoriesMu.RUnlock()

	factories := make([]SourceFactory, 0, len(sourceFactories))
	for _, f := range sourceFactories {
		factories = append(factories, f)
	}

	slices.SortFunc(factories, func(a, b SourceFactory) int {
		return cmp.Or(cmp.Compare(a.Priority, b.Priority), cmp.Compare(a.ID, b.ID))
	})

	return factories
}
