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

package middleware

import (
	"net/netip"

	"github.com/gin-gonic/gin"
)

// UnknownClientKey groups all callers whose address could not be determined
// (unix socket, malformed remote address). They share a single bucket rather
// than escaping rate limiting entirely.
const UnknownClientKey = "unknown"

// ClientKey returns the bucket a caller is rate limited under. It normalizes
// ClientIP: an IPv4 address keys on the exact address, an IPv6 address keys on
// its /64 prefix, because a single subscriber is routinely delegated a whole
// /64 and could otherwise rotate addresses inside it for free.
//
// Use this instead of ClientIP for any counter, quota or lockout. ClientIP
// stays the right choice for log lines and for captcha providers, which want
// the exact address.
func ClientKey(c *gin.Context) string {
	addr, err := netip.ParseAddr(c.ClientIP())
	if err != nil {
		return UnknownClientKey
	}

	addr = addr.Unmap()
	if addr.Is4() {
		return addr.String()
	}

	// addr is necessarily a 128 bit address here, so a /64 always fits: no
	// error case to handle, and no per-address fallback that would silently
	// reinstate the rotation bypass this function exists to prevent.
	return netip.PrefixFrom(addr, 64).Masked().String()
}
