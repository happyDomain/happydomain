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

// Package netguard decides whether happyDomain may dial a destination the
// caller chose.
//
// Several endpoints take their destination from the request: the DNS provider
// endpoints, the certificate fetcher, the MTA-STS policy fetcher, the public
// resolver and the notification webhooks. Unchecked, they reach whatever the
// server itself can reach: services behind the firewall, the admin socket on
// loopback, or the cloud metadata endpoint at 169.254.169.254. The provider arm
// is the worst of them, as it carries the configured API key to the address it
// is given.
//
// A Guard refuses every address that is not globally routable. Operators
// re-open the ranges their deployment genuinely needs (a co-located
// authoritative server, a router on the LAN) through an allow-list of IP
// addresses and CIDR blocks.
//
// This package deliberately depends on nothing but the standard library. It is
// a security primitive: everything that dials on a user's behalf must be able
// to import it without dragging in the rest of happyDomain.
package netguard // import "git.happydns.org/happyDomain/internal/netguard"

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/netip"
	"strings"
)

// ErrBlocked is the sentinel every refusal wraps. Its own message names
// nothing: no host, no address, no port. Callers may surface it as is.
var ErrBlocked = errors.New("the requested address is not allowed by this happyDomain instance")

// ErrTemporary marks a check that could not be carried out, as opposed to one
// that was carried out and refused: a resolver timeout, a SERVFAIL, a cancelled
// request. The destination may well be perfectly acceptable, so callers must
// not tell the user their address is forbidden, nor treat the failure as a
// reason to reject the value they typed.
var ErrTemporary = errors.New("the requested address could not be resolved at the moment")

// bogons are the ranges the net/netip predicates do not report. netip.Addr
// already covers loopback, unspecified, multicast, link-local (169.254.0.0/16
// and fe80::/10) and private (10/8, 172.16/12, 192.168/16, fc00::/7).
var bogons = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),       // "this network", routes to the local host
	netip.MustParsePrefix("100.64.0.0/10"),   // RFC 6598 carrier-grade NAT
	netip.MustParsePrefix("192.0.0.0/24"),    // IETF protocol assignments
	netip.MustParsePrefix("192.0.2.0/24"),    // documentation
	netip.MustParsePrefix("198.18.0.0/15"),   // benchmarking
	netip.MustParsePrefix("198.51.100.0/24"), // documentation
	netip.MustParsePrefix("203.0.113.0/24"),  // documentation
	netip.MustParsePrefix("240.0.0.0/4"),     // reserved, and covers 255.255.255.255

	netip.MustParsePrefix("64:ff9b::/96"),   // NAT64 well-known: embeds an IPv4 address
	netip.MustParsePrefix("64:ff9b:1::/48"), // NAT64 local-use: embeds an IPv4 address
	netip.MustParsePrefix("100::/64"),       // discard-only
	netip.MustParsePrefix("2001::/32"),      // Teredo: embeds an IPv4 address
	netip.MustParsePrefix("2001:db8::/32"),  // documentation
	netip.MustParsePrefix("2002::/16"),      // 6to4: embeds an IPv4 address
}

// IsGloballyRoutable reports whether addr is a public unicast address, that is
// one that could not possibly be an internal service.
//
// Teredo, 6to4 and NAT64 are refused because they embed an IPv4 address: they
// are a documented way to smuggle a private target past a table that only
// knows about IPv4 ranges.
func IsGloballyRoutable(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}

	// An IPv4-mapped IPv6 literal (::ffff:10.0.0.1) is the same host as its
	// IPv4 form, but netip.Addr predicates, unlike net.IP ones, do not see
	// through the mapping. Unmapping first is what makes them meaningful.
	addr = addr.Unmap()

	if addr.IsUnspecified() || addr.IsLoopback() || addr.IsMulticast() ||
		addr.IsInterfaceLocalMulticast() || addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() || addr.IsPrivate() {
		return false
	}

	for _, prefix := range bogons {
		if prefix.Contains(addr) {
			return false
		}
	}

	return true
}

