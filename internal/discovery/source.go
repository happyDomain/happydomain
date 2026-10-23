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

// Package discovery recomposes the zone of a domain happyDomain has no
// provider access to, from what its authoritative servers publicly answer.
//
// A zone cannot be listed over plain DNS, so the names to ask about come
// from NameSources, each implementing one enumeration technique. Each
// technique lives in its own file and registers itself (RegisterSource);
// its settings are embedded in Settings, which the providers exposing the
// discovery embed in turn. Resolve
// asks the authoritative servers about every name they yield, follows the
// in-zone targets of the answers, and returns the records found. The result
// is a best-effort view of the zone: a name no source thought of stays
// invisible.
package discovery

import (
	"context"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
)

// Name is a name worth asking the authoritative servers about.
type Name struct {
	// FQDN is the fully qualified name, with its trailing dot.
	FQDN string

	// Sources tells which techniques yielded the name.
	Sources []string

	// Types, when set, are the only record types asked for, for sources
	// that know which types exist at the name. Such a source also vouches
	// for the existence of the name, so its answers are kept even when a
	// wildcard would answer the same. Empty asks for a default set chosen
	// after the shape of the name.
	Types []uint16
}

// Target is the domain being explored, as handed to the sources.
type Target struct {
	// Domain is the fully qualified name of the zone apex.
	Domain string

	// Authoritative are the "ip:port" of the authoritative servers of the
	// zone, already vetted by the outbound guard.
	Authoritative []string

	// Known are the subdomains, relative to Domain, found by previous
	// explorations, so a name seen once keeps being looked after.
	Known []string

	// Query asks the authoritative servers about name, within the query
	// budget and the pace of the exploration.
	Query func(ctx context.Context, name string, qtype uint16) (*dns.Msg, error)
}

// NameSource is one technique yielding names to explore.
type NameSource interface {
	// ID identifies the technique in the provenance of the names.
	ID() string

	// Discover returns the names this technique found in t.Domain. Every
	// name must be t.Domain or one of its subdomains; others are ignored.
	Discover(ctx context.Context, t Target) ([]Name, error)
}

// RecordSource is implemented by sources obtaining the records of the zone
// themselves, such as a zone transfer, rather than names to ask about. When
// one of them succeeds, its records are taken as the whole zone and no name
// is explored.
type RecordSource interface {
	NameSource

	// Records returns the records of the zone, or none when the technique
	// does not work on this zone.
	Records(ctx context.Context, t Target) ([]dns.RR, error)
}

// ExhaustiveSource is implemented by sources that may find every name of
// the zone, such as a walk of its NSEC chain. When one of them does, the
// sources coming after it, mere guesses, are not asked.
type ExhaustiveSource interface {
	NameSource

	// DiscoverAll is Discover, also telling whether the names returned are
	// all the names of the zone.
	DiscoverAll(ctx context.Context, t Target) (names []Name, complete bool, err error)
}

// KnownSource yields the subdomains found by previous explorations.
type KnownSource struct{}

func (KnownSource) ID() string {
	return "known"
}

func (s KnownSource) Discover(ctx context.Context, t Target) (names []Name, err error) {
	for _, subdomain := range t.Known {
		names = append(names, Name{
			FQDN:    dns.CanonicalName(helpers.DomainFQDN(subdomain, t.Domain)),
			Sources: []string{s.ID()},
		})
	}
	return
}

// namesFromLists turns lists of names relative to domain, one per line,
// into Names yielded by source. Empty lines, lines starting with # and
// lines that are not host-like names are skipped.
func namesFromLists(source, domain string, lists ...string) (names []Name) {
	for _, list := range lists {
		for line := range strings.Lines(list) {
			label := strings.TrimSpace(line)
			if label == "" || strings.HasPrefix(label, "#") {
				continue
			}
			if !isHostLikeName(label) {
				continue
			}

			names = append(names, Name{
				FQDN:    dns.CanonicalName(helpers.DomainFQDN(label, domain)),
				Sources: []string{source},
			})
		}
	}

	return
}

// isHostLikeName tells whether name is made of letters, digits, hyphens
// and underscores, in dot-separated labels: what lies in a zone, unlike the
// mere DNS syntax that dns.IsDomainName checks, which takes nearly anything.
func isHostLikeName(name string) bool {
	if _, ok := dns.IsDomainName(name); !ok {
		return false
	}

	for _, label := range strings.Split(strings.TrimSuffix(name, "."), ".") {
		if label == "" {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
				return false
			}
		}
	}

	return true
}
