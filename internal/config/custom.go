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
	"git.happydns.org/happyDomain/pkg/favicon"
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

// entryList is a flag.Value that accumulates validated string entries, in the
// order given. It accepts both repeated flags and a single comma separated
// value, as an environment variable can only be given once. Each entry is
// passed through validate, which checks it and returns the canonical form to
// store, so a typo fails at startup instead of silently widening or disabling
// whatever the list feeds.
//
// The keyword `none` empties the list. Because the config file, the
// environment and the command line all feed this same flag.Value, a
// lower-precedence source can otherwise only ever be widened: `none` is what
// makes such a list narrowable from the command line. It sets Values to a
// non-nil empty slice rather than nil, so callers that apply a default only
// when the list is nil can still tell "explicitly emptied" apart from
// "never set".
type entryList struct {
	stringSlice

	// validate checks and canonicalizes one entry.
	validate func(item string) (string, error)
}

func (p *entryList) Set(value string) error {
	for item := range strings.SplitSeq(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		if strings.EqualFold(item, "none") {
			*p.Values = []string{}
			continue
		}

		entry, err := p.validate(item)
		if err != nil {
			return err
		}

		*p.Values = append(*p.Values, entry)
	}

	return nil
}

// canonicalIPEntry validates one entry and returns the form that will actually
// be matched against, so that what the startup log reports is what really
// applies.
//
// Two inputs are accepted by the naive parsers but mean something else to the
// matchers, and both silently widen or misplace the boundary, so they are
// refused rather than reinterpreted:
//
//   - a prefix with host bits set (`198.51.100.5/24`), which masks down to a
//     whole subnet while reading like a single host;
//   - an IPv4-mapped IPv6 literal (`::ffff:198.51.100.1`), which gin turns into
//     `::/32`, leaving the intended address out and `::1` in instead.
func canonicalIPEntry(item, what string) (string, error) {
	if strings.Contains(item, "/") {
		prefix, err := netip.ParsePrefix(item)
		if err != nil {
			return "", fmt.Errorf("%s %q is not a valid CIDR block: %w", what, item, err)
		}

		if masked := prefix.Masked(); masked != prefix {
			return "", fmt.Errorf("%s %q has bits set outside its /%d prefix: write %q for the whole block, or %q for that single address", what, item, prefix.Bits(), masked, prefix.Addr())
		}

		if prefix.Addr().Is4In6() {
			return "", fmt.Errorf("%s %q is an IPv4-mapped IPv6 block: write it as an IPv4 block instead", what, item)
		}

		return prefix.String(), nil
	}

	addr, err := netip.ParseAddr(item)
	if err != nil {
		return "", fmt.Errorf("%s %q is neither a valid IP address nor a CIDR block", what, item)
	}

	if addr.Is4In6() {
		return "", fmt.Errorf("%s %q is an IPv4-mapped IPv6 address: write it as %q instead", what, item, addr.Unmap())
	}

	return addr.String(), nil
}

// proxyList builds the flag.Value behind -trusted-proxy.
func proxyList(values *[]string) *entryList {
	return &entryList{
		stringSlice: stringSlice{values},
		validate: func(item string) (string, error) {
			return canonicalIPEntry(item, "trusted proxy")
		},
	}
}

// targetList builds the flag.Value behind the two outbound allow-lists.
func targetList(values *[]string, what string) *entryList {
	return &entryList{
		stringSlice: stringSlice{values},
		validate: func(item string) (string, error) {
			return canonicalIPEntry(item, what)
		},
	}
}

// faviconSourceList builds the flag.Value behind -favicon-source. Names are
// validated here so that a typo stops happyDomain at startup: left to the
// first request, an unknown source would show up as icons missing, which
// looks exactly like a network problem.
func faviconSourceList(values *[]string) *entryList {
	return &entryList{
		stringSlice: stringSlice{values},
		validate: func(item string) (string, error) {
			return item, favicon.ValidateSourceName(item)
		},
	}
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