// Guard is one outbound destination policy. Two instances exist at runtime: one
// for the unauthenticated resolver endpoints, one for everything else.
//
// A nil *Guard applies the default policy with an empty allow-list, so a caller
// that was never wired one still refuses internal destinations rather than
// dialing them.
type Guard struct {
	// name identifies the guard in the operator-facing log lines.
	name string

	// flagName is the option an operator sets to allow more. It is quoted back
	// to the user in the refusal, so they know what to ask for.
	flagName string

	allowed []netip.Prefix

	// lookupIP is swapped in tests. Nil means the system resolver.
	lookupIP func(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// New builds a Guard from an allow-list of IP addresses and CIDR blocks.
//
// Entries are parsed here as well as at flag-parsing time, not instead of it: a
// Guard silently built from a list it could not read would allow something
// other than what the operator wrote.
func New(name, flagName string, entries []string) (*Guard, error) {
	g := &Guard{name: name, flagName: flagName}

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if strings.Contains(entry, "/") {
			prefix, err := netip.ParsePrefix(entry)
			if err != nil {
				return nil, fmt.Errorf("%q is not a valid CIDR block: %w", entry, err)
			}
			g.allowed = append(g.allowed, prefix.Masked())
			continue
		}

		addr, err := netip.ParseAddr(entry)
		if err != nil {
			return nil, fmt.Errorf("%q is neither a valid IP address nor a CIDR block", entry)
		}
		addr = addr.Unmap()
		g.allowed = append(g.allowed, netip.PrefixFrom(addr, addr.BitLen()))
	}

	return g, nil
}

// Flag returns the configuration option feeding this guard.
func (g *Guard) Flag() string {
	if g == nil || g.flagName == "" {
		return "-outbound-allowed-target"
	}
	return g.flagName
}

// Refusal is the message shown to the user when subject was blocked. It names
// the field they filled and the option their administrator can set, and
// nothing else: not the address, not the port, not the reason.
//
// Uniform wording is the point. The outbound arms would otherwise answer "is
// something listening at this address and port?" for anyone who asks, so a
// blocked range, a name that does not resolve and a refused connection have to
// be indistinguishable from outside.
func (g *Guard) Refusal(subject string) string {
	return fmt.Sprintf("%s does not resolve to an address this happyDomain instance is allowed to reach. Only public addresses are reachable; ask your administrator to allow it with %s.", subject, g.Flag())
}

// Unavailable is the message shown when the check could not be completed. It
// says nothing about the destination, only that we failed to look it up, and
// invites a retry: the value the user typed may be fine.
func (g *Guard) Unavailable(subject string) string {
	return fmt.Sprintf("%s could not be checked: this happyDomain instance failed to resolve it in time. Please try again in a moment.", subject)
}

// isTemporaryLookupFailure tells a resolver that did not answer from one that
// answered "no such host".
//
// Only the latter is a fact about the destination, and only the latter may be
// reported with the uniform refusal wording: everything else is a fact about
// our own resolver, and repeating the refusal there tells the user to fix an
// address that was never the problem.
func isTemporaryLookupFailure(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return !dnsErr.IsNotFound
	}

	return false
}

// Describe renders the policy for the startup log.
func (g *Guard) Describe() string {
	if g == nil || len(g.allowed) == 0 {
		return "public addresses only"
	}

	entries := make([]string, 0, len(g.allowed))
	for _, prefix := range g.allowed {
		entries = append(entries, prefix.String())
	}
	return "public addresses, plus " + strings.Join(entries, ", ")
}

// AllowsAddr is the single policy decision point.
func (g *Guard) AllowsAddr(addr netip.Addr) bool {
	if !addr.IsValid() {
		return false
	}

	if g != nil {
		unmapped := addr.Unmap()
		for _, prefix := range g.allowed {
			if prefix.Contains(unmapped) {
				return true
			}
		}
	}

	return IsGloballyRoutable(addr)
}

