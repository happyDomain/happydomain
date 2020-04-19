package config // import "happydns.org/config"

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func (o *Options) parseFile(filename string) error {
	fp, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)
	n := 0
	for scanner.Scan() {
		n += 1
		line := strings.TrimSpace(scanner.Text())
		if len(line) > 0 && !strings.HasPrefix(line, "#") && strings.Index(line, "=") > 0 {
			err := o.parseLine(line)
			if err != nil {
				return fmt.Errorf("%v:%d: error in configuration: %w", filename, n, err)
			}
		}
	}

	return nil
}
