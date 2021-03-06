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
	"fmt"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happydomain/config"
	"git.happydns.org/happydomain/forms"
	"git.happydns.org/happydomain/model"
)

type FormState struct {
	Id       interface{} `json:"_id,omitempty"`
	Name     string      `json:"_comment"`
	State    int32       `json:"state"`
	Recall   *int64      `json:"recall,omitempty"`
	Redirect *string     `json:"redirect,omitempty"`
}

type FormResponse struct {
	From     *forms.CustomForm `json:"form,omitempty"`
	Redirect *string           `json:"redirect,omitempty"`
}

func formDoState(cfg *config.Options, c *gin.Context, fs *FormState, data interface{}, defaultForm func(interface{}) *forms.CustomForm) (form *forms.CustomForm, err error) {
	session := c.MustGet("MySession").(*happydns.Session)

	if fs.Recall != nil {
		session.GetValue(fmt.Sprintf("form-%d", *fs.Recall), data)
		session.GetValue(fmt.Sprintf("form-%d-name", *fs.Recall), &fs.Name)
		session.GetValue(fmt.Sprintf("form-%d-id", *fs.Recall), &fs.Id)
		session.GetValue(fmt.Sprintf("form-%d-next", *fs.Recall), &fs.Redirect)
	}

	csf, ok := data.(forms.CustomSettingsForm)
	if !ok {
		if fs.State == 1 {
			err = forms.DoneForm
		} else {
			form = defaultForm(data)
		}
		return
	} else {
		return csf.DisplaySettingsForm(fs.State, cfg, session, func() int64 {
			key, recallid := session.FindNewKey("form-")
			session.SetValue(key, data)
			session.SetValue(key+"-id", fs.Id)
			if fs.Redirect != nil {
				session.SetValue(key+"-next", *fs.Redirect)
			}
			return recallid
		})
	}
}
