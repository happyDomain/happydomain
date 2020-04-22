package happydns

import (
	"github.com/miekg/dns"
)

type Source interface {
	Validate() error
	ImportZone(*Domain) ([]dns.RR, error)
	AddRR(*Domain, dns.RR) error
	DeleteRR(*Domain, dns.RR) error
}

type SourceType struct {
	Type    string `json:"_srctype"`
	Id      int64  `json:"_id"`
	OwnerId int64  `json:"_ownerid"`
	Comment string `json:"_comment,omitempty"`
}

type SourceCombined struct {
	Source
	SourceType
}
