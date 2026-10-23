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
	RegisterSource("wordlist", 900, func(s *Settings) NameSource {
		if s.DisableWordlist {
			return nil
		}
		return WordlistSource{}
	})
}

// WordlistSettings are the settings of the wordlist technique.
type WordlistSettings struct {
	DisableWordlist bool `json:"disable_wordlist,omitempty" happydomain:"label=Do not try common host names,description=By default the 5000 host names most often found in zones (www; mail; vpn...) are tried. The names of usual services (_dmarc; DKIM selectors; SRV services...) always are."`
}

// commonHostNames are the 5000 host names most often found in zones, from
// most to least frequent. It is a verbatim copy of a SecLists file: see
// SECLISTS-LICENSE.
//
//go:embed subdomains-top1million-5000.txt
var commonHostNames string

// WordlistSource yields the host names most often found in zones (www,
// mail, vpn...), the most frequent first, so that a budget running out
// only leaves the rarest ones unasked.
type WordlistSource struct{}

func (WordlistSource) ID() string {
	return "wordlist"
}

func (s WordlistSource) Discover(ctx context.Context, t Target) ([]Name, error) {
	return namesFromLists(s.ID(), t.Domain, commonHostNames), nil
}
