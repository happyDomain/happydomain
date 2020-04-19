package config // import "happydns.org/config"

import (
	"fmt"
	"os"
	"strings"
)

func (o *Options) parseEnvironmentVariables() (err error) {
	for _, line := range os.Environ() {
		if strings.HasPrefix(line, "HAPPYDNS_") {
			err := o.parseLine(line)
			if err != nil {
				return fmt.Errorf("error in environment (%q): %w", line, err)
			}
		}
	}
	return
}