// AllowsIP is AllowsAddr for callers that still hold a net.IP.
func (g *Guard) AllowsIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	return g.AllowsAddr(addr)
}

// ResolveAllowed resolves host and returns its addresses, provided every one of
// them is allowed. host may be a name or an IP literal, with or without the
// brackets an IPv6 literal is usually written with.
//
// Every address must pass, not just the one we would end up dialing: a name
// that resolves to both a public and a private address is a rebinding attempt,
// and which of the two the dialer picks is not ours to decide.
func (g *Guard) ResolveAllowed(ctx context.Context, host string) ([]netip.Addr, error) {
	if host == "" {
		return nil, fmt.Errorf("no host to reach: %w", ErrBlocked)
	}

	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")

	if addr, err := netip.ParseAddr(host); err == nil {
		if !g.AllowsAddr(addr) {
			return nil, g.blocked("%s is not an allowed destination", addr)
		}
		return []netip.Addr{addr.Unmap()}, nil
	}

	// A nil *Guard has no resolver of its own, and reading the field would
	// panic instead of applying the default policy the type documents.
	var lookup func(ctx context.Context, network, host string) ([]netip.Addr, error)
	if g != nil {
		lookup = g.lookupIP
	}
	if lookup == nil {
		lookup = net.DefaultResolver.LookupNetIP
	}

	addrs, err := lookup(ctx, "ip", host)
	if err != nil {
		if isTemporaryLookupFailure(err) {
			return nil, fmt.Errorf("unable to resolve %q: %w: %w", host, err, ErrTemporary)
		}
		return nil, fmt.Errorf("unable to resolve %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("%q does not resolve to any address", host)
	}

	for _, addr := range addrs {
		if !g.AllowsAddr(addr) {
			return nil, g.blocked("%s resolves to %s, which is not an allowed destination", host, addr)
		}
	}

	return addrs, nil
}

// ResolveAddrPort checks a destination and returns it as a dialable "ip:port",
// pinned to an address the guard accepted. defaultPort applies when hostport
// carries none.
//
// It exists for the callers that cannot be handed a dialer: github.com/miekg/dns
// dials through a plain net.Dialer with no post-resolution hook, so the only
// way to be sure it talks to the address we vetted is to give it an address
// rather than a name.
func (g *Guard) ResolveAddrPort(ctx context.Context, hostport string, defaultPort uint16) (string, error) {
	host, port := splitHostPort(hostport, defaultPort)

	addrs, err := g.ResolveAllowed(ctx, host)
	if err != nil {
		return "", err
	}

	return net.JoinHostPort(addrs[0].String(), port), nil
}

// splitHostPort separates an optional port from a destination that may be a
// bare name, a bare IPv4 or IPv6 literal, or either of them with a port.
//
// net.SplitHostPort alone cannot do this: it fails on "2001:db8::1" (too many
// colons) and, worse, happily reads "1.2.3.4:5353" as host and port, which is
// what we want, but reads nothing sensible out of "example.com".
func splitHostPort(hostport string, defaultPort uint16) (host, port string) {
	port = fmt.Sprintf("%d", defaultPort)

	if h, p, err := net.SplitHostPort(hostport); err == nil {
		return h, p
	}

	// No port, or an unbracketed IPv6 literal. Telling them apart is exactly
	// what ParseAddr does.
	return strings.TrimSuffix(strings.TrimPrefix(hostport, "["), "]"), port
}

// blocked builds the internal error and records the real reason, which the
// caller must not surface. This log line is the operator's copy-paste source
// for the allow-list, so it names the host and the address on purpose.
func (g *Guard) blocked(format string, args ...any) error {
	name := "outbound"
	if g != nil && g.name != "" {
		name = g.name
	}

	reason := fmt.Sprintf(format, args...)
	log.Printf("netguard[%s]: %s (allow it with %s if this is intended)", name, reason, g.Flag())

	return fmt.Errorf("%s: %w", reason, ErrBlocked)
}
