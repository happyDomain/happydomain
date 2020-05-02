package sources // import "happydns.org/sources"

import ()

type SourceInfos struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ListDomainsSource interface {
	ListDomains() ([]string, error)
}
