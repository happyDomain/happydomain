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

package usecase

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

// pickResolver turns the resolver a request asked for into a dialable
// "ip:port". It is the single place that decides which DNS server happyDomain
// talks to on a caller's behalf, for every resolver endpoint.
//
// Everything the request supplies goes through the resolver guard, and that
// includes the plain `name` form: the frontend sends "8.8.8.8" or "9.9.9.10"
// straight in that field (see web/src/lib/resolver.ts), so it is exactly as
// caller-controlled as `custom` is. Only the "local" branch is exempt, because
// /etc/resolv.conf is operator-chosen and routinely points at loopback.
func (ru *resolverUsecase) pickResolver(ctx context.Context, name, custom string) (string, error) {
	resolver := name
	switch resolver {
	case "":
		// Default to a public, well-known resolver when the caller did not
		// specify one. Use Cloudflare's 1.1.1.1 as a sane default.
		resolver = "1.1.1.1"
	case "custom":
		if custom == "" {
			return "", happydns.ValidationError{Msg: "custom resolver address required"}
		}
		resolver = custom
	case "local":
		cConf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
		if err != nil {
			return "", happydns.InternalError{
				Err:         fmt.Errorf("unable to load ClientConfigFromFile: %s", err.Error()),
				UserMessage: "Sorry, we are currently unable to perform the request. Please try again later.",
			}
		}
		if len(cConf.Servers) == 0 {
			return "", happydns.InternalError{Err: errors.New("no resolver in /etc/resolv.conf")}
		}
		return net.JoinHostPort(cConf.Servers[rand.Intn(len(cConf.Servers))], "53"), nil
	}

	ctx, cancel := context.WithTimeout(ctx, resolverLookupTimeout)
	defer cancel()

	// ResolveAddrPort hands back an IP literal, never the name it was given:
	// miekg/dns would otherwise resolve it a second time, and the second answer
	// is the one an attacker gets to choose.
	target, err := ru.resolverGuard.ResolveAddrPort(ctx, resolver, 53)
	if err != nil {
		// Our own resolver failing to answer says nothing about the server the
		// caller asked for, so it must not be reported as a refused one.
		if errors.Is(err, netguard.ErrTemporary) {
			return "", happydns.InternalError{
				Err:         fmt.Errorf("unable to check the requested resolver: %w", err),
				UserMessage: ru.resolverGuard.Unavailable("The requested DNS server"),
			}
		}

		return "", happydns.ValidationError{Msg: ru.resolverGuard.Refusal("The requested DNS server")}
	}

	return target, nil
}
