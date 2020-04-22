package happydns

import (
	"strings"
)

type Domain struct {
	Id         int64  `json:"id"`
	IdUser     int64  `json:"id_owner"`
	IdSource   int64  `json:"id_source"`
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

func NewDomain(u *User, st *SourceType, dn string) (d *Domain) {
	d = &Domain{
		IdUser:     u.Id,
		IdSource:   st.Id,
		DomainName: dn,
	}

	d.DomainName = d.NormalizedNSServer()

	return
}
