// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
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

package happydns

import (
	"github.com/miekg/dns"
)

// DomainMinimal is used for swagger documentation as Domain add.
type DomainMinimal struct {
	// IsProvider is the identifier of the Provider used to access and edit the
	// Domain.
	IdProvider Identifier `json:"id_provider" swaggertype:"string"`

	// DomainName is the FQDN of the managed Domain.
	DomainName string `json:"domain"`
}

// Domain holds information about a domain name own by a User.
type Domain struct {
	// Id is the Domain's identifier in the database.
	Id Identifier `json:"id" swaggertype:"string"`

	// IdUser is the identifier of the Domain's Owner.
	IdUser Identifier `json:"id_owner" swaggertype:"string"`

	// IsProvider is the identifier of the Provider used to access and edit the
	// Domain.
	IdProvider Identifier `json:"id_provider" swaggertype:"string"`

	// DomainName is the FQDN of the managed Domain.
	DomainName string `json:"domain"`

	// Group is a hint string aims to group domains.
	Group string `json:"group,omitempty"`

	// ZoneHistory are the identifiers to the Zone attached to the current
	// Domain.
	ZoneHistory []Identifier `json:"zone_history" swaggertype:"array,string"`
}

// Domains is an array of Domain.
type Domains []*Domain

// HasZone checks if the given Zone's identifier is part of this Domain
// history.
func (d *Domain) HasZone(zoneId Identifier) (found bool) {
	for _, v := range d.ZoneHistory {
		if v.Equals(zoneId) {
			return true
		}
	}
	return
}

// NewDomain fills a new Domain structure.
func NewDomain(u *User, st *ProviderMeta, dn string) (d *Domain) {
	d = &Domain{
		IdUser:     u.Id,
		IdProvider: st.Id,
		DomainName: dns.Fqdn(dn),
	}

	return
}
