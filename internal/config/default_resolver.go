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

package config // import "git.happydns.org/happyDomain/config"

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/miekg/dns"
)

// resolverList builds the flag.Value behind -default-resolver.
func resolverList(values *[]string) *entryList {
	return &entryList{
		stringSlice: stringSlice{values},
		validate:    canonicalResolver,
	}
}

// canonicalResolver turns an IP address, with or without a port, into the
// "ip:port" to dial, port 53 by default. A host name is refused: resolving it
// would need the very resolver being configured.
func canonicalResolver(item string) (string, error) {
	if addrPort, err := netip.ParseAddrPort(item); err == nil {
		return addrPort.String(), nil
	}

	addr, err := netip.ParseAddr(strings.TrimSuffix(strings.TrimPrefix(item, "["), "]"))
	if err != nil {
		return "", fmt.Errorf("default resolver %q is not an IP address, with an optional port", item)
	}
	return netip.AddrPortFrom(addr, 53).String(), nil
}

// systemResolversFrom returns the "ip:port" of every resolver of the given
// resolv.conf, in the order set there.
func systemResolversFrom(path string) ([]string, error) {
	conf, err := dns.ClientConfigFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read the system resolvers: %w", err)
	}
	if len(conf.Servers) == 0 {
		return nil, errors.New("no resolver in " + path)
	}

	servers := make([]string, len(conf.Servers))
	for i, server := range conf.Servers {
		servers[i] = net.JoinHostPort(server, conf.Port)
	}
	return servers, nil
}
