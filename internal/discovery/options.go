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
	"time"

	"git.happydns.org/happyDomain/internal/netguard"
)

// Default values of Options.
const (
	DefaultConcurrency  = 16
	DefaultMaxQueries   = 2000
	DefaultMaxDepth     = 3
	DefaultTimeout      = 30 * time.Second
	DefaultQueryTimeout = 3 * time.Second
)

// Options tunes an exploration. Zero values take the defaults.
type Options struct {
	// Guard vets every DNS server the exploration talks to. Their
	// addresses come from DNS answers, controlled by whoever runs the
	// domain: without the guard, they would make happyDomain query its own
	// network. A nil Guard refuses every non-public destination.
	Guard *netguard.Guard

	// Resolvers are the "ip:port" of the recursive resolvers asked to find
	// the authoritative servers, the next one being asked when one does not
	// answer: the -default-resolver ones. Chosen by the operator, they are
	// not vetted by the guard.
	Resolvers []string

	// Concurrency is the number of names explored at once.
	Concurrency int

	// MaxQueries caps the number of questions asked to the authoritative
	// servers.
	MaxQueries int

	// MaxDepth caps how many times in-zone targets of answers are
	// followed, from the names the sources yielded.
	MaxDepth int

	// Timeout bounds the whole exploration.
	Timeout time.Duration

	// QueryTimeout bounds each question.
	QueryTimeout time.Duration

	// authPort is the port of the authoritative servers. Tests set it, as
	// they cannot listen on 53.
	authPort string
}

func (o Options) withDefaults() Options {
	if o.Concurrency <= 0 {
		o.Concurrency = DefaultConcurrency
	}
	if o.MaxQueries <= 0 {
		o.MaxQueries = DefaultMaxQueries
	}
	if o.MaxDepth <= 0 {
		o.MaxDepth = DefaultMaxDepth
	}
	if o.Timeout <= 0 {
		o.Timeout = DefaultTimeout
	}
	if o.QueryTimeout <= 0 {
		o.QueryTimeout = DefaultQueryTimeout
	}
	if o.authPort == "" {
		o.authPort = "53"
	}
	return o
}
