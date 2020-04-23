package sources // import "happydns.org/sources"

import (
	"fmt"
	"log"

	"git.happydns.org/happydns/model"
)

type SourceCreator func() happydns.Source

type Source struct {
	Creator SourceCreator
	Infos   SourceInfos
}

var sources map[string]Source = map[string]Source{}

func RegisterSource(name string, creator SourceCreator, infos SourceInfos) {
	log.Println("Registering new source:", name)
	sources[name] = Source{
		creator,
		infos,
	}
}

func GetSources() *map[string]Source {
	return &sources
}

func FindSource(name string) (happydns.Source, error) {
	src, ok := sources[name]
	if !ok {
		return nil, fmt.Errorf("Unable to find corresponding source for `%s`.", name)
	}

	return src.Creator(), nil
}
