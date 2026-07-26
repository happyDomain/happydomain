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

package netguard // import "git.happydns.org/happyDomain/internal/netguard"

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"testing"
)

func TestIsGloballyRoutable(t *testing.T) {
	tests := []struct {
		addr string
		want bool
		why  string
	}{
		{addr: "1.1.1.1", want: true},
		{addr: "9.9.9.9", want: true},
		{addr: "2606:4700:4700::1111", want: true},

		{addr: "127.0.0.1", why: "loopback"},
		{addr: "::1", why: "loopback"},
		{addr: "0.0.0.0", why: "unspecified"},
		{addr: "::", why: "unspecified"},
		{addr: "10.0.0.1", why: "RFC1918"},
		{addr: "172.16.0.1", why: "RFC1918"},
		{addr: "192.168.1.1", why: "RFC1918"},
		{addr: "169.254.169.254", why: "link-local: the cloud metadata endpoint"},
		{addr: "fe80::1", why: "link-local"},
		{addr: "fc00::1", why: "unique local"},
		{addr: "fd00::1", why: "unique local"},
		{addr: "ff02::1", why: "multicast"},
		{addr: "224.0.0.1", why: "multicast"},
		{addr: "100.64.0.1", why: "carrier-grade NAT"},
		{addr: "0.1.2.3", why: "0.0.0.0/8 routes to the local host"},
		{addr: "192.0.0.170", why: "IETF protocol assignments"},
		{addr: "192.0.2.1", why: "documentation range"},
		{addr: "198.18.0.1", why: "benchmarking range"},
		{addr: "203.0.113.1", why: "documentation range"},
		{addr: "240.0.0.1", why: "reserved"},
		{addr: "255.255.255.255", why: "broadcast"},
		{addr: "2001:db8::1", why: "documentation range"},
		{addr: "100::1", why: "discard-only"},

		// These are the ones a v4-only table misses.
		{addr: "::ffff:127.0.0.1", why: "IPv4-mapped loopback"},
		{addr: "::ffff:10.0.0.1", why: "IPv4-mapped RFC1918"},
		{addr: "::ffff:169.254.169.254", why: "IPv4-mapped metadata endpoint"},
		{addr: "2002:7f00:1::", why: "6to4 wrapping 127.0.0.1"},
		{addr: "2001:0:1:2:3:4:5:6", why: "Teredo embeds an IPv4 address"},
		{addr: "64:ff9b::a9fe:a9fe", why: "NAT64 wrapping the metadata endpoint"},
		{addr: "64:ff9b:1::7f00:1", why: "NAT64 local-use wrapping 127.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			addr := netip.MustParseAddr(tt.addr)
			if got := IsGloballyRoutable(addr); got != tt.want {
				t.Errorf("IsGloballyRoutable(%s) = %v, want %v (%s)", tt.addr, got, tt.want, tt.why)
			}
		})
	}
}

func TestIsGloballyRoutableInvalid(t *testing.T) {
	if IsGloballyRoutable(netip.Addr{}) {
		t.Error("IsGloballyRoutable(zero Addr) = true, want false")
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		wantErr bool
	}{
		{name: "empty list", entries: nil},
		{name: "single address", entries: []string{"127.0.0.1"}},
		{name: "CIDR block", entries: []string{"10.0.0.0/8"}},
		{name: "IPv6 block", entries: []string{"fd00::/8"}},
		{name: "blank entries are skipped", entries: []string{"", "  "}},
		{name: "not an address", entries: []string{"example.com"}, wantErr: true},
		{name: "not a block", entries: []string{"10.0.0.0/99"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New("test", "-test", tt.entries)
			if (err != nil) != tt.wantErr {
				t.Fatalf("New(%v) error = %v, wantErr %v", tt.entries, err, tt.wantErr)
			}
		})
	}
}

func TestGuardAllowsAddr(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		addr    string
		want    bool
	}{
		{name: "loopback refused by default", addr: "127.0.0.1"},
		{name: "loopback allowed when listed", entries: []string{"127.0.0.1"}, addr: "127.0.0.1", want: true},
		{name: "loopback allowed through a block", entries: []string{"127.0.0.0/8"}, addr: "127.0.0.53", want: true},
		{name: "a listed host does not allow its neighbours", entries: []string{"10.0.0.1"}, addr: "10.0.0.2"},
		{name: "private block", entries: []string{"192.168.0.0/16"}, addr: "192.168.1.1", want: true},
		{name: "outside the listed block", entries: []string{"192.168.0.0/16"}, addr: "10.0.0.1"},
		{name: "IPv6 block", entries: []string{"fd00::/8"}, addr: "fd12::1", want: true},
		{name: "public stays allowed", entries: []string{"127.0.0.1"}, addr: "1.1.1.1", want: true},

		// Listing the IPv4 form must cover the mapped form, otherwise the
		// allow-list and the block-list disagree about what an address is.
		{name: "mapped form of a listed address", entries: []string{"127.0.0.1"}, addr: "::ffff:127.0.0.1", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := New("test", "-test", tt.entries)
			if err != nil {
				t.Fatalf("New(%v) => %v", tt.entries, err)
			}

			if got := g.AllowsAddr(netip.MustParseAddr(tt.addr)); got != tt.want {
				t.Errorf("AllowsAddr(%s) with %v = %v, want %v", tt.addr, tt.entries, got, tt.want)
			}
		})
	}
}

