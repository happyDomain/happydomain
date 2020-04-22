package sources // import "happydns.org/sources"

import (
	"fmt"
	"log"

	"git.happydns.org/happydns/model"
)

type SourceCreator func() happydns.Source

var sources map[string]SourceCreator = map[string]SourceCreator{}

func RegisterSource(name string, creator SourceCreator) {
	log.Println("Registering new source:", name)
	sources[name] = creator
}

func FindSource(name string) (happydns.Source, error) {
	src, ok := sources[name]
	if !ok {
		return nil, fmt.Errorf("Unable to find corresponding source for `%s`.", name)
	}

	return src(), nil
}
