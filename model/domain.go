package happydns

import (
	"strings"
)

type Domain struct {
	Id         int64 `json:"id"`
	IdUser     int64
	IdSource   int64
	DomainName string `json:"domain"`
}

type Domains []*Domain

func (d *Domain) NormalizedNSServer() string {
	if strings.Index(d.DomainName, ":") > -1 {
		return d.DomainName
	} else {
		return d.DomainName + ":53"
	}
}

func NewDomain(u *User, s Source, dn string) (d *Domain) {
	d = &Domain{
		IdUser: u.Id,
		//IdSource:   s.GetId(),
		DomainName: dn,
	}

	d.DomainName = d.NormalizedNSServer()

	return
}