func TestNilGuardRefusesPrivate(t *testing.T) {
	var g *Guard

	if g.AllowsAddr(netip.MustParseAddr("127.0.0.1")) {
		t.Error("a nil Guard allowed loopback; it must apply the default policy, not disable it")
	}
	if !g.AllowsAddr(netip.MustParseAddr("1.1.1.1")) {
		t.Error("a nil Guard refused a public address")
	}

	if _, err := g.ResolveAllowed(context.Background(), "127.0.0.1"); err == nil {
		t.Error("a nil Guard resolved loopback; it must apply the default policy")
	}

	// A name sends ResolveAllowed down the resolver path, where reading the
	// guard's own lookup function used to panic. The cancelled context keeps
	// the test off the network: what matters is that it returns rather than
	// panics.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := g.ResolveAllowed(ctx, "example.com"); err == nil {
		t.Error("a nil Guard resolved a name with a cancelled context")
	}
}

// withLookup returns a Guard resolving names from a static table, so the tests
// never depend on real DNS.
func withLookup(t *testing.T, entries []string, table map[string][]string) *Guard {
	t.Helper()

	g, err := New("test", "-test", entries)
	if err != nil {
		t.Fatalf("New(%v) => %v", entries, err)
	}

	g.lookupIP = func(_ context.Context, _, host string) ([]netip.Addr, error) {
		values, ok := table[host]
		if !ok {
			return nil, fmt.Errorf("no such host: %s", host)
		}

		addrs := make([]netip.Addr, 0, len(values))
		for _, v := range values {
			addrs = append(addrs, netip.MustParseAddr(v))
		}
		return addrs, nil
	}

	return g
}

func TestResolveAllowed(t *testing.T) {
	table := map[string][]string{
		"public.example.com":  {"1.1.1.1"},
		"private.example.com": {"10.0.0.1"},
		"mixed.example.com":   {"1.1.1.1", "127.0.0.1"},
		"empty.example.com":   {},
	}

	tests := []struct {
		name      string
		entries   []string
		host      string
		wantErr   bool
		wantBlock bool
		wantFirst string
	}{
		{name: "public name", host: "public.example.com", wantFirst: "1.1.1.1"},
		{name: "private name", host: "private.example.com", wantErr: true, wantBlock: true},
		{name: "private name, allowed", entries: []string{"10.0.0.0/8"}, host: "private.example.com", wantFirst: "10.0.0.1"},
		{
			// One private answer poisons the whole set: which address the
			// dialer would pick is not ours to decide.
			name: "a single private answer refuses the name", host: "mixed.example.com", wantErr: true, wantBlock: true,
		},
		{name: "IP literal", host: "1.1.1.1", wantFirst: "1.1.1.1"},
		{name: "bracketed IPv6 literal", host: "[2606:4700::1111]", wantFirst: "2606:4700::1111"},
		{name: "private IP literal", host: "169.254.169.254", wantErr: true, wantBlock: true},
		{name: "empty host", host: "", wantErr: true, wantBlock: true},
		{name: "unresolvable name", host: "nope.example.com", wantErr: true},
		{name: "name without addresses", host: "empty.example.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := withLookup(t, tt.entries, table)

			addrs, err := g.ResolveAllowed(t.Context(), tt.host)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveAllowed(%q) error = %v, wantErr %v", tt.host, err, tt.wantErr)
			}
			if tt.wantBlock && !errors.Is(err, ErrBlocked) {
				t.Fatalf("ResolveAllowed(%q) error = %v, want it to wrap ErrBlocked", tt.host, err)
			}
			if tt.wantErr {
				return
			}

			if len(addrs) == 0 || addrs[0].String() != tt.wantFirst {
				t.Errorf("ResolveAllowed(%q) = %v, want it to start with %s", tt.host, addrs, tt.wantFirst)
			}
		})
	}
}

// A resolver that did not answer says nothing about the destination, so it
// must not be reported the way a refused address is.
func TestResolveAllowedTemporaryFailure(t *testing.T) {
	tests := []struct {
		name          string
		lookupErr     error
		wantTemporary bool
	}{
		{name: "timeout", lookupErr: &net.DNSError{Err: "i/o timeout", Name: "ns.example.com", IsTimeout: true}, wantTemporary: true},
		{name: "server failure", lookupErr: &net.DNSError{Err: "server misbehaving", Name: "ns.example.com"}, wantTemporary: true},
		{name: "deadline exceeded", lookupErr: context.DeadlineExceeded, wantTemporary: true},
		{name: "no such host", lookupErr: &net.DNSError{Err: "no such host", Name: "ns.example.com", IsNotFound: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := withLookup(t, nil, nil)
			g.lookupIP = func(context.Context, string, string) ([]netip.Addr, error) {
				return nil, tt.lookupErr
			}

			_, err := g.ResolveAllowed(t.Context(), "ns.example.com")
			if err == nil {
				t.Fatal("ResolveAllowed succeeded on a failing resolver")
			}
			if got := errors.Is(err, ErrTemporary); got != tt.wantTemporary {
				t.Errorf("errors.Is(%v, ErrTemporary) = %v, want %v", err, got, tt.wantTemporary)
			}
			if errors.Is(err, ErrBlocked) {
				t.Errorf("ResolveAllowed(%v) reported a resolver failure as a policy refusal", tt.lookupErr)
			}
		})
	}
}

