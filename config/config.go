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
	"net/url"
	"os"
	"path"
	"strings"

	"git.happydns.org/happyDomain/storage"
)

// Options stores the configuration of the software.
type Options struct {
	// Bind is the address:port used to bind the main interface with API.
	Bind string

	// AdminBind is the address:port or unix socket used to serve the admin
	// API.
	AdminBind string

	// ExternalURL keeps the URL used in communications (such as email,
	// ...), when it needs to use complete URL, not only relative parts.
	ExternalURL URL

	// BaseURL is the relative path where begins the root of the app.
	BaseURL string

	// DevProxy is the URL that override static assets.
	DevProxy string

	// DefaultNameServer is the NS server suggested by default.
	DefaultNameServer string

	// StorageEngine points to the storage engine used.
	StorageEngine storage.StorageEngine

	// NoAuth controls if there is user access control or not.
	NoAuth bool

	// ExternalAuth is the URL of the login form to use instead of the embedded one.
	ExternalAuth URL

	// JWTSecretKey stores the private key to sign and verify JWT tokens.
	JWTSecretKey JWTSecretKey
}

// BuildURL appends the given url to the absolute ExternalURL.
func (o *Options) BuildURL(url string) string {
	return fmt.Sprintf("%s%s%s", o.ExternalURL.URL.String(), o.BaseURL, url)
}

// BuildURL_noescape build an URL containing formater.
func (o *Options) BuildURL_noescape(url string, args ...interface{}) string {
	args = append([]interface{}{o.ExternalURL.URL.String(), o.BaseURL}, args...)
	return fmt.Sprintf("%s%s"+url, args...)
}

// ConsolidateConfig fills an Options struct by reading configuration from
// config files, environment, then command line.
//
// Should be called only one time.
func ConsolidateConfig() (opts *Options, err error) {
	u, _ := url.Parse("http://localhost:8081")

	// Define defaults options
	opts = &Options{
		Bind:              ":8081",
		AdminBind:         "./happydomain.sock",
		ExternalURL:       URL{URL: u},
		BaseURL:           "/",
		DefaultNameServer: "127.0.0.1:53",
		StorageEngine:     storage.StorageEngine("leveldb"),
	}

	opts.declareFlags()

	// Establish a list of possible configuration file locations
	configLocations := []string{
		"happydomain.conf",
	}

	if home, err := os.UserConfigDir(); err == nil {
		configLocations = append(configLocations, path.Join(home, "happydomain", "happydomain.conf"))
	}

	configLocations = append(configLocations, path.Join("etc", "happydomain.conf"))

	// If config file exists, read configuration from it
	for _, filename := range configLocations {
		if _, e := os.Stat(filename); !os.IsNotExist(e) {
			log.Printf("Loading configuration from %s\n", filename)
			err = opts.parseFile(filename)
			if err != nil {
				return
			}
			break
		}
	}

	// Then, overwrite that by what is present in the environment
	err = opts.parseEnvironmentVariables()
	if err != nil {
		return
	}

	// Finaly, command line takes precedence
	err = opts.parseCLI()
	if err != nil {
		return
	}

	// Sanitize options
	if opts.BaseURL != "/" {
		opts.BaseURL = path.Clean(opts.BaseURL)
	} else {
		opts.BaseURL = ""
	}

	if opts.ExternalURL.URL.Host == "" || opts.ExternalURL.URL.Scheme == "" {
		u, err2 := url.Parse("http://" + opts.ExternalURL.URL.String())
		if err2 == nil {
			opts.ExternalURL.URL = u
		} else {
			err = fmt.Errorf("You defined an external URL without a scheme. The expected value is eg. http://localhost:8081")
			return
		}
	}
	if len(opts.ExternalURL.URL.Path) > 1 {
		if opts.BaseURL != "" && opts.BaseURL != opts.ExternalURL.URL.Path {
			err = fmt.Errorf("You defined both baseurl and a path to externalurl that are different. Define only one of those.")
			return
		}

		opts.BaseURL = path.Clean(opts.ExternalURL.URL.Path)
	}
	opts.ExternalURL.URL.Path = ""
	opts.ExternalURL.URL.Fragment = ""
	opts.ExternalURL.URL.RawQuery = ""

	if len(opts.JWTSecretKey) == 0 {
		opts.JWTSecretKey = make([]byte, 32)
		_, err = rand.Read(opts.JWTSecretKey)
		if err != nil {
			return
		}
	}

	return
}

// parseLine treats a config line and place the read value in the variable
// declared to the corresponding flag.
func (o *Options) parseLine(line string) (err error) {
	fields := strings.SplitN(line, "=", 2)
	orig_key := strings.TrimSpace(fields[0])
	value := strings.TrimSpace(fields[1])

	if len(value) == 0 {
		return
	}

	key := strings.TrimPrefix(strings.TrimPrefix(orig_key, "HAPPYDNS_"), "HAPPYDOMAIN_")
	key = strings.Replace(key, "_", "-", -1)
	key = strings.ToLower(key)

	err = flag.Set(key, value)

	return
}
