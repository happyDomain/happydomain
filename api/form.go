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

package api

import (
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/forms"
	"git.happydns.org/happyDomain/model"
)

type FormState struct {
	// Id for an already existing element.
	Id *happydns.Identifier `json:"_id,omitempty" swaggertype:"string"`

	// User defined name of the element.
	Name string `json:"_comment"`

	// State is the desired form to shows next (starting at 0).
	State int32 `json:"state"`

	// Recall is the identifier for a saved FormState you want to retrieve.
	Recall string `json:"recall,omitempty"`
}

func formDoState(cfg *config.Options, c *gin.Context, fs *FormState, data interface{}, defaultForm func(interface{}) *forms.CustomForm) (form *forms.CustomForm, d map[string]interface{}, err error) {
	session := c.MustGet("MySession").(*happydns.Session)

	csf, ok := data.(forms.CustomSettingsForm)
	if !ok {
		if fs.State == 1 {
			err = forms.DoneForm
		} else {
			form = defaultForm(data)
		}
		return
	} else {
		return csf.DisplaySettingsForm(fs.State, cfg, session, func() string {
			return fs.Recall
		})
	}
}