func TestResolveAddrPort(t *testing.T) {
	table := map[string][]string{
		"ns.example.com": {"1.1.1.1"},
	}

	tests := []struct {
		name    string
		host    string
		want    string
		wantErr bool
	}{
		{name: "bare address takes the default port", host: "1.1.1.1", want: "1.1.1.1:53"},
		{name: "explicit port is kept", host: "1.1.1.1:5353", want: "1.1.1.1:5353"},
		{name: "name is replaced by its address", host: "ns.example.com", want: "1.1.1.1:53"},
		{name: "name with a port", host: "ns.example.com:5353", want: "1.1.1.1:5353"},
		{name: "bare IPv6 literal", host: "2606:4700::1111", want: "[2606:4700::1111]:53"},
		{name: "bracketed IPv6 literal with a port", host: "[2606:4700::1111]:5353", want: "[2606:4700::1111]:5353"},
		{name: "loopback", host: "127.0.0.1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := withLookup(t, nil, table)

			got, err := g.ResolveAddrPort(t.Context(), tt.host, 53)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveAddrPort(%q) error = %v, wantErr %v", tt.host, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("ResolveAddrPort(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	table := map[string][]string{
		"public.example.com":  {"1.1.1.1"},
		"private.example.com": {"10.0.0.1"},
	}

	tests := []struct {
		name    string
		entries []string
		url     string
		wantErr bool
	}{
		{name: "public https", url: "https://public.example.com/hook"},
		{name: "public http", url: "http://public.example.com:8080/hook"},
		{name: "unsupported scheme", url: "ftp://public.example.com/", wantErr: true},
		{name: "file scheme", url: "file:///etc/passwd", wantErr: true},
		{name: "no host", url: "http:///path", wantErr: true},
		{name: "private name", url: "https://private.example.com/", wantErr: true},
		{name: "private name, allowed", entries: []string{"10.0.0.0/8"}, url: "https://private.example.com/"},
		{name: "loopback literal", url: "http://127.0.0.1:8081/api", wantErr: true},
		{name: "loopback literal, allowed", entries: []string{"127.0.0.1"}, url: "http://127.0.0.1:8081/api"},
		{name: "metadata endpoint", url: "http://169.254.169.254/latest/meta-data/", wantErr: true},
		{name: "bracketed loopback", url: "http://[::1]:8081/", wantErr: true},
		{name: "credentials do not hide the host", url: "http://user:pass@127.0.0.1/", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := withLookup(t, tt.entries, table)

			_, err := g.ValidateURL(t.Context(), tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestHostFromEndpoint(t *testing.T) {
	tests := []struct {
		endpoint string
		want     string
		wantErr  bool
	}{
		{endpoint: "https://pdns.example.com:8081/api", want: "pdns.example.com"},
		{endpoint: "http://127.0.0.1:3000", want: "127.0.0.1"},
		{endpoint: "http://user:pass@10.0.0.1/api", want: "10.0.0.1"},
		{endpoint: "https://[::1]:8443/api", want: "::1"},
		{endpoint: "192.168.88.1:8080", want: "192.168.88.1"},
		{endpoint: "192.168.88.1", want: "192.168.88.1"},
		{endpoint: "pdns.example.com", want: "pdns.example.com"},
		{endpoint: "pdns.example.com/api", want: "pdns.example.com"},
		{endpoint: "[::1]", want: "::1"},
		{endpoint: "[::1]:53", want: "::1"},
		{endpoint: "2001:db8::1", want: "2001:db8::1"},
		// A symbolic region name is not a host, but it must not error out
		// either: refusing to parse a value means refusing to check it.
		{endpoint: "ovh-eu", want: "ovh-eu"},
		{endpoint: "", wantErr: true},
		{endpoint: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			got, err := HostFromEndpoint(tt.endpoint)
			if (err != nil) != tt.wantErr {
				t.Fatalf("HostFromEndpoint(%q) error = %v, wantErr %v", tt.endpoint, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("HostFromEndpoint(%q) = %q, want %q", tt.endpoint, got, tt.want)
			}
		})
	}
}

func TestDescribe(t *testing.T) {
	var nilGuard *Guard
	if got := nilGuard.Describe(); got != "public addresses only" {
		t.Errorf("(*Guard)(nil).Describe() = %q", got)
	}

	g, err := New("test", "-test", []string{"127.0.0.1", "10.0.0.0/8"})
	if err != nil {
		t.Fatalf("New() => %v", err)
	}
	if got, want := g.Describe(), "public addresses, plus 127.0.0.1/32, 10.0.0.0/8"; got != want {
		t.Errorf("Describe() = %q, want %q", got, want)
	}
}
