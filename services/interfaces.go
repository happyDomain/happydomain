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

package svcs

import (
	"encoding/json"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

type ServiceRestrictions struct {
	// Alone restricts the service to be the only one for a given subdomain.
	Alone bool `json:"alone,omitempty"`

	// ExclusiveRR restricts the service to be present along with others given types.
	ExclusiveRR []string `json:"exclusive,omitempty"`

	// GLUE allows a service to be present under Leaf, as GLUE record.
	GLUE bool `json:"glue,omitempty"`

	// Leaf restricts the creation of subdomains under this kind of service (blocks NearAlone).
	Leaf bool `json:"leaf,omitempty"`

	// NearAlone allows a service to be present along with Alone restricted services (eg. services that will create sub-subdomain from their given subdomain).
	NearAlone bool `json:"nearAlone,omitempty"`

	// NeedTypes restricts the service to sources that are compatibles with ALL the given types.
	NeedTypes []uint16 `json:"needTypes,omitempty"`

	// RootOnly restricts the service to be present at the root of the domain only.
	RootOnly bool `json:"rootOnly,omitempty"`

	// Single restricts the service to be present only once per subdomain.
	Single bool `json:"single,omitempty"`
}

type ServiceInfos struct {
	Name         string              `json:"name"`
	Type         string              `json:"_svctype"`
	Icon         string              `json:"_svcicon,omitempty"`
	Description  string              `json:"description"`
	Family       string              `json:"family"`
	Categories   []string            `json:"categories"`
	Tabs         bool                `json:"tabs,omitempty"`
	Restrictions ServiceRestrictions `json:"restrictions,omitempty"`
}

type serviceCombined struct {
	Service happydns.Service
}

type ServiceNotFoundError struct {
	name string
}

func (err ServiceNotFoundError) Error() string {
	return fmt.Sprintf("Unable to find corresponding service for `%s`.", err.name)
}

// UnmarshalServiceJSON implements the UnmarshalJSON function for the
// encoding/json module.
func UnmarshalServiceJSON(svc *happydns.ServiceCombined, b []byte) (err error) {
	var svcType happydns.ServiceMeta
	err = json.Unmarshal(b, &svcType)
	if err != nil {
		return
	}

	var tsvc happydns.Service
	tsvc, err = FindService(svcType.Type)
	if err != nil {
		return
	}

	mySvc := &serviceCombined{
		tsvc,
	}

	err = json.Unmarshal(b, mySvc)

	svc.Service = tsvc
	svc.ServiceMeta = svcType

	return
}

func init() {
	// Register the UnmarshalServiceJSON variable thats points to the
	// Service's UnmarshalJSON implementation that can't be made in model
	// module due to cyclic dependancy.
	happydns.UnmarshalServiceJSON = UnmarshalServiceJSON
}
