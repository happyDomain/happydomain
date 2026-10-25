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

package helpers

import (
	"context"

	"github.com/miekg/dns"
)

// Exchange sends m to server, and sends it again over TCP when the answer
// is truncated: an answer too large for UDP, DNSSEC records included, would
// otherwise come back empty or partial.
func Exchange(ctx context.Context, client *dns.Client, m *dns.Msg, server string) (*dns.Msg, error) {
	r, _, err := client.ExchangeContext(ctx, m, server)
	if err != nil || r == nil || !r.Truncated || client.Net == "tcp" {
		return r, err
	}

	tcp := *client
	tcp.Net = "tcp"
	r, _, err = tcp.ExchangeContext(ctx, m, server)
	return r, err
}
