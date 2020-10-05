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

// CustomForm is used to create a form with several steps when creating or updating source's settings.
type CustomForm struct {
	// BeforeText is the text presented before the fields.
	BeforeText string `json:"beforeText,omitempty"`

	// SideText is displayed in the sidebar, after any already existing text. When a sidebar is avaiable.
	SideText string `json:"sideText,omitempty"`

	// AfterText is the text presented after the fields and before the buttons
	AfterText string `json:"afterText,omitempty"`

	// Fields are the fields presented to the User.
	Fields []*Field `json:"fields"`

	// NextButtonText is the next button content.
	NextButtonText string `json:"nextButtonText,omitempty"`

	// NextEditButtonText is the next button content when updating the settings (if not set, NextButtonText is used instead).
	NextEditButtonText string `json:"nextEditButtonText,omitempty"`

	// PreviousButtonText is previous/cancel button content.
	PreviousButtonText string `json:"previousButtonText,omitempty"`

	// PreviousEditButtonText is the previous/cancel button content when updating the settings (if not set, NextButtonText is used instead).
	PreviousEditButtonText string `json:"previousEditButtonText,omitempty"`

	// NextButtonLink is the target of the next button, exclusive with NextButtonState.
	NextButtonLink string `json:"nextButtonLink,omitempty"`

	// NextButtonState is the step number asked when submiting the form.
	NextButtonState int32 `json:"nextButtonState,omitempty"`

	// PreviousButtonState is the step number to go when hitting the previous button.
	PreviousButtonState int32 `json:"previousButtonState,omitempty"`
}
