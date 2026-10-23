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
	_ "embed"
)

func init() {
	RegisterSource("wellknown", 100, func(s *Settings) NameSource {
		return WellKnownSource{Extra: s.ExtraNames}
	})
}

// WellKnownSettings are the settings of the well-known names technique.
type WellKnownSettings struct {
	ExtraNames string `json:"extra_names,omitempty" happydomain:"label=Additional names,placeholder=intranet,description=Names to look for besides the common ones: one per line and relative to the domain.,textarea"`
}

//go:embed wordlist-wellknown.txt
var wellKnownNames string

// WellKnownSource yields the names where usual services are published: mail
// policies, DKIM selectors and the names of services located through SRV or
// TLSA records, which no list of host names holds. Being few and the most
// useful to the checks, they are meant to be explored before any longer
// list.
type WellKnownSource struct {
	// Extra are more names, relative to the domain, one per line, such as
	// the user knows of.
	Extra string
}

func (WellKnownSource) ID() string {
	return "wellknown"
}

func (s WellKnownSource) Discover(ctx context.Context, t Target) ([]Name, error) {
	return namesFromLists(s.ID(), t.Domain, wellKnownNames, s.Extra), nil
}
