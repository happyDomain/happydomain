// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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
	"encoding/base64"
	"fmt"
	"net/mail"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"git.happydns.org/happyDomain/model"
)

// stringSlice is a flag.Value that accumulates string values across repeated
// invocations of the same flag (e.g. -plugins-directory a -plugins-directory b).
type stringSlice struct {
	Values *[]string
}

func (s *stringSlice) String() string {
	if s.Values == nil {
		return ""
	}
	return strings.Join(*s.Values, ",")
}

func (s *stringSlice) Set(value string) error {
	*s.Values = append(*s.Values, value)
	return nil
}

// proxyList is a flag.Value that accumulates IP addresses and CIDR blocks of
// trusted reverse proxies. It accepts both repeated flags and a single comma
// separated value, as an environment variable can only be given once. Entries
// are validated and canonicalized eagerly, so a typo fails at startup instead
// of silently disabling the trust check, or silently trusting more than the
// operator wrote.
//
// The keyword `none` empties the list. Because the config file, the
// environment and the command line all feed this same flag.Value, a
// lower-precedence source can otherwise only ever be widened: `none` is what
// makes a trust list narrowable from the command line.
type proxyList struct {
	stringSlice
}

func (p *proxyList) Set(value string) error {
	for item := range strings.SplitSeq(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		if strings.EqualFold(item, "none") {
			*p.Values = nil
			continue
		}

		// `local` used to expand to loopback plus every private range. That
		// trusts the whole network the proxy sits on, not the proxy: on a
		// container bridge or a LAN, any neighbour reaching the port directly
		// could then pick the address it was throttled under. Point at the
		// replacement rather than letting it fail as an unparseable address.
		if strings.EqualFold(item, "local") {
			return fmt.Errorf("%q is no longer accepted: it trusted every private range, so any host on the same network could forge X-Forwarded-For. Give the exact address of your proxy instead (eg. the container IP, or 127.0.0.1 for a proxy on the same host)", item)
		}

		entry, err := canonicalProxyEntry(item)
		if err != nil {
			return err
		}

		*p.Values = append(*p.Values, entry)
	}

	return nil
}

// canonicalProxyEntry validates one trusted proxy entry and returns the form
// gin will actually match against, so that what the startup log reports is
// what is really trusted.
//
// Two inputs are accepted by the naive parsers but mean something else to gin,
// and both silently widen or misplace the trust boundary, so they are refused
// rather than reinterpreted:
//
//   - a prefix with host bits set (`192.0.2.5/24`), which masks down to a whole
//     subnet while reading like a single host;
//   - an IPv4-mapped IPv6 literal (`::ffff:192.0.2.1`), which gin turns into
//     `::/32`, leaving the intended proxy untrusted and `::1` trusted instead.
func canonicalProxyEntry(item string) (string, error) {
	if strings.Contains(item, "/") {
		prefix, err := netip.ParsePrefix(item)
		if err != nil {
			return "", fmt.Errorf("%q is not a valid CIDR block: %w", item, err)
		}

		if masked := prefix.Masked(); masked != prefix {
			return "", fmt.Errorf("%q has bits set outside its /%d prefix: write %q to trust the whole block, or %q to trust that single address", item, prefix.Bits(), masked, prefix.Addr())
		}

		if prefix.Addr().Is4In6() {
			return "", fmt.Errorf("%q is an IPv4-mapped IPv6 block: write it as an IPv4 block instead", item)
		}

		return prefix.String(), nil
	}

	addr, err := netip.ParseAddr(item)
	if err != nil {
		return "", fmt.Errorf("%q is neither a valid IP address nor a CIDR block", item)
	}

	if addr.Is4In6() {
		return "", fmt.Errorf("%q is an IPv4-mapped IPv6 address: write it as %q instead", item, addr.Unmap())
	}

	return addr.String(), nil
}

// checkerOptionFlag is a flag.Value that writes the parsed flag value into a
// per-checker happydns.CheckerOptions map under a preset Key, converting the
// raw input string according to the option's declared CheckerOptionField.Type.
// The map must already exist in the parent Options map; the indirection is
// intentional so multiple flags share the same backing CheckerOptions value.
type checkerOptionFlag struct {
	Opts happydns.CheckerOptions
	Key  string
	Type string
}

func (c *checkerOptionFlag) String() string {
	if c.Opts == nil {
		return ""
	}
	v, ok := c.Opts[c.Key]
	if !ok {
		return ""
	}
	return fmt.Sprint(v)
}

func (c *checkerOptionFlag) Set(value string) error {
	parsed, err := parseCheckerOptionValue(c.Type, value)
	if err != nil {
		return fmt.Errorf("option %q: %w", c.Key, err)
	}
	c.Opts[c.Key] = parsed
	return nil
}

// parseCheckerOptionValue converts a CLI/env string into the type expected by
// the checker, mirroring how JSON-decoded option values arrive at runtime
// (numbers as float64, booleans as bool, everything else as string).
func parseCheckerOptionValue(typ, value string) (any, error) {
	switch {
	case typ == "bool" || typ == "boolean":
		return strconv.ParseBool(value)
	case typ == "number",
		strings.HasPrefix(typ, "int"),
		strings.HasPrefix(typ, "uint"),
		strings.HasPrefix(typ, "float"):
		return strconv.ParseFloat(value, 64)
	default:
		return value, nil
	}
}

type JWTSecretKey struct {
	Secret *[]byte
}

func (i *JWTSecretKey) String() string {
	if i.Secret == nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(*i.Secret)
}

func (i *JWTSecretKey) Set(value string) error {
	z, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return err
	}

	*i.Secret = z
	return nil
}

// mailAddress defines an interface that handle mail.Address configuration
// throught custom flag.
type mailAddress struct {
	*mail.Address
}

func (i *mailAddress) String() string {
	if i.Address == nil {
		return ""
	}
	return i.Address.String()
}

func (i *mailAddress) Set(value string) error {
	v, err := mail.ParseAddress(value)
	if err != nil {
		return err
	}
	*i.Address = *v
	return nil
}

type URL struct {
	URL *url.URL
}

func (i *URL) String() string {
	if i.URL != nil {
		return i.URL.String()
	} else {
		return ""
	}
}

func (i *URL) Set(value string) error {
	u, err := url.Parse(value)
	if err != nil {
		return err
	}

	*i.URL = *u
	return nil
}
