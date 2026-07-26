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

package app

import (
	"log"

	"git.happydns.org/happyDomain/internal/netguard"
)

// outboundGuards holds the two destination policies, built once at startup and
// injected into everything that dials on a user's behalf.
//
// They are split because the two surfaces have different exposure: Resolver
// backs endpoints that need no account at all, while Outbound backs actions a
// registered user takes, one of which hands a configured API key to the address
// it is given.
type outboundGuards struct {
	// Outbound covers provider endpoints, certificate probes, MTA-STS policy
	// fetches and notification webhooks.
	Outbound *netguard.Guard

	// Resolver covers the DNS server chosen in the resolver tool.
	Resolver *netguard.Guard
}

// initGuards builds both guards from the configuration. A malformed allow-list
// stops happyDomain here rather than at the first refused request, when it
// would read as a bug in the feature instead of a typo in the config.
func (app *App) initGuards() {
	var err error

	app.guards.Outbound, err = netguard.New("outbound", "-outbound-allowed-target", app.cfg.OutboundAllowedTargets)
	if err != nil {
		log.Fatalf("Invalid -outbound-allowed-target: %s", err)
	}

	app.guards.Resolver, err = netguard.New("resolver", "-resolver-allowed-target", app.cfg.ResolverAllowedTargets)
	if err != nil {
		log.Fatalf("Invalid -resolver-allowed-target: %s", err)
	}
}
