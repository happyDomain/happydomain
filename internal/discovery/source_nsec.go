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
	"log"
	"slices"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
)

func init() {
	RegisterSource("nsec", 20, func(s *Settings) NameSource {
		if s.DisableNSECWalk {
			return nil
		}
		return NSECSource{}
	})
}

// NSECSettings are the settings of the NSEC walking technique.
type NSECSettings struct {
	DisableNSECWalk bool `json:"disable_nsec_walk,omitempty" happydomain:"label=Do not walk the NSEC chain,description=By default the NSEC records of a zone signed with DNSSEC are followed from name to name: when the zone uses NSEC (not NSEC3) they list every name it holds."`
}

// NSECSource walks the NSEC chain of a zone signed with DNSSEC. Each NSEC
// record names the next name of the zone and the types present at its
// owner, so following them from the apex back to the apex lists every name
// of the zone, and the types to ask for at each.
//
// Zones using NSEC3 hash their names, and servers signing on the fly forge
// NSEC records around the name asked about (RFC 9824): neither can be
// walked, the walk then stops with the names it has confirmed.
type NSECSource struct{}

func (NSECSource) ID() string {
	return "nsec"
}

func (s NSECSource) Discover(ctx context.Context, t Target) ([]Name, error) {
	names, _, err := s.DiscoverAll(ctx, t)
	return names, err
}

// DiscoverAll walks the chain from the apex. It is complete when the chain
// leads back to the apex.
func (s NSECSource) DiscoverAll(ctx context.Context, t Target) (names []Name, complete bool, err error) {
	if t.Query == nil {
		return nil, false, nil
	}

	seen := map[string]bool{}
	for cur := t.Domain; ; {
		seen[cur] = true

		nsec, nsec3, err := s.nsecAt(ctx, t, cur)
		if err != nil {
			return names, false, err
		}
		if nsec == nil {
			if nsec3 {
				log.Printf("discovery: %s: the zone is signed with NSEC3, its names cannot be walked", t.Domain)
			}
			return names, false, nil
		}

		if types := zoneTypes(nsec.TypeBitMap); len(types) > 0 {
			names = append(names, Name{FQDN: cur, Sources: []string{s.ID()}, Types: types})
		}

		next := dns.CanonicalName(nsec.NextDomain)
		switch {
		case next == t.Domain:
			return names, true, nil
		case !dns.IsSubDomain(t.Domain, next), seen[next]:
			return names, false, nil
		case next == "\\000."+cur:
			// Compact denial of existence: the server signs on the
			// fly and makes up the next name.
			return names, false, nil
		}
		cur = next
	}
}

// nsecAt returns the NSEC record owned by name, nil when there is none to
// be found, and whether the zone appears to be signed with NSEC3 instead.
//
// It asks for the NSEC record at name, and falls back on asking about the
// name right after it in the canonical order, which does not exist: the
// NSEC record proving so is owned by name. A record owned by another name
// means name does not exist, the chain is forged.
func (NSECSource) nsecAt(ctx context.Context, t Target, name string) (*dns.NSEC, bool, error) {
	nsec3 := false

	for _, question := range []struct {
		name  string
		qtype uint16
	}{
		{name, dns.TypeNSEC},
		{"\\000." + name, dns.TypeA},
	} {
		r, err := t.Query(ctx, question.name, question.qtype)
		if err != nil {
			if errors.Is(err, errBudget) || ctx.Err() != nil {
				return nil, nsec3, err
			}
			continue
		}

		for _, rr := range slices.Concat(r.Answer, r.Ns) {
			switch rr := rr.(type) {
			case *dns.NSEC:
				if dns.CanonicalName(rr.Hdr.Name) == name {
					return rr, false, nil
				}
			case *dns.NSEC3, *dns.NSEC3PARAM:
				nsec3 = true
			}
		}
	}

	return nil, nsec3, nil
}

// zoneTypes returns the types of bitmap that are worth asking for: those
// of the records making the DNSSEC chain of trust are left out, they are
// not part of the zone as its owner edits it.
func zoneTypes(bitmap []uint16) (types []uint16) {
	for _, rrtype := range bitmap {
		if !helpers.IsDNSSECType(rrtype) {
			types = append(types, rrtype)
		}
	}
	return
}
