// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package config // import "happydns.org/config"

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
	return fmt.Sprintf("%s%s%s", o.ExternalURL, o.BaseURL, url)
}

// BuildURL_noescape build an URL containing formater.
func (o *Options) BuildURL_noescape(url string, args ...interface{}) string {
	args = append([]interface{}{o.ExternalURL, o.BaseURL}, args...)
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
