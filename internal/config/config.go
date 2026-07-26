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
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net/mail"
	"net/url"
	"os"
	"path"
	"strings"

	"git.happydns.org/happyDomain/model"
)

// ConsolidateConfig fills an Options struct by reading configuration from
// config files, environment, then command line.
//
// Should be called only one time.
func ConsolidateConfig() (opts *happydns.Options, err error) {
	u, _ := url.Parse("http://localhost:8081")

	// Define defaults options
	opts = &happydns.Options{
		AdminBind:         "./happydomain.sock",
		BasePath:          "/",
		Bind:              ":8081",
		DefaultNameServer: "127.0.0.1:53",
		ExternalURL:       *u,
		JWTSigningMethod:  "HS512",
		MailFrom:          mail.Address{Name: "happyDomain", Address: "happydomain@localhost"},
		MailSMTPPort:      587,
		StorageEngine:     "leveldb",
	}

	declareFlags(opts)

	// Establish a list of possible configuration file locations
	configLocations := []string{
		"happydomain.conf",
	}

	if home, dirErr := os.UserConfigDir(); dirErr == nil {
		configLocations = append(configLocations, path.Join(home, "happydomain", "happydomain.conf"))
	}

	configLocations = append(configLocations, path.Join("etc", "happydomain.conf"))

	// If config file exists, read configuration from it
	for _, filename := range configLocations {
		if _, e := os.Stat(filename); !os.IsNotExist(e) {
			log.Printf("Loading configuration from %s\n", filename)
			err = parseFile(opts, filename)
			if err != nil {
				return
			}
			break
		}
	}

	// Then, overwrite that by what is present in the environment
	err = parseEnvironmentVariables(opts)
	if err != nil {
		return
	}

	// Finaly, command line takes precedence
	err = parseCLI(opts)
	if err != nil {
		return
	}

	// Sanitize options
	if opts.BasePath != "/" {
		opts.BasePath = path.Clean(opts.BasePath)
	} else {
		opts.BasePath = ""
	}
	if opts.DevProxy != "" && opts.BasePath != "" {
		err = fmt.Errorf("-base-path is not supported in -dev mode")
		return
	}

	if opts.NoMail && opts.MailSMTPHost != "" {
		err = fmt.Errorf("-no-mail and -mail-smtp-* cannot be defined at the same time")
		return
	}

	if opts.ExternalURL.Host == "" || opts.ExternalURL.Scheme == "" {
		u, err2 := url.Parse("http://" + opts.ExternalURL.String())
		if err2 == nil {
			opts.ExternalURL = *u
		} else {
			err = fmt.Errorf("You defined an external URL without a scheme. The expected value is eg. http://localhost:8081")
			return
		}
	}
	if len(opts.ExternalURL.Path) > 1 {
		if opts.BasePath != "" && opts.BasePath != opts.ExternalURL.Path {
			err = fmt.Errorf("You defined both baseurl and a path to externalurl that are different. Define only one of those.")
			return
		}

		opts.BasePath = path.Clean(opts.ExternalURL.Path)
	}
	opts.ExternalURL.Path = ""
	opts.ExternalURL.Fragment = ""
	opts.ExternalURL.RawQuery = ""

	if len(opts.JWTSecretKey) == 0 {
		opts.JWTSecretKey = make([]byte, 32)
		_, err = rand.Read(opts.JWTSecretKey)
		if err != nil {
			return
		}
	}

	// Make the client IP trust posture explicit: it decides whether the
	// per-source rate limiters and the login lockout can be bypassed by
	// forging a X-Forwarded-For header.
	if len(opts.TrustedProxies) > 0 {
		log.Printf("Trusting client IP headers from: %s\n", strings.Join(opts.TrustedProxies, ", "))
	} else {
		log.Println("No trusted proxy configured: X-Forwarded-For and X-Real-IP headers are ignored, clients are identified by their socket address. If happyDomain runs behind a reverse proxy, declare it with -trusted-proxy (see docs/reverse-proxy.md).")
	}

	// Unlike the trusted proxies, silence here means the safe default, so only
	// the widened posture is worth a line. It is the one an operator needs to
	// see in the logs when a provider suddenly refuses to reach their LAN.
	if len(opts.OutboundAllowedTargets) > 0 {
		log.Printf("Requests made on behalf of users may reach public addresses, plus: %s\n", strings.Join(opts.OutboundAllowedTargets, ", "))
	}
	if len(opts.ResolverAllowedTargets) > 0 {
		log.Printf("The resolver tool may query public addresses, plus: %s\n", strings.Join(opts.ResolverAllowedTargets, ", "))
	}

	err = ExtendsConfigWithOIDC(opts)
	if err != nil {
		return
	}

	return
}

// parseLine treats a config line and place the read value in the variable
// declared to the corresponding flag.
func parseLine(_ *happydns.Options, line string) (err error) {
	fields := strings.SplitN(line, "=", 2)
	origKey := strings.TrimSpace(fields[0])
	value := strings.TrimSpace(fields[1])

	if len(value) == 0 {
		return
	}

	key := strings.TrimPrefix(strings.TrimPrefix(origKey, "HAPPYDNS_"), "HAPPYDOMAIN_")
	key = strings.ReplaceAll(key, "_", "-")
	key = strings.ToLower(key)

	err = flag.Set(key, value)

	return
}
