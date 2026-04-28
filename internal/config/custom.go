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
