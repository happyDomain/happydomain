// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
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
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"git.happydns.org/happydns/storage"
)

type Options struct {
	Bind              string
	AdminBind         string
	ExternalURL       string
	BaseURL           string
	DevProxy          string
	DefaultNameServer string
	StorageEngine     storage.StorageEngine
}

func (o *Options) BuildURL(url string) string {
	return fmt.Sprintf("%s%s%s", o.ExternalURL, o.BaseURL, url)
}

func (o *Options) BuildURL_noescape(url string, args ...interface{}) string {
	args = append([]interface{}{o.ExternalURL, o.BaseURL}, args...)
	return fmt.Sprintf("%s%s"+url, args...)
}

func ConsolidateConfig() (opts *Options, err error) {
	// Define defaults options
	opts = &Options{
		Bind:              ":8081",
		AdminBind:         "./happydns.sock",
		ExternalURL:       "http://localhost:8081",
		BaseURL:           "/",
		DefaultNameServer: "127.0.0.1:53",
		StorageEngine:     storage.StorageEngine("leveldb"),
	}

	opts.declareFlags()

	// Establish a list of possible configuration file locations
	configLocations := []string{
		"happydns.conf",
	}

	if home, err := os.UserConfigDir(); err == nil {
		configLocations = append(configLocations, path.Join(home, "happydns", "happydns.conf"))
	}

	configLocations = append(configLocations, path.Join("etc", "happydns.conf"))

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

	return
}

func (o *Options) parseLine(line string) (err error) {
	fields := strings.SplitN(line, "=", 2)
	orig_key := strings.TrimSpace(fields[0])
	value := strings.TrimSpace(fields[1])

	key := strings.TrimPrefix(orig_key, "HAPPYDNS_")
	key = strings.Replace(key, "_", "-", -1)
	key = strings.ToLower(key)

	err = flag.Set(key, value)

	return
}
