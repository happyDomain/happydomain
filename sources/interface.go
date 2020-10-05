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

package sources // import "happydns.org/sources"

import (
	"git.happydns.org/happydns/forms"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/utils"

	"github.com/miekg/dns"
)

// SourceInfos describes the purpose of a user usable source.
type SourceInfos struct {
	// Name is the name displayed.
	Name string `json:"name"`

	// Description is a brief description of what the source is.
	Description string `json:"description"`
}

// ListDomainsSource are functions to declare when we can retrives a list of managable domains from the given Source.
type ListDomainsSource interface {
	// ListDomains retrieves the list of avaiable domains inside the Source.
	ListDomains() ([]string, error)
}

type LimitedResourceTypesSource interface {
	ListAvailableTypes() []uint16
}

var DefaultAvailableTypes []uint16

// GetSourceCapabilities lists available capabilities for the given Source.
func GetSourceCapabilities(src happydns.Source) (caps []string) {
	if _, ok := src.(ListDomainsSource); ok {
		caps = append(caps, "ListDomains")
	}

	if _, ok := src.(forms.CustomSettingsForm); ok {
		caps = append(caps, "CustomSettingsForm")
	}

	if _, ok := src.(LimitedResourceTypesSource); ok {
		caps = append(caps, "LimitedResourceTypes")
	}

	return
}

// GenDefaultSettingsForm generates a generic CustomForm presenting all the fields in one page.
func GenDefaultSettingsForm(src happydns.Source) *forms.CustomForm {
	return &forms.CustomForm{
		Fields:                 GenSourceFields(src),
		NextButtonText:         "Create",
		NextEditButtonText:     "Update",
		NextButtonState:        1,
		PreviousButtonText:     "Use another source",
		PreviousEditButtonText: "Cancel",
		PreviousButtonState:    -1,
	}
}

func init() {
	for t := range dns.TypeToRR {
		if !utils.IsDNSSECType(t) {
			DefaultAvailableTypes = append(DefaultAvailableTypes, t)
		}
	}
}
