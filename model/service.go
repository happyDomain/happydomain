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

// Service represents a service provided by one or more DNS record.
type Service interface {
	// GetNbResources get the number of main Resources contains in the Service.
	GetNbResources() int

	// GenComment sum up the content of the Service, in a small usefull string.
	GenComment(origin string) string

	// genRRs generates corresponding RRs.
	GenRRs(domain string, ttl uint32, origin string) []dns.RR
}

// ServiceMeta holds the metadata associated to a Service.
type ServiceMeta struct {
	// Type is the string representation of the Service's type.
	Type string `json:"_svctype"`

	// Id is the Service's identifier.
	Id []byte `json:"_id,omitempty"`

	// OwnerId is the User's identifier for the current Service.
	OwnerId int64 `json:"_ownerid,omitempty"`

	// Domain contains the abstract domain where this Service relates.
	Domain string `json:"_domain"`

	// Ttl contains the specific TTL for the underlying Resources.
	Ttl uint32 `json:"_ttl"`

	// Comment is a string that helps user to distinguish the Service.
	Comment string `json:"_comment,omitempty"`

	// UserComment is a supplementary string defined by the user to
	// distinguish the Service.
	UserComment string `json:"_mycomment,omitempty"`

	// Aliases exposes the aliases defined on this Service.
	Aliases []string `json:"_aliases,omitempty"`

	// NbResources holds the number of Resources stored inside this Service.
	NbResources int `json:"_tmp_hint_nb"`
}

// ServiceCombined combined ServiceMeta + Service
type ServiceCombined struct {
	Service
	ServiceMeta
}

// UnmarshalServiceJSON stores a functor defined in services/interfaces.go that
// can't be defined here due to cyclic imports.
var UnmarshalServiceJSON func(*ServiceCombined, []byte) error

// UnmarshalJSON points to the implementation of the UnmarshalJSON function for
// the encoding/json module.
func (svc *ServiceCombined) UnmarshalJSON(b []byte) error {
	return UnmarshalServiceJSON(svc, b)
}
