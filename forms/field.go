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

package forms // import "happydns.org/forms"

import ()

// Field
type Field struct {
	// Id is the field identifier.
	Id string `json:"id"`

	// Type is the string representation of the field's type.
	Type string `json:"type"`

	// Label is the title given to the field, displayed as <label> tag on the interface.
	Label string `json:"label,omitempty"`

	// Placeholder is the placeholder attribute of the corresponding <input> tag.
	Placeholder string `json:"placeholder,omitempty"`

	// Default is the preselected value or the default value in case the field is not filled by the user.
	Default string `json:"default,omitempty"`

	// Choices holds the differents choices shown to the user in <select> tag.
	Choices []string `json:"choices,omitempty"`

	// Required indicates whether the field has to be filled or not.
	Required bool `json:"required,omitempty"`

	// Secret indicates if the field contains sensitive information such as API key, in order to hide
	// the field when not needed. When typing, it doesn't hide characters like in password input.
	Secret bool `json:"secret,omitempty"`

	// Description stores an helpfull sentence describing the field.
	Description string `json:"description,omitempty"`
}
