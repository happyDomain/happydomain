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
	BaseURL           string
	DevProxy          string
	DSN               string
	DefaultNameServer string
}

func ConsolidateConfig() (opts *Options, err error) {
	// Define defaults options
	opts = &Options{
		Bind:              ":8081",
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
