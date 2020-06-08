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
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"git.happydns.org/happydns/storage/mysql"
)

type Options struct {
	Bind              string
	AdminBind         string
	ExternalURL       string
	BaseURL           string
	DevProxy          string
	DSN               string
	DefaultNameServer string
}

func ConsolidateConfig() (opts *Options, err error) {
	// Define defaults options
	opts = &Options{
		Bind:              ":8081",
		AdminBind:         "./happydns.sock",
		ExternalURL:       "http://localhost:8081",
		BaseURL:           "/",
		DSN:               database.DSNGenerator(),
		DefaultNameServer: "127.0.0.1:53",
	}

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
	key := strings.TrimSpace(fields[0])
	value := strings.TrimSpace(fields[1])

	key = strings.TrimPrefix(key, "HAPPYDNS_")
	key = strings.Replace(key, "_", "", -1)
	key = strings.ToUpper(key)

	switch key {
	case "DEVPROXY":
		err = parseString(&o.DevProxy, value)
	}

	return
}

func parseString(store *string, value string) error {
	*store = value
	return nil
}

func parseBool(store *bool, value string) error {
	value = strings.ToLower(value)

	if value == "1" || value == "yes" || value == "true" || value == "on" {
		*store = true
	} else if value == "" || value == "0" || value == "no" || value == "false" || value == "off" {
		*store = false
	} else {
		return fmt.Errorf("%s is not a valid bool value", value)
	}

	return nil
}
