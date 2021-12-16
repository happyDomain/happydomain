// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
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

// Source is where Domains and Zones can be managed.
type Source interface {
	// Validate tells if the Source's settings are good.
	Validate() error

	// DomainExists tells if the given domain exists for the Source.
	DomainExists(string) error

	// ImportZone retrieves all RRs for the given Domain.
	ImportZone(*Domain) ([]dns.RR, error)

	// AddRR adds an RR in the zone of the given Domain.
	AddRR(*Domain, dns.RR) error

	// DeleteRR removes the given RR in the zone of the given Domain.
	DeleteRR(*Domain, dns.RR) error

	// UpdateSOA tries to update the Zone's SOA record, according to the
	// given parameters.
	UpdateSOA(*Domain, *dns.SOA, bool) error
}

// SourceMeta holds the metadata associated to a Source.
type SourceMeta struct {
	// Type is the string representation of the Source's type.
	Type string `json:"_srctype"`

	// Id is the Source's identifier.
	Id int64 `json:"_id"`

	// OwnerId is the User's identifier for the current Source.
	OwnerId []byte `json:"_ownerid"`

	// Comment is a string that helps user to distinguish the Source.
	Comment string `json:"_comment,omitempty"`
}

// SourceCombined combined SourceMeta + Source
type SourceCombined struct {
	Source
	SourceMeta
}
